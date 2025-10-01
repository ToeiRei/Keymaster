// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// package bootstrap provides cleanup and signal handling for bootstrap sessions.
// This file contains functionality to recover from crashes, handle graceful shutdown,
// and clean up orphaned temporary keys from remote hosts.
package bootstrap

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/pkg/sftp"
	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/model"
	"golang.org/x/crypto/ssh"
)

var (
	// Global registry of active bootstrap sessions for cleanup
	activeSessions = make(map[string]*BootstrapSession)
	sessionsMutex  sync.RWMutex

	// Signal handler installed flag
	signalHandlerInstalled bool
	signalHandlerMutex     sync.Mutex
)

// RegisterSession adds a bootstrap session to the active sessions registry.
// This ensures the session can be cleaned up even if the program crashes.
func RegisterSession(session *BootstrapSession) {
	sessionsMutex.Lock()
	defer sessionsMutex.Unlock()
	activeSessions[session.ID] = session
}

// UnregisterSession removes a bootstrap session from the active sessions registry.
// This should be called when a session completes successfully or is manually cancelled.
func UnregisterSession(sessionID string) {
	sessionsMutex.Lock()
	defer sessionsMutex.Unlock()

	if session, exists := activeSessions[sessionID]; exists {
		session.Cleanup() // Secure memory wipe
		delete(activeSessions, sessionID)
	}
}

// InstallSignalHandler sets up signal handling for graceful shutdown.
// This ensures that temporary keys are cleaned up even if the program is interrupted.
// It's safe to call this multiple times - subsequent calls are ignored.
func InstallSignalHandler() {
	signalHandlerMutex.Lock()
	defer signalHandlerMutex.Unlock()

	if signalHandlerInstalled {
		return // Already installed
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan

		CleanupAllActiveSessions()

		os.Exit(0)
	}()

	signalHandlerInstalled = true
}

// CleanupAllActiveSessions attempts to remove temporary keys from remote hosts
// and clean up all currently active bootstrap sessions.
func CleanupAllActiveSessions() error {
	sessionsMutex.Lock()
	defer sessionsMutex.Unlock()

	var lastError error

	for _, session := range activeSessions {
		if err := cleanupSession(session); err != nil {
			lastError = err
		}
	}

	// Clear the registry
	activeSessions = make(map[string]*BootstrapSession)

	return lastError
}

// RecoverFromCrash identifies and cleans up orphaned bootstrap sessions.
// This should be called during application startup to handle sessions
// that were interrupted by a program crash or force-kill.
func RecoverFromCrash() error {
	// Mark all active sessions as orphaned first
	if err := markActiveSessionsAsOrphaned(); err != nil {
		return fmt.Errorf("failed to mark sessions as orphaned: %w", err)
	}

	// Get all orphaned sessions
	orphanedSessions, err := db.GetOrphanedBootstrapSessions()
	if err != nil {
		return fmt.Errorf("failed to get orphaned sessions: %w", err)
	}

	// Attempt to clean up each orphaned session
	var lastError error
	for _, session := range orphanedSessions {
		if err := cleanupOrphanedSessionModel(session); err != nil {
			lastError = err
		}
	}

	return lastError
}

// CleanupExpiredSessions removes bootstrap sessions that have exceeded their timeout.
// This should be called periodically to prevent database accumulation.
func CleanupExpiredSessions() error {
	expiredSessions, err := db.GetExpiredBootstrapSessions()
	if err != nil {
		return fmt.Errorf("failed to get expired sessions: %w", err)
	}

	var lastError error
	for _, session := range expiredSessions {
		if err := cleanupExpiredSessionModel(session); err != nil {
			lastError = err
		}
	}

	return lastError
}

// StartSessionReaper launches a background goroutine that periodically cleans up
// expired bootstrap sessions. This helps prevent database accumulation.
func StartSessionReaper() {
	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		for range ticker.C {
			CleanupExpiredSessions()
		}
	}()
}

// markActiveSessionsAsOrphaned marks all currently active sessions as orphaned.
// This is called during startup to identify sessions that were interrupted by a crash.
func markActiveSessionsAsOrphaned() error {
	// This would need to be implemented in the store interface
	// For now, we'll handle it during recovery
	return nil
}

// cleanupSession attempts to remove a temporary key from a remote host and cleanup the session.
func cleanupSession(session *BootstrapSession) error {
	// Log the signal interruption
	_ = db.LogAction("BOOTSTRAP_FAILED", fmt.Sprintf("%s@%s, reason: interrupted by signal",
		session.PendingAccount.Username, session.PendingAccount.Hostname))

	// Attempt to remove temporary key from remote host
	removeTempKeyFromRemoteHost(session)

	// Cleanup sensitive memory
	session.Cleanup()

	// Update session status in database
	if err := session.UpdateStatus(StatusFailed); err != nil {
		return fmt.Errorf("failed to update session status: %w", err)
	}

	return nil
}

// cleanupOrphanedSession handles cleanup of a session that was abandoned due to a crash.
func cleanupOrphanedSession(session *BootstrapSession) error {
	// Attempt to remove temporary key from remote host
	removeTempKeyFromRemoteHost(session)

	// Remove session from database
	if err := db.DeleteBootstrapSession(session.ID); err != nil {
		return fmt.Errorf("failed to delete orphaned session: %w", err)
	}

	return nil
}

// cleanupExpiredSession handles cleanup of a session that has exceeded its timeout.
func cleanupExpiredSession(session *BootstrapSession) error {
	// Attempt to remove temporary key from remote host
	removeTempKeyFromRemoteHost(session)

	// Remove session from database
	if err := db.DeleteBootstrapSession(session.ID); err != nil {
		return fmt.Errorf("failed to delete expired session: %w", err)
	}

	return nil
}

// removeTempKeyFromRemoteHost attempts to connect to a remote host and remove
// the temporary bootstrap key from the authorized_keys file.
func removeTempKeyFromRemoteHost(session *BootstrapSession) error {
	if session.TempKeyPair == nil {
		return fmt.Errorf("no temporary key pair in session")
	}

	// Parse the private key for SSH connection
	signer, err := ssh.ParsePrivateKey(session.TempKeyPair.GetPrivateKeyPEM())
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}

	// Create SSH client configuration with proper host key verification
	hostKeyCallback := func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		// Canonicalize hostname to host:port format
		canonical := hostname
		if _, _, err := net.SplitHostPort(hostname); err != nil {
			canonical = net.JoinHostPort(hostname, "22")
		}

		// Get the presented key
		presentedKey := string(ssh.MarshalAuthorizedKey(key))

		// Verify against known_hosts database
		knownKey, err := db.GetKnownHostKey(canonical)
		if err != nil {
			return fmt.Errorf("failed to query known_hosts database: %w", err)
		}

		// If no key found for canonical format, try legacy host-only format
		if knownKey == "" {
			if hostOnly, _, err := net.SplitHostPort(canonical); err == nil {
				legacyKey, lerr := db.GetKnownHostKey(hostOnly)
				if lerr != nil {
					return fmt.Errorf("failed to query known_hosts database: %w", lerr)
				}
				if legacyKey != "" {
					knownKey = legacyKey
				}
			}
		}

		// If still no key found, reject the connection
		if knownKey == "" {
			return fmt.Errorf("unknown host key for %s - cannot verify host identity", canonical)
		}

		// Verify the key matches
		if knownKey != presentedKey {
			return fmt.Errorf("!!! HOST KEY MISMATCH FOR %s !!! - possible man-in-the-middle attack", canonical)
		}

		return nil
	}

	config := &ssh.ClientConfig{
		User: session.PendingAccount.Username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: hostKeyCallback,
		Timeout:         10 * time.Second,
	}

	// Connect to the remote host
	conn, err := ssh.Dial("tcp", session.PendingAccount.Hostname+":22", config)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", session.PendingAccount.Hostname, err)
	}
	defer conn.Close()

	// Create SFTP session
	sftpClient, err := sftp.NewClient(conn)
	if err != nil {
		return fmt.Errorf("failed to create SFTP client: %w", err)
	}
	defer sftpClient.Close()

	// Read current authorized_keys file
	authKeysPath := ".ssh/authorized_keys"
	file, err := sftpClient.Open(authKeysPath)
	if err != nil {
		return fmt.Errorf("failed to open authorized_keys: %w", err)
	}
	defer file.Close()

	content := make([]byte, 0, 4096)
	buffer := make([]byte, 1024)
	for {
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return fmt.Errorf("failed to read authorized_keys: %w", err)
		}
		content = append(content, buffer[:n]...)
		if err == io.EOF {
			break
		}
	}

	// Remove our temporary key from the content
	tempKeyLine := session.TempKeyPair.GetPublicKey()
	newContent := removeLine(string(content), tempKeyLine)

	// Write back the cleaned content
	outFile, err := sftpClient.Create(authKeysPath)
	if err != nil {
		return fmt.Errorf("failed to create authorized_keys: %w", err)
	}
	defer outFile.Close()

	if _, err := outFile.Write([]byte(newContent)); err != nil {
		return fmt.Errorf("failed to write cleaned authorized_keys: %w", err)
	}

	return nil
}

// removeLine removes a specific line from a multi-line string.
func removeLine(content, lineToRemove string) string {
	lines := strings.Split(content, "\n")
	var filteredLines []string

	for _, line := range lines {
		if strings.TrimSpace(line) != strings.TrimSpace(lineToRemove) {
			filteredLines = append(filteredLines, line)
		}
	}

	return strings.Join(filteredLines, "\n")
}

// cleanupOrphanedSessionModel cleans up an orphaned session using model.BootstrapSession.
// This is a simplified version that just removes the session from the database.
func cleanupOrphanedSessionModel(session *model.BootstrapSession) error {
	// Log the orphaned session cleanup
	_ = db.LogAction("BOOTSTRAP_FAILED", fmt.Sprintf("%s@%s, reason: session orphaned",
		session.Username, session.Hostname))

	// For now, just remove from database - remote cleanup would require
	// reconstructing the temporary key, which is complex
	if err := db.DeleteBootstrapSession(session.ID); err != nil {
		return fmt.Errorf("failed to delete orphaned session: %w", err)
	}

	return nil
}

// cleanupExpiredSessionModel cleans up an expired session using model.BootstrapSession.
// This is a simplified version that just removes the session from the database.
func cleanupExpiredSessionModel(session *model.BootstrapSession) error {
	// Log the expired session cleanup
	_ = db.LogAction("BOOTSTRAP_FAILED", fmt.Sprintf("%s@%s, reason: session expired",
		session.Username, session.Hostname))

	// For now, just remove from database - remote cleanup would require
	// reconstructing the temporary key, which is complex
	if err := db.DeleteBootstrapSession(session.ID); err != nil {
		return fmt.Errorf("failed to delete expired session: %w", err)
	}

	return nil
}
