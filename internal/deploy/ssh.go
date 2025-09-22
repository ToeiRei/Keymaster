package deploy

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"runtime"
	"strings"
	"time"

	"github.com/Microsoft/go-winio"
	"github.com/davidmz/go-pageant"
	"github.com/pkg/sftp"
	"github.com/toeirei/keymaster/internal/db"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// Deployer handles the connection and deployment to a remote host.
type Deployer struct {
	client *ssh.Client
	sftp   *sftp.Client
}

// NewDeployer creates a new SSH connection and returns a Deployer.
func NewDeployer(host, user, privateKey string) (*Deployer, error) {
	signer, err := ssh.ParsePrivateKey([]byte(privateKey))
	if err != nil {
		return nil, fmt.Errorf("unable to parse private key: %w", err)
	}

	// Build our list of authentication methods.
	var authMethods []ssh.AuthMethod

	// 1. Try to use a running SSH agent. This is great for bootstrapping.
	var agentClient agent.Agent
	if runtime.GOOS == "windows" {
		// On Windows, try to connect to a Pageant-like agent first (like gpg-agent or putty's pageant).
		// This is a blocking call, but it's very fast if the agent isn't running.
		if pageant.Available() {
			agentClient = pageant.New()
		}
	}

	// If Pageant isn't available or we're not on Windows, try the OpenSSH agent socket/pipe.
	if agentClient == nil {
		var agentConn net.Conn
		var agentErr error
		if sshAgentSocket := os.Getenv("SSH_AUTH_SOCK"); sshAgentSocket != "" {
			// Use the socket defined in the environment variable.
			if runtime.GOOS == "windows" {
				agentConn, agentErr = winio.DialPipe(sshAgentSocket, nil)
			} else {
				agentConn, agentErr = net.Dial("unix", sshAgentSocket)
			}
		} else if runtime.GOOS == "windows" {
			// If no env var, try the default OpenSSH for Windows named pipe as a fallback.
			agentConn, agentErr = winio.DialPipe(`\\.\pipe\openssh-ssh-agent`, nil)
		}

		if agentErr == nil && agentConn != nil {
			agentClient = agent.NewClient(agentConn)
		}
	}

	// If we successfully connected to any agent, add its signers.
	if agentClient != nil {
		authMethods = append(authMethods, ssh.PublicKeysCallback(agentClient.Signers))
	}

	// 2. Always include the specific Keymaster system key as a reliable fallback.
	authMethods = append(authMethods, ssh.PublicKeys(signer))

	config := &ssh.ClientConfig{
		User: user,
		// The client will try each auth method in order (agent, then key).
		Auth: authMethods,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			// The hostname passed to the callback can include the port. We need to strip it
			// to ensure we're looking up the correct key in our database.
			host, _, err := net.SplitHostPort(hostname)
			if err != nil {
				// If SplitHostPort fails, it means there was no port, so we use the original string.
				host = hostname
			}

			// The key is presented in the format "ssh-ed25519 AAA..."
			presentedKey := string(ssh.MarshalAuthorizedKey(key))

			// Check if we have a trusted key for this host in our database.
			knownKey, err := db.GetKnownHostKey(host)
			if err != nil {
				return fmt.Errorf("failed to query known_hosts database: %w", err)
			}

			// If we don't have a key, this is the first connection.
			if knownKey == "" {
				return fmt.Errorf("unknown host key for %s. run 'keymaster trust-host' to add it", host)
			}

			// If the key exists, it must match exactly.
			if knownKey != presentedKey {
				return fmt.Errorf("!!! HOST KEY MISMATCH FOR %s !!!\nRemote key presented: %s\nThis could be a man-in-the-middle attack.", host, presentedKey)
			}

			return nil // Host key is trusted.
		},
		Timeout: 10 * time.Second,
	}

	// Add port 22 if not specified.
	addr := host
	if _, _, err := net.SplitHostPort(host); err != nil {
		addr = net.JoinHostPort(host, "22")
	}

	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("failed to dial: %w", err)
	}

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to create sftp client: %w", err)
	}

	return &Deployer{
		client: client,
		sftp:   sftpClient,
	}, nil
}

// DeployAuthorizedKeys uploads the new authorized_keys content and moves it into place.
func (d *Deployer) DeployAuthorizedKeys(content string) error {
	// 1. Ensure .ssh directory exists with correct permissions.
	_ = d.sftp.Mkdir(".ssh") // Ignore error if it already exists.
	if err := d.sftp.Chmod(".ssh", 0700); err != nil {
		return fmt.Errorf("failed to chmod .ssh directory: %w", err)
	}

	// 2. Upload to a temporary file.
	tmpPath := path.Join("/tmp", fmt.Sprintf("authorized_keys.keymaster.%d", time.Now().UnixNano()))
	f, err := d.sftp.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create temporary file on remote: %w", err)
	}
	if _, err := f.Write([]byte(content)); err != nil {
		f.Close()
		return fmt.Errorf("failed to write to temporary file on remote: %w", err)
	}
	f.Close()

	// 3. Move the file into place atomically and set permissions.
	finalPath := ".ssh/authorized_keys"
	cmd := fmt.Sprintf("mv %s %s && chmod 600 %s", tmpPath, finalPath, finalPath)

	session, err := d.client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session for mv command: %w", err)
	}
	defer session.Close()

	var stderr bytes.Buffer
	session.Stderr = &stderr
	if err := session.Run(cmd); err != nil {
		return fmt.Errorf("failed to move authorized_keys file (err: %v, stderr: %s)", err, stderr.String())
	}

	return nil
}

// Close closes the underlying SSH and SFTP clients.
func (d *Deployer) Close() {
	if d.sftp != nil {
		d.sftp.Close()
	}
	if d.client != nil {
		d.client.Close()
	}
}

// GetAuthorizedKeys reads and returns the content of the remote authorized_keys file.
func (d *Deployer) GetAuthorizedKeys() ([]byte, error) {
	finalPath := ".ssh/authorized_keys"
	f, err := d.sftp.Open(finalPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open remote file %s: %w", finalPath, err)
	}
	defer f.Close()

	content, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read from remote file %s: %w", finalPath, err)
	}
	return content, nil
}

// GetRemoteHostKey connects to a host just to retrieve its public key.
func GetRemoteHostKey(host string) (ssh.PublicKey, error) {
	keyChan := make(chan ssh.PublicKey, 1)

	config := &ssh.ClientConfig{
		// We don't need to authenticate for this, just start the handshake.
		User: "keymaster-probe",
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			// We got the key, send it back on the channel.
			keyChan <- key
			// Return a specific error to gracefully stop the handshake.
			return fmt.Errorf("keymaster: successfully retrieved host key")
		},
		Timeout: 5 * time.Second,
	}

	addr := host
	if _, _, err := net.SplitHostPort(host); err != nil {
		addr = net.JoinHostPort(host, "22")
	}

	// We expect ssh.Dial to fail with our specific error.
	_, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		// Check if it's our specific error.
		if strings.Contains(err.Error(), "keymaster: successfully retrieved host key") {
			// Success, the key is in the channel.
			return <-keyChan, nil
		}
		// It's a different, real error (e.g., connection refused).
		return nil, fmt.Errorf("failed to connect to %s: %w", host, err)
	}

	// This case should ideally not be reached if the callback returns an error.
	return nil, fmt.Errorf("ssh.Dial succeeded unexpectedly, could not retrieve key")
}
