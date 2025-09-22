package deploy

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"path"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
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

	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		// In a production tool, you would implement a proper host key verification
		// (e.g., checking against a known_hosts file).
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
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
