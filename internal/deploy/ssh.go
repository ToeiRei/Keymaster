// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// Package deploy provides functionality for connecting to remote hosts via SSH
// and managing their authorized_keys files. This file contains the core SSH and
// SFTP client logic for connecting, authenticating, and transferring files.
//
// It includes configurable timeout support and enhanced error classification
// to provide better user feedback when connections fail.
package deploy // import "github.com/toeirei/keymaster/internal/deploy"

import (
	"errors"
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

// Default timeout values for SSH operations
const (
	// DefaultConnectionTimeout is the default timeout for establishing SSH connections
	DefaultConnectionTimeout = 10 * time.Second
	// DefaultCommandTimeout is the default timeout for executing commands
	DefaultCommandTimeout = 30 * time.Second
	// DefaultHostKeyTimeout is the default timeout for host key retrieval
	DefaultHostKeyTimeout = 5 * time.Second
	// DefaultSFTPTimeout is the default timeout for SFTP operations
	DefaultSFTPTimeout = 60 * time.Second
)

// ConnectionConfig holds timeout configuration for SSH connections
type ConnectionConfig struct {
	ConnectionTimeout time.Duration
	CommandTimeout    time.Duration
	SFTPTimeout       time.Duration
}

// DefaultConnectionConfig returns a ConnectionConfig with default timeout values
func DefaultConnectionConfig() *ConnectionConfig {
	return &ConnectionConfig{
		ConnectionTimeout: DefaultConnectionTimeout,
		CommandTimeout:    DefaultCommandTimeout,
		SFTPTimeout:       DefaultSFTPTimeout,
	}
}

// Deployer handles the connection and deployment to a remote host.
type Deployer struct {
	client *ssh.Client
	sftp   *sftp.Client
	config *ConnectionConfig
}

// NewDeployer creates a new SSH connection and returns a Deployer.
// For bootstrap connections, use NewBootstrapDeployer instead.
func NewDeployer(host, user, privateKey string) (*Deployer, error) {
	return NewDeployerWithConfig(host, user, privateKey, DefaultConnectionConfig(), false)
}

// NewBootstrapDeployer creates a new SSH connection for bootstrap operations.
// It accepts any host key and saves it to the database for future connections.
func NewBootstrapDeployer(host, user, privateKey string) (*Deployer, error) {
	return NewDeployerWithConfig(host, user, privateKey, DefaultConnectionConfig(), true)
}

// NewBootstrapDeployerWithExpectedKey creates a new SSH connection for bootstrap operations
// that only accepts the specific expected host key. This is used when the host key has been
// manually verified by the user.
func NewBootstrapDeployerWithExpectedKey(host, user, privateKey, expectedHostKey string) (*Deployer, error) {
	return newDeployerWithExpectedHostKey(host, user, privateKey, DefaultConnectionConfig(), expectedHostKey)
}

// NewDeployerWithConfig creates a new SSH connection with custom timeout configuration.
func NewDeployerWithConfig(host, user, privateKey string, config *ConnectionConfig, isBootstrap bool) (*Deployer, error) {
	return newDeployerInternal(host, user, privateKey, config, isBootstrap)
}

// newDeployerInternal is the internal implementation for creating deployers.
func newDeployerInternal(host, user, privateKey string, config *ConnectionConfig, isBootstrap bool) (*Deployer, error) {
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
			sshConfig := &ssh.ClientConfig{
				User:            user,
				Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
				HostKeyCallback: hostKeyCallback,
				Timeout:         config.ConnectionTimeout,
			}
			client, err = ssh.Dial("tcp", addr, sshConfig)
			if err == nil {
				// Success! We connected with the system key.
				sftpClient, sftpErr := sftp.NewClient(client)
				if sftpErr != nil {
					client.Close()
					return nil, fmt.Errorf("failed to create sftp client: %w", sftpErr)
				}
				return &Deployer{client: client, sftp: sftpClient, config: config}, nil
			} else {
				// Classify the error for better debugging
				err = ClassifyConnectionError(host, err)
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

	sshConfig := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.PublicKeysCallback(agentClient.Signers)},
		HostKeyCallback: hostKeyCallback,
		Timeout:         config.ConnectionTimeout,
	}

	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		err = ClassifyConnectionError(host, err)
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
		config: config,
	}, nil
}

// newDeployerWithExpectedHostKey creates a deployer that only accepts a specific host key
func newDeployerWithExpectedHostKey(host, user, privateKey string, config *ConnectionConfig, expectedHostKey string) (*Deployer, error) {
	// Create a host key callback that only accepts the expected key
	hostKeyCallback := func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		presentedKey := string(ssh.MarshalAuthorizedKey(key))
		if strings.TrimSpace(presentedKey) != strings.TrimSpace(expectedHostKey) {
			return fmt.Errorf("host key mismatch: server presented different key than expected")
		}

		// Strip port if present for database storage
		hostOnly, _, err := net.SplitHostPort(hostname)
		if err != nil {
			hostOnly = hostname
		}

		// Save the verified host key to database
		if err := db.AddKnownHostKey(hostOnly, presentedKey); err != nil {
			// Log error but don't fail the connection
		}

		return nil
	}

	// Add port 22 if not specified
	addr := host
	if _, _, err := net.SplitHostPort(host); err != nil {
		addr = net.JoinHostPort(host, "22")
	}

	// Parse the private key
	signer, err := ssh.ParsePrivateKey([]byte(privateKey))
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	// Create SSH client configuration
	sshConfig := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: hostKeyCallback,
		Timeout:         config.ConnectionTimeout,
	}

	// Connect
	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		err = ClassifyConnectionError(host, err)
		return nil, err
	}

	// Create SFTP client
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to create sftp client: %w", err)
	}

	return &Deployer{
		client: client,
		sftp:   sftpClient,
		config: config,
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

// Error classification functions for better error handling

// IsConnectionTimeoutError checks if the error is due to a connection timeout
func IsConnectionTimeoutError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	// Check for various timeout-related error messages
	return strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "deadline exceeded") ||
		strings.Contains(errStr, "i/o timeout")
}

// IsConnectionRefusedError checks if the error is due to connection being refused
func IsConnectionRefusedError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	return strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "no route to host")
}

// IsAuthenticationError checks if the error is due to authentication failure
func IsAuthenticationError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	return strings.Contains(errStr, "authentication failed") ||
		strings.Contains(errStr, "permission denied") ||
		strings.Contains(errStr, "public key") ||
		strings.Contains(errStr, "unable to authenticate")
}

// IsHostKeyError checks if the error is due to host key verification failure
func IsHostKeyError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	return strings.Contains(errStr, "HOST KEY MISMATCH") ||
		strings.Contains(errStr, "unknown host key") ||
		strings.Contains(errStr, "host key verification failed")
}

// ClassifyConnectionError provides a more descriptive error message based on the error type
func ClassifyConnectionError(host string, err error) error {
	if err == nil {
		return nil
	}

	switch {
	case IsConnectionTimeoutError(err):
		return fmt.Errorf("connection to %s timed out (host may be unreachable or firewall blocking connection): %w", host, err)
	case IsConnectionRefusedError(err):
		return fmt.Errorf("connection to %s refused (SSH daemon may not be running or wrong port): %w", host, err)
	case IsAuthenticationError(err):
		return fmt.Errorf("authentication failed for %s (check SSH keys or credentials): %w", host, err)
	case IsHostKeyError(err):
		return fmt.Errorf("host key verification failed for %s (run 'keymaster trust-host %s' to accept): %w", host, host, err)
	default:
		return fmt.Errorf("failed to connect to %s: %w", host, err)
	}
}

// GetRemoteHostKey connects to a host just to retrieve its public key.
func GetRemoteHostKey(host string) (ssh.PublicKey, error) {
	return GetRemoteHostKeyWithTimeout(host, DefaultHostKeyTimeout)
}

// GetRemoteHostKeyWithTimeout connects to a host with a custom timeout to retrieve its public key.
func GetRemoteHostKeyWithTimeout(host string, timeout time.Duration) (ssh.PublicKey, error) {
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
		Timeout: timeout,
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
		err = ClassifyConnectionError(host, err)
		return nil, err
	}

	// This case should ideally not be reached if the callback returns an error.
	return nil, fmt.Errorf("ssh.Dial succeeded unexpectedly, could not retrieve key")
}
