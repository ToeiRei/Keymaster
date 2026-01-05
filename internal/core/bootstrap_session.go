package core

import (
	"github.com/toeirei/keymaster/internal/bootstrap"
)

// NewSession creates a new bootstrap session (including temporary keypair)
// and persists its DB representation via the provided SessionStore. The in-memory
// session (with private key) is returned so callers may render commands and
// perform connection tests. If store is nil, the session is not persisted.
func NewSession(store SessionStore, username, hostname, label, tags string) (*bootstrap.BootstrapSession, error) {
	s, err := bootstrap.NewBootstrapSession(username, hostname, label, tags)
	if err != nil {
		return nil, err
	}

	if store == nil {
		return s, nil
	}

	// Persist DB-friendly session representation.
	err = store.SaveBootstrapSession(s.ID, s.PendingAccount.Username, s.PendingAccount.Hostname, s.PendingAccount.Label, s.PendingAccount.Tags, s.TempKeyPair.GetPublicKey(), s.ExpiresAt, string(s.Status))
	if err != nil {
		// Wipe sensitive memory before returning
		s.Cleanup()
		return nil, err
	}

	return s, nil
}
