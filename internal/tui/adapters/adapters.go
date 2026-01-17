// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package adapters

import (
	"fmt"

	"github.com/toeirei/keymaster/internal/core"
	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/model"
)

// tuiStoreAdapter removed: TUI now uses internal/uiadapters.NewStoreAdapter() for store operations.

// tuiAccountReader adapts db helpers to core.AccountReader.
type tuiAccountReader struct{}

func (r *tuiAccountReader) GetAllAccounts() ([]model.Account, error) { return db.GetAllAccounts() }

var _ core.AccountReader = (*tuiAccountReader)(nil)

// tuiKeyReader adapts db helpers to core.KeyReader.
type tuiKeyReader struct{}

func (r *tuiKeyReader) GetAllPublicKeys() ([]model.PublicKey, error) {
	km := db.DefaultKeyManager()
	if km == nil {
		return nil, fmt.Errorf("no key manager available")
	}
	return km.GetAllPublicKeys()
}
func (r *tuiKeyReader) GetActiveSystemKey() (*model.SystemKey, error) { return db.GetActiveSystemKey() }
func (r *tuiKeyReader) GetSystemKeyBySerial(serial int) (*model.SystemKey, error) {
	return db.GetSystemKeyBySerial(serial)
}

var _ core.KeyReader = (*tuiKeyReader)(nil)

// tuiAuditReader adapts db helpers to core.AuditReader.
type tuiAuditReader struct{}

func (a *tuiAuditReader) GetAllAuditLogEntries() ([]model.AuditLogEntry, error) {
	return db.GetAllAuditLogEntries()
}

var _ core.AuditReader = (*tuiAuditReader)(nil)

// tuiKeyLister adapts the package-level KeyManager to core.KeyLister.
type tuiKeyLister struct{}

func (k *tuiKeyLister) GetGlobalPublicKeys() ([]model.PublicKey, error) {
	km := db.DefaultKeyManager()
	if km == nil {
		return nil, fmt.Errorf("no key manager available")
	}
	return km.GetGlobalPublicKeys()
}
func (k *tuiKeyLister) GetKeysForAccount(accountID int) ([]model.PublicKey, error) {
	km := db.DefaultKeyManager()
	if km == nil {
		return nil, fmt.Errorf("no key manager available")
	}
	return km.GetKeysForAccount(accountID)
}
func (k *tuiKeyLister) GetAllPublicKeys() ([]model.PublicKey, error) {
	km := db.DefaultKeyManager()
	if km == nil {
		return nil, fmt.Errorf("no key manager available")
	}
	return km.GetAllPublicKeys()
}

var _ core.KeyLister = (*tuiKeyLister)(nil)

// package-level adapter instances and helpers used by TUI runtime files.
var (
	AccountReader = &tuiAccountReader{}
	KeyReader     = &tuiKeyReader{}
	AuditReader   = &tuiAuditReader{}
	KeyLister     = &tuiKeyLister{}
)

// Note: small db delegators (DefaultKeyManager, DefaultKeySearcher, ToggleAccountStatus,
// DefaultAccountSearcher, DefaultAuditSearcher, HasSystemKeys) were removed
// from this package in Phase G5. Callers should use the shared helpers in
// `internal/ui` or call `internal/db` directly where appropriate.
