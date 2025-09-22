package model

import "fmt"

// Account represents a user on a specific host (e.g., deploy@server-01).
// This is the core entity for which we manage access.
type Account struct {
	ID       int
	Username string
	Hostname string
	Serial   int
}

// String returns the user@host representation.
func (a Account) String() string {
	return fmt.Sprintf("%s@%s", a.Username, a.Hostname)
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
