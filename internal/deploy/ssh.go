package deploy

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"path"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"github.com/toeirei/keymaster/internal/db"
	"golang.org/x/crypto/ssh"
)

// Deployer handles the connection and deployment to a remote host.
type Deployer struct {
	client *ssh.Client
	sftp   *sftp.Client
}

// NewDeployer creates a new SSH connection and returns a Deployer.
func NewDeployer(host, user, privateKey string) (*Deployer, error) {
	// Define the host key callback once to be reused.
	hostKeyCallback := func(hostname string, remote net.Addr, key ssh.PublicKey) error {
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
			return fmt.Errorf("!!! HOST KEY MISMATCH FOR %s !!!\nRemote key presented: %s\nThis could be a man-in-the-middle attack", host, presentedKey)
		}

		return nil // Host key is trusted.
	}

	// Add port 22 if not specified.
	addr := host
	if _, _, err := net.SplitHostPort(host); err != nil {
		addr = net.JoinHostPort(host, "22")
	}
	var client *ssh.Client
	var finalErr error

	// --- Attempt 1: Use the Keymaster system key exclusively ---
	if privateKey != "" {
		signer, err := ssh.ParsePrivateKey([]byte(privateKey))
		if err != nil {
			return nil, fmt.Errorf("unable to parse private key: %w", err)
		}

		config := &ssh.ClientConfig{
			User:            user,
			Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
			HostKeyCallback: hostKeyCallback,
			Timeout:         10 * time.Second,
		}

		client, err = ssh.Dial("tcp", addr, config)
		if err == nil {
			// Success! We connected with the system key.
			sftpClient, sftpErr := sftp.NewClient(client)
			if sftpErr != nil {
				client.Close()
				return nil, fmt.Errorf("failed to create sftp client: %w", sftpErr)
			}
			return &Deployer{client: client, sftp: sftpClient}, nil
		}

		// If the error was not an auth failure, we should fail fast.
		if !strings.Contains(err.Error(), "unable to authenticate") {
			return nil, fmt.Errorf("connection with system key failed: %w", err)
		}
		// It was an auth error, so we'll store it and try the agent.
		finalErr = err
	}

	// --- Attempt 2: Use the SSH agent as a fallback ---
	agentClient := getSSHAgent()
	if agentClient == nil {
		if finalErr != nil { // This means the private key auth failed before this.
			return nil, fmt.Errorf("system key authentication failed, and no SSH agent available for fallback: %w", finalErr)
		}
		return nil, fmt.Errorf("no authentication method available (no system key provided and no ssh agent found)")
	}

	config := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.PublicKeysCallback(agentClient.Signers)},
		HostKeyCallback: hostKeyCallback,
		Timeout:         10 * time.Second,
	}

	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("connection with ssh agent failed: %w", err)
	}

	// Success with agent.

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
