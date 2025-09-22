package model

import "fmt"

// Account represents a user on a specific host (e.g., deploy@server-01).
// This is the core entity for which we manage access.
type Account struct {
	ID       int
	Username string
	Hostname string
	Label    string // A user-friendly alias for the account.
	Tags     string // Comma-separated key:value pairs.
	Serial   int
	IsActive bool
}

// String returns the user@host representation, prefixed with the label if it exists.
func (a Account) String() string {
	base := fmt.Sprintf("%s@%s", a.Username, a.Hostname)
	if a.Label != "" {
		return fmt.Sprintf("%s (%s)", a.Label, base)
	}
	return base
}

// PublicKey represents a single SSH public key stored in the database.
type PublicKey struct {
	ID        int
	Algorithm string
	KeyData   string
	Comment   string
}

// String returns the full public key line suitable for an authorized_keys file.
func (k PublicKey) String() string {
	return fmt.Sprintf("%s %s %s", k.Algorithm, k.KeyData, k.Comment)
}

// SystemKey represents a key pair used by Keymaster itself for deployment.
// The private key is stored to allow for agentless operation.
type SystemKey struct {
	ID         int
	Serial     int
	PublicKey  string
	PrivateKey string // Note: Storing private keys requires secure handling.
	IsActive   bool
}

// AuditLogEntry represents a single event in the audit log.
type AuditLogEntry struct {
	ID        int
	Timestamp string // Using string for simplicity in display
	Username  string
	Action    string
	Details   string
}
