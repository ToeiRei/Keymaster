// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

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
		// Register session for in-memory cleanup registry even when not persisted.
		bootstrap.RegisterSession(s)
		return s, nil
	}

	// Persist DB-friendly session representation.
	err = store.SaveBootstrapSession(s.ID, s.PendingAccount.Username, s.PendingAccount.Hostname, s.PendingAccount.Label, s.PendingAccount.Tags, s.TempKeyPair.GetPublicKey(), s.ExpiresAt, string(s.Status))
	if err != nil {
		// Wipe sensitive memory before returning
		s.Cleanup()
		return nil, err
	}

	// Register session for in-memory cleanup registry.
	bootstrap.RegisterSession(s)

	return s, nil
}

// CancelBootstrapSession unregisters the in-memory session and removes any
// persisted session record via the provided store. If store is nil, only the
// in-memory registry is updated.
func CancelBootstrapSession(store SessionStore, sessionID string) error {
	// Unregister from in-memory cleanup registry first.
	bootstrap.UnregisterSession(sessionID)
	if store == nil {
		return nil
	}
	return store.DeleteBootstrapSession(sessionID)
}

