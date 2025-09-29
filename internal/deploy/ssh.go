// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// package deploy provides functionality for connecting to remote hosts via SSH
// and managing their authorized_keys files. This file contains the core SSH and
// SFTP client logic for connecting, authenticating, and transferring files.
package deploy // import "github.com/toeirei/keymaster/internal/deploy"

import (
	"errors"
	"fmt"
	"io"
	"net"
	"path"
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
// For bootstrap connections, use NewBootstrapDeployer instead.
func NewDeployer(host, user, privateKey string) (*Deployer, error) {
	return newDeployerInternal(host, user, privateKey, false)
}

// NewBootstrapDeployer creates a new SSH connection for bootstrap operations.
// It accepts any host key and saves it to the database for future connections.
func NewBootstrapDeployer(host, user, privateKey string) (*Deployer, error) {
	return newDeployerInternal(host, user, privateKey, true)
}

// newDeployerInternal is the internal implementation for creating deployers.
func newDeployerInternal(host, user, privateKey string, isBootstrap bool) (*Deployer, error) {
	// Define the host key callback based on bootstrap mode.
	var hostKeyCallback ssh.HostKeyCallback

	if isBootstrap {
		// For bootstrap, accept any host key and save it
		hostKeyCallback = func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			// Strip port if present
			hostOnly, _, err := net.SplitHostPort(hostname)
			if err != nil {
				hostOnly = hostname
			}

			// Save the host key for future connections
			presentedKey := string(ssh.MarshalAuthorizedKey(key))
			if err := db.AddKnownHostKey(hostOnly, presentedKey); err != nil {
				// Log error but don't fail the connection
				// The host key can be added manually later
			}

			return nil // Accept the key for bootstrap
		}
	} else {
		// Normal mode: verify host keys
		hostKeyCallback = func(hostname string, remote net.Addr, key ssh.PublicKey) error {
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
	}

	// Add port 22 if not specified.
	addr := host
	if _, _, err := net.SplitHostPort(host); err != nil {
		addr = net.JoinHostPort(host, "22")
	}
	var client *ssh.Client

	// If a private key is provided, use it exclusively. This is the standard path
	// for deployment and auditing with a Keymaster system key.
	if privateKey != "" {
		signer, err := ssh.ParsePrivateKey([]byte(privateKey))
		if err == nil {
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
			// If we provided a key and it failed, we will fall through to try the agent.
		}
	}

	// If no private key was provided, attempt to use the SSH agent.
	// This is used for bootstrapping/importing keys.
	agentClient := getSSHAgent()
	if agentClient == nil {
		return nil, fmt.Errorf("no authentication method available (system key failed and no ssh agent found)")
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
// This function uses a pure-SFTP method to be compatible with restricted keys
// (e.g., command="internal-sftp"). It uses a backup-and-rename strategy for
// compatibility with SFTP servers that don't support atomic overwrites (e.g., on Windows).
func (d *Deployer) DeployAuthorizedKeys(content string) error {
	// 1. Ensure .ssh directory exists with correct permissions.
	const sshDir = ".ssh"
	if _, err := d.sftp.Stat(sshDir); err != nil {
		// If the directory doesn't exist, create it.
		if err := d.sftp.Mkdir(sshDir); err != nil {
			return fmt.Errorf("failed to create .ssh directory: %w", err)
		}
	}
	if err := d.sftp.Chmod(sshDir, 0700); err != nil {
		return fmt.Errorf("failed to chmod .ssh directory: %w", err)
	}

	// 2. Upload to a temporary file within the .ssh directory for atomic rename.
	tmpPath := path.Join(sshDir, fmt.Sprintf("authorized_keys.keymaster.%d", time.Now().UnixNano()))
	f, err := d.sftp.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create temporary file on remote: %w", err)
	}
	if _, err := f.Write([]byte(content)); err != nil {
		f.Close()
		// Best effort to clean up the failed upload
		_ = d.sftp.Remove(tmpPath)
		return fmt.Errorf("failed to write to temporary file on remote: %w", err)
	}
	f.Close()

	// 3. Set permissions on the temporary file before moving.
	if err := d.sftp.Chmod(tmpPath, 0600); err != nil {
		_ = d.sftp.Remove(tmpPath)
		return fmt.Errorf("failed to chmod temporary file: %w", err)
	}

	// 4. Move the file into place using a backup-and-rename strategy.
	finalPath := path.Join(sshDir, "authorized_keys")
	backupPath := finalPath + ".keymaster-bak"

	// Step A: Remove any old backup file from a previous failed run.
	_ = d.sftp.Remove(backupPath)

	// Step B: Rename the current file to a backup. We ignore the error, as
	// the file may not exist on the first deployment.
	_ = d.sftp.Rename(finalPath, backupPath)

	// Step C: Rename the new file to the final destination.
	if err := d.sftp.Rename(tmpPath, finalPath); err != nil {
		// If the rename fails, try to restore the backup to leave the system in a stable state.
		_ = d.sftp.Rename(backupPath, finalPath)
		// Clean up the temp file regardless.
		_ = d.sftp.Remove(tmpPath)
		return fmt.Errorf("failed to rename authorized_keys file into place: %w", err)
	}

	// Step D: Success. Clean up the backup file.
	_ = d.sftp.Remove(backupPath)

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

// ErrHostKeySuccessfullyRetrieved is a sentinel error used to gracefully stop the SSH handshake
// in GetRemoteHostKey once the host key has been captured.
var ErrHostKeySuccessfullyRetrieved = errors.New("keymaster: successfully retrieved host key")

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
			return ErrHostKeySuccessfullyRetrieved
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
		// Check if it's our specific sentinel error.
		if errors.Is(err, ErrHostKeySuccessfullyRetrieved) {
			// Success, the key is in the channel.
			return <-keyChan, nil
		}
		// It's a different, real error (e.g., connection refused).
		return nil, fmt.Errorf("failed to connect to %s: %w", host, err)
	}

	// This case should ideally not be reached if the callback returns an error.
	return nil, fmt.Errorf("ssh.Dial succeeded unexpectedly, could not retrieve key")
}
