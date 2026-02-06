// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// package bootstrap provides functionality for bootstrapping new hosts by creating
// temporary SSH keys, managing bootstrap sessions, and performing atomic account setup.
// This package handles the complex workflow of securely adding new hosts to Keymaster
// without requiring manual system key distribution.
package bootstrap // import "github.com/toeirei/keymaster/core/bootstrap"

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	internalSSH "github.com/toeirei/keymaster/core/crypto/ssh"
	"github.com/toeirei/keymaster/core/db"
	"github.com/toeirei/keymaster/core/model"
)

// SessionStatus represents the current state of a bootstrap session.
type SessionStatus string

const (
	// StatusActive indicates the session is actively being used for bootstrap.
	StatusActive SessionStatus = "active"
	// StatusCommitting indicates the session is in the final deployment phase.
	StatusCommitting SessionStatus = "committing"
	// StatusCompleted indicates the bootstrap was successful and session can be cleaned up.
	StatusCompleted SessionStatus = "completed"
	// StatusFailed indicates the bootstrap failed and session should be cleaned up.
	StatusFailed SessionStatus = "failed"
	// StatusOrphaned indicates the session was abandoned (e.g., due to program crash).
	StatusOrphaned SessionStatus = "orphaned"
)

// BootstrapTimeout is the maximum duration a bootstrap session can remain active.
const BootstrapTimeout = 30 * time.Minute

// TemporaryKeyPair holds a temporary SSH key pair used during bootstrap.
// The private key is kept in memory only and should be securely wiped after use.
type TemporaryKeyPair struct {
	privateKey []byte // PEM-encoded private key - NEVER persist to disk
	publicKey  string // Public key in authorized_keys format
	createdAt  time.Time
}

// BootstrapSession represents an ongoing bootstrap operation for a new host.
// Sessions are persisted to the database to enable recovery from crashes or interruptions.
type BootstrapSession struct {
	ID             string            // Unique session identifier
	PendingAccount model.Account     // Account data to be created (not yet in DB)
	TempKeyPair    *TemporaryKeyPair // Temporary SSH key for initial access
	SelectedKeys   []string          // Comments of keys to assign to the account
	Status         SessionStatus     // Current state of the bootstrap process
	CreatedAt      time.Time         // When the session was created
	ExpiresAt      time.Time         // When the session expires and should be cleaned up
}

// NewBootstrapSession creates a new bootstrap session with a temporary key pair.
// The session is assigned a unique ID and configured with reasonable defaults.
func NewBootstrapSession(username, hostname, label, tags string) (*BootstrapSession, error) {
	// Generate unique session ID
	sessionID, err := generateSessionID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session ID: %w", err)
	}

	// Generate temporary key pair
	tempKeyPair, err := generateTemporaryKeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate temporary key pair: %w", err)
	}

	now := defaultClock.Now()
	session := &BootstrapSession{
		ID: sessionID,
		PendingAccount: model.Account{
			Username: username,
			Hostname: hostname,
			Label:    label,
			Tags:     tags,
			IsActive: true,
		},
		TempKeyPair: tempKeyPair,
		Status:      StatusActive,
		CreatedAt:   now,
		ExpiresAt:   now.Add(BootstrapTimeout),
	}

	return session, nil
}

// GetBootstrapCommand returns the shell command that should be pasted on the target host
// to install the temporary SSH key. This command creates the .ssh directory if needed,
// adds the temporary key, and sets proper permissions.
func (s *BootstrapSession) GetBootstrapCommand() string {
	return fmt.Sprintf(
		"mkdir -p ~/.ssh && echo '%s' >> ~/.ssh/authorized_keys && chmod 700 ~/.ssh && chmod 600 ~/.ssh/authorized_keys",
		s.TempKeyPair.publicKey,
	)
}

// IsExpired returns true if the session has exceeded its timeout duration.
func (s *BootstrapSession) IsExpired() bool {
	return defaultClock.Now().After(s.ExpiresAt)
}

// Cleanup securely wipes sensitive data from memory.
// This should be called when the session is no longer needed.
func (s *BootstrapSession) Cleanup() {
	if s.TempKeyPair != nil {
		s.TempKeyPair.Cleanup()
	}
}

// Cleanup securely overwrites the private key in memory.
func (t *TemporaryKeyPair) Cleanup() {
	if t.privateKey != nil {
		// Secure memory wipe - overwrite with random data first, then zeros
		for i := range t.privateKey {
			t.privateKey[i] = 0
		}
		clear(t.privateKey)
		t.privateKey = nil
	}
}

// GetPrivateKeyPEM returns the PEM-encoded private key for SSH authentication.
// This should only be used for establishing the initial connection.
func (t *TemporaryKeyPair) GetPrivateKeyPEM() []byte {
	return t.privateKey
}

// GetPublicKey returns the public key in authorized_keys format.
func (t *TemporaryKeyPair) GetPublicKey() string {
	return t.publicKey
}

// generateSessionID creates a cryptographically secure random session identifier.
func generateSessionID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// generateTemporaryKeyPair creates a new Ed25519 key pair for temporary access.
// The private key is kept in memory only and should be securely wiped after use.
func generateTemporaryKeyPair() (*TemporaryKeyPair, error) {
	// Generate Ed25519 key pair using the existing crypto package
	publicKeyLine, privateKeyPEM, err := internalSSH.GenerateAndMarshalEd25519Key("keymaster-bootstrap-temp", "")
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	return &TemporaryKeyPair{
		privateKey: []byte(privateKeyPEM),
		publicKey:  publicKeyLine,
		createdAt:  defaultClock.Now(),
	}, nil
}

// Save persists the bootstrap session to the database.
func (s *BootstrapSession) Save() error {
	return db.SaveBootstrapSession(s.ID, s.PendingAccount.Username, s.PendingAccount.Hostname,
		s.PendingAccount.Label, s.PendingAccount.Tags, s.TempKeyPair.publicKey, s.ExpiresAt, string(s.Status))
}

// Delete removes the bootstrap session from the database.
func (s *BootstrapSession) Delete() error {
	return db.DeleteBootstrapSession(s.ID)
}

// UpdateStatus changes the session status in the database.
func (s *BootstrapSession) UpdateStatus(status SessionStatus) error {
	s.Status = status
	return db.UpdateBootstrapSessionStatus(s.ID, string(status))
}
