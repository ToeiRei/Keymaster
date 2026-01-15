// Copyright (c) 2026 Keymaster Team
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
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/logging"
	"github.com/toeirei/keymaster/internal/security"
	"golang.org/x/crypto/ssh"
)

// ErrPassphraseRequired is a sentinel error returned when an encrypted key is
// encountered but no passphrase was provided, signaling that the caller should prompt for one.
var ErrPassphraseRequired = errors.New("passphrase required for encrypted system key")

// CanonicalizeHostPort returns a normalized host:port string.
// - If no port is provided, :22 is assumed.
// - IPv6 literals will be bracketed as needed (e.g., [2001:db8::1]:22).
// - If input is of the form user@host, the user part is discarded.
// StripIPv6Brackets removes surrounding [ ] from an IPv6 literal if present.
func StripIPv6Brackets(host string) string {
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		return strings.TrimSuffix(strings.TrimPrefix(host, "["), "]")
	}
	return host
}

// ParseHostPort splits an address into host and port.
// Behavior:
// - Accepts host, host:port, [ipv6], [ipv6]:port, ipv6, ipv6:port
// - Returns port "" if not specified
// - Returns host without IPv6 brackets
func ParseHostPort(addr string) (host string, port string, err error) {
	s := strings.TrimSpace(addr)
	if s == "" {
		return "", "", fmt.Errorf("empty address")
	}
	// Strip user@ part if present
	if at := strings.LastIndex(s, "@"); at != -1 {
		s = s[at+1:]
	}

	// If bracketed IPv6
	if strings.HasPrefix(s, "[") {
		// regex: ^\[([^\]]+)\](?::(\d+))?$ -> capture host and optional port
		re := regexp.MustCompile(`^\[([^\]]+)\](?::(\d+))?$`)
		m := re.FindStringSubmatch(s)
		if m == nil {
			return "", "", fmt.Errorf("invalid bracketed IPv6: %s", s)
		}
		return m[1], m[2], nil
	}

	// Try net.SplitHostPort for host:port or ipv6:port (unbracketed)
	if h, p, e := net.SplitHostPort(s); e == nil {
		return h, p, nil
	}

	// No port specified, whole string is host (could be ipv4, name, or unbracketed ipv6)
	return s, "", nil
}

// JoinHostPort joins host and port into a canonical host:port.
// - If port is empty, defaultPort is used.
// - IPv6 hosts will be bracketed.
func JoinHostPort(host, port, defaultPort string) string {
	h := StripIPv6Brackets(strings.TrimSpace(host))
	p := strings.TrimSpace(port)
	if p == "" {
		p = defaultPort
	}
	return net.JoinHostPort(h, p)
}

// CanonicalizeHostPort returns host:port form using default 22 if missing.
func CanonicalizeHostPort(input string) string {
	host, port, _ := ParseHostPort(input)
	return JoinHostPort(host, port, "22")
}

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

// sftpClient defines an interface for SFTP operations, allowing for mocking in tests.
// It is satisfied by the *sftp.Client type.
type sftpClient interface {
	// Create and Open return an io.ReadWriteCloser to allow easier mocking
	// in tests while still allowing production code to use *sftp.File which
	// implements io.ReadWriteCloser.
	Create(path string) (io.ReadWriteCloser, error)
	Stat(p string) (os.FileInfo, error)
	Mkdir(path string) error
	Chmod(path string, mode os.FileMode) error
	Remove(path string) error
	Rename(oldpath, newpath string) error
	Open(path string) (io.ReadWriteCloser, error)
	Close() error
}

// sftpClientAdapter adapts a *sftp.Client to the sftpClient interface by
// delegating calls and returning *sftp.File values as io.ReadWriteCloser.
// sftpRaw is an internal abstraction over the concrete *sftp.Client methods we
// use. Using this interface allows tests to inject a mock implementation while
// production code wraps the real *sftp.Client behind a small adapter.
type sftpRaw interface {
	Create(path string) (io.ReadWriteCloser, error)
	Stat(p string) (os.FileInfo, error)
	Mkdir(path string) error
	Chmod(path string, mode os.FileMode) error
	Remove(path string) error
	Rename(oldpath, newpath string) error
	Open(path string) (io.ReadWriteCloser, error)
	Close() error
}

type sftpClientAdapter struct {
	client sftpRaw
}

func (a *sftpClientAdapter) Create(path string) (io.ReadWriteCloser, error) {
	return a.client.Create(path)
}

func (a *sftpClientAdapter) Stat(p string) (os.FileInfo, error) {
	return a.client.Stat(p)
}

func (a *sftpClientAdapter) Mkdir(path string) error {
	return a.client.Mkdir(path)
}

func (a *sftpClientAdapter) Chmod(path string, mode os.FileMode) error {
	return a.client.Chmod(path, mode)
}

func (a *sftpClientAdapter) Remove(path string) error {
	return a.client.Remove(path)
}

func (a *sftpClientAdapter) Rename(oldpath, newpath string) error {
	return a.client.Rename(oldpath, newpath)
}

func (a *sftpClientAdapter) Open(path string) (io.ReadWriteCloser, error) {
	return a.client.Open(path)
}

func (a *sftpClientAdapter) Close() error {
	if a == nil || a.client == nil {
		return nil
	}
	return a.client.Close()
}

// sftpRealAdapter wraps a *sftp.Client and implements sftpRaw by delegating
// calls and converting the concrete *sftp.File return values to
// io.ReadWriteCloser where necessary.
type sftpRealAdapter struct {
	client *sftp.Client
}

func (r *sftpRealAdapter) Create(path string) (io.ReadWriteCloser, error) {
	return r.client.Create(path)
}

func (r *sftpRealAdapter) Stat(p string) (os.FileInfo, error) { return r.client.Stat(p) }
func (r *sftpRealAdapter) Mkdir(path string) error            { return r.client.Mkdir(path) }
func (r *sftpRealAdapter) Chmod(path string, mode os.FileMode) error {
	return r.client.Chmod(path, mode)
}
func (r *sftpRealAdapter) Remove(path string) error { return r.client.Remove(path) }
func (r *sftpRealAdapter) Rename(oldpath, newpath string) error {
	return r.client.Rename(oldpath, newpath)
}
func (r *sftpRealAdapter) Open(path string) (io.ReadWriteCloser, error) { return r.client.Open(path) }
func (r *sftpRealAdapter) Close() error {
	if r == nil || r.client == nil {
		return nil
	}
	return r.client.Close()
}

// sshClientIface is a minimal interface used by this package to represent an
// SSH client. Using an interface allows tests to provide fake implementations
// without constructing a concrete *ssh.Client.
type sshClientIface interface {
	Close() error
}

// Deployer handles the connection and deployment to a remote host.
type Deployer struct {
	client sshClientIface
	sftp   sftpClient
	config *ConnectionConfig
}

// NewDeployerFunc is a overridable factory used to create Deployers. Tests may
// replace this with a fake implementation to avoid real network connections.
var NewDeployerFunc = func(host, user string, privateKey security.Secret, passphrase []byte) (*Deployer, error) {
	return NewDeployerWithConfig(host, user, privateKey, passphrase, DefaultConnectionConfig(), false)
}

// sshDial is a package-level wrapper around ssh.Dial to allow tests to
// replace the dialing behavior. Tests can override this to return a fake
// *ssh.Client or a controlled error without making real network calls.
// sshDial is a package-level wrapper around ssh.Dial to allow tests to
// replace the dialing behavior. It returns an `sshClientIface` so tests can
// provide fake clients.
var sshDial = func(network, addr string, cfg *ssh.ClientConfig) (sshClientIface, error) {
	return ssh.Dial(network, addr, cfg)
}

// newSftpClient is a package-level wrapper used to create an sftpRaw from an
// existing *ssh.Client. By default it wraps the real *sftp.Client with
// sftpRealAdapter. Tests may override this to return a mock sftpRaw.
var newSftpClient = func(c sshClientIface) (sftpRaw, error) {
	// The real sftp.NewClient requires a concrete *ssh.Client. In production
	// we expect the provided client to be a *ssh.Client; tests may override
	// this variable to accept alternative types.
	if c == nil {
		return nil, fmt.Errorf("nil ssh client")
	}
	if realClient, ok := c.(*ssh.Client); ok {
		real, err := sftp.NewClient(realClient)
		if err != nil {
			return nil, err
		}
		return &sftpRealAdapter{client: real}, nil
	}
	return nil, fmt.Errorf("unsupported ssh client type for sftp client creation")
}

// closeSSHClient is a safe wrapper around (*ssh.Client).Close that protects
// against nil pointers and panics that can occur when tests provide a
// zero-valued *ssh.Client. Tests may override this to provide a noop or
// controlled behavior.
var closeSSHClient = func(c sshClientIface) error {
	if c == nil {
		return nil
	}
	defer func() {
		if r := recover(); r != nil {
			_ = r
		}
	}()
	return c.Close()
}

// sshAgentGetter is an overrideable hook used to retrieve an SSH agent for
// authentication. Tests may replace this to return a fake agent.Agent.
var sshAgentGetter = getSSHAgent

// NewDeployer creates a new SSH connection and returns a Deployer.
// For bootstrap connections, use NewBootstrapDeployer instead.
func NewDeployer(host, user string, privateKey security.Secret, passphrase []byte) (*Deployer, error) {
	return NewDeployerWithConfig(host, user, privateKey, passphrase, DefaultConnectionConfig(), false)
}

// NewBootstrapDeployer creates a new SSH connection for bootstrap operations.
// It accepts any host key and saves it to the database for future connections.
func NewBootstrapDeployer(host, user string, privateKey security.Secret) (*Deployer, error) {
	return NewDeployerWithConfig(host, user, privateKey, nil, DefaultConnectionConfig(), true)
}

// NewBootstrapDeployerWithExpectedKey creates a new SSH connection for bootstrap operations
// that only accepts the specific expected host key. This is used when the host key has been
// manually verified by the user.
func NewBootstrapDeployerWithExpectedKey(host, user string, privateKey security.Secret, expectedHostKey string) (*Deployer, error) {
	return newDeployerWithExpectedHostKey(host, user, privateKey, DefaultConnectionConfig(), expectedHostKey)
}

// NewDeployerWithConfig creates a new SSH connection with custom timeout configuration.
func NewDeployerWithConfig(host, user string, privateKey security.Secret, passphrase []byte, config *ConnectionConfig, isBootstrap bool) (*Deployer, error) {
	return newDeployerInternal(host, user, privateKey, passphrase, config, isBootstrap)
}

// newDeployerInternal is the internal implementation for creating deployers.
func newDeployerInternal(host, user string, privateKey security.Secret, passphrase []byte, config *ConnectionConfig, isBootstrap bool) (*Deployer, error) {
	// Define the host key callback based on bootstrap mode.
	var hostKeyCallback ssh.HostKeyCallback

	if isBootstrap {
		// For bootstrap, accept any host key and save it as canonical host:port
		hostKeyCallback = func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			canonical := CanonicalizeHostPort(hostname)

			// Save the host key for future connections
			presentedKey := string(ssh.MarshalAuthorizedKey(key))
			if err := db.AddKnownHostKey(canonical, presentedKey); err != nil {
				logging.Warnf("failed to save known host key for %s: %v", canonical, err)
			}

			return nil // Accept the key for bootstrap
		}
	} else {
		// Normal mode: verify host keys
		hostKeyCallback = func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			// Always check canonical host:port first
			canonical := CanonicalizeHostPort(hostname)

			// The key is presented in the format "ssh-ed25519 AAA..."
			presentedKey := string(ssh.MarshalAuthorizedKey(key))

			// Check if we have a trusted key for this canonical host:port in our database.
			knownKey, err := db.GetKnownHostKey(canonical)
			if err != nil {
				return fmt.Errorf("failed to query known_hosts database: %w", err)
			}

			// If we don't have a key, this is the first connection.
			if knownKey == "" {
				// Backward compatibility: try legacy host-only key (without port)
				if hostOnly, _, err := net.SplitHostPort(canonical); err == nil {
					legacyKey, lerr := db.GetKnownHostKey(hostOnly)
					if lerr != nil {
						return fmt.Errorf("failed to query known_hosts database: %w", lerr)
					}
					if legacyKey != "" {
						knownKey = legacyKey
					}
				}
				if knownKey == "" {
					return fmt.Errorf("unknown host key for %s. run 'keymaster trust-host' to add it", canonical)
				}
			}

			// If the key exists, it must match exactly.
			if knownKey != presentedKey {
				return fmt.Errorf("!!! HOST KEY MISMATCH FOR %s !!!\nRemote key presented: %s\nThis could be a man-in-the-middle attack", canonical, presentedKey)
			}

			return nil // Host key is trusted.
		}
	}

	// Add port 22 if not specified.
	addr := CanonicalizeHostPort(host)
	var client sshClientIface

	// If a private key is provided, use it exclusively. This is the standard path
	// for deployment and auditing with a Keymaster system key.
	if len(privateKey) != 0 {
		signer, err := ssh.ParsePrivateKey(privateKey.Bytes())
		if err != nil {
			// Check if the error is because the key is encrypted.
			var pme *ssh.PassphraseMissingError
			if errors.As(err, &pme) {
				// If it's encrypted and we have a passphrase, try to parse it again.
				if len(passphrase) > 0 {
					signer, err = ssh.ParsePrivateKeyWithPassphrase(privateKey.Bytes(), passphrase)
				} else {
					// Key is encrypted, but we have no passphrase. Signal to the caller.
					return nil, ErrPassphraseRequired
				}
			}
		}

		// If we have a valid signer at this point (either unencrypted or successfully decrypted).
		if err == nil && signer != nil {
			sshConfig := &ssh.ClientConfig{
				User:            user,
				Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
				HostKeyCallback: hostKeyCallback,
				Timeout:         config.ConnectionTimeout,
			}
			client, err = sshDial("tcp", addr, sshConfig)
			if err == nil {
				// Success! We connected with the system key.
				sftpClient, sftpErr := newSftpClient(client)
				if sftpErr != nil {
					_ = closeSSHClient(client)
					return nil, fmt.Errorf("failed to create sftp client: %w", sftpErr)
				}
				return &Deployer{client: client, sftp: &sftpClientAdapter{client: sftpClient}, config: config}, nil
			} else {
				// Classify the error for better debugging (log it); we'll fall back to ssh-agent.
				logging.Infof("system key connection attempt failed for %s: %v", host, err)
			}
			// If we provided a key and it failed, we will fall through to try the agent.
		}
	}

	// If no private key was provided, attempt to use the SSH agent.
	// This is used for bootstrapping/importing keys.
	agentClient := sshAgentGetter()
	if agentClient == nil {
		return nil, fmt.Errorf("no authentication method available (system key failed and no ssh agent found)")
	}

	sshConfig := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.PublicKeysCallback(agentClient.Signers)},
		HostKeyCallback: hostKeyCallback,
		Timeout:         config.ConnectionTimeout,
	}

	client, err := sshDial("tcp", addr, sshConfig)
	if err != nil {
		err = ClassifyConnectionError(host, err)
		return nil, fmt.Errorf("connection with ssh agent failed: %w", err)
	}

	// Success with agent.

	sftpClient, err := newSftpClient(client)
	if err != nil {
		_ = closeSSHClient(client)
		return nil, fmt.Errorf("failed to create sftp client: %w", err)
	}

	return &Deployer{
		client: client,
		sftp:   &sftpClientAdapter{client: sftpClient},
		config: config,
	}, nil
}

// newDeployerWithExpectedHostKey creates a deployer that only accepts a specific host key
func newDeployerWithExpectedHostKey(host, user string, privateKey security.Secret, config *ConnectionConfig, expectedHostKey string) (*Deployer, error) {
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
			logging.Warnf("failed to save verified host key for %s: %v", hostOnly, err)
		}

		return nil
	}

	// Add port 22 if not specified
	addr := CanonicalizeHostPort(host)

	// Parse the private key
	signer, err := ssh.ParsePrivateKey(privateKey.Bytes())
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
	client, err := sshDial("tcp", addr, sshConfig)
	if err != nil {
		err = ClassifyConnectionError(host, err)
		return nil, err
	}

	// Create SFTP client
	sftpClient, err := newSftpClient(client)
	if err != nil {
		_ = closeSSHClient(client)
		return nil, fmt.Errorf("failed to create sftp client: %w", err)
	}

	return &Deployer{
		client: client,
		sftp:   &sftpClientAdapter{client: sftpClient},
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
		_ = f.Close()
		// Best effort to clean up the failed upload
		_ = d.sftp.Remove(tmpPath)
		return fmt.Errorf("failed to write to temporary file on remote: %w", err)
	}
	_ = f.Close()

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
		_ = d.sftp.Close()
	}
	_ = closeSSHClient(d.client)
}

// GetAuthorizedKeys reads and returns the content of the remote authorized_keys file.
func (d *Deployer) GetAuthorizedKeys() ([]byte, error) {
	finalPath := ".ssh/authorized_keys"
	f, err := d.sftp.Open(finalPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open remote file %s: %w", finalPath, err)
	}
	defer func() { _ = f.Close() }()

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

	addr := CanonicalizeHostPort(host)

	// We expect ssh.Dial to fail with our specific error.
	_, err := sshDial("tcp", addr, config)
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
