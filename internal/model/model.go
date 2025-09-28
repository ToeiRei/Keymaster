// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// package model defines the core data structures used throughout Keymaster.
// These structs represent the entities stored in the database and used by
// the application logic, such as accounts, keys, and audit logs.
package model // import "github.com/toeirei/keymaster/internal/model"

import (
	"fmt"
	"time"
)

// Account represents a user on a specific host (e.g., deploy@server-01).
// This is the core entity for which we manage access.
type Account struct {
	ID       int    // The primary key for the account.
	Username string // The SSH username for the account.
	Hostname string // The hostname or IP address of the target machine.
	Label    string // A user-friendly alias for the account (e.g., "prod-web-01").
	Tags     string // Comma-separated key:value pairs for organization.
	// Serial is the serial number of the SystemKey last deployed to this account.
	// A value of 0 indicates the account has never been deployed to.
	Serial int
	// IsActive determines if the account is included in bulk operations like 'deploy' and 'audit'.
	IsActive bool
}

// String returns a user-friendly representation of the account.
// It formats as "Label (user@host)" if a label is present, otherwise just "user@host".
func (a Account) String() string {
	base := fmt.Sprintf("%s@%s", a.Username, a.Hostname)
	if a.Label != "" {
		return fmt.Sprintf("%s (%s)", a.Label, base)
	}
	return base
}

// PublicKey represents a single SSH public key stored in the database.
type PublicKey struct {
	ID        int    // The primary key for the public key.
	Algorithm string // The key algorithm (e.g., "ssh-ed25519").
	KeyData   string // The base64-encoded key data.
	Comment   string // The unique comment associated with the key, used as an identifier.
	// IsGlobal indicates if the key should be deployed to all active accounts by default.
	IsGlobal bool
}

// String returns the full public key line suitable for an authorized_keys file.
func (k PublicKey) String() string {
	return fmt.Sprintf("%s %s %s", k.Algorithm, k.KeyData, k.Comment)
}

// SystemKey represents a key pair used by Keymaster itself for deployment.
// The private key is stored to allow for agentless operation.
type SystemKey struct {
	ID         int    // The primary key for the system key.
	Serial     int    // A unique, auto-incrementing number identifying this key version.
	PublicKey  string // The public part of the key in authorized_keys format.
	PrivateKey string // The private part of the key in PEM format.
	// IsActive indicates if this is the current key for new deployments. Only one key can be active.
	IsActive bool
}

// AuditLogEntry represents a single event in the audit log.
type AuditLogEntry struct {
	ID        int    // The primary key for the log entry.
	Timestamp string // The timestamp of the event (as a string for display simplicity).
	Username  string // The OS user who performed the action.
	Action    string // A category for the event (e.g., "DEPLOY_SUCCESS", "ADD_ACCOUNT").
	Details   string // A free-text description of the event.
}

// BootstrapSession represents an ongoing bootstrap operation for a new host.
// Sessions track temporary keys and pending account information during the bootstrap workflow.
type BootstrapSession struct {
	ID            string    // Unique session identifier.
	Username      string    // Username for the pending account.
	Hostname      string    // Hostname for the pending account.
	Label         string    // Optional label for the pending account.
	Tags          string    // Optional tags for the pending account.
	TempPublicKey string    // Temporary public key for initial access.
	CreatedAt     time.Time // When the session was created.
	ExpiresAt     time.Time // When the session expires.
	Status        string    // Current status (active, committing, completed, failed, orphaned).
}
