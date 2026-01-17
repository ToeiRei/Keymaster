// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package adapters

import (
	"context"
	"fmt"
	"strconv"

	"github.com/toeirei/keymaster/internal/core"
	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/keys"
	"github.com/toeirei/keymaster/internal/model"
)

// tuiStoreAdapter adapts package-level db helpers to core.Store for the TUI.
// Keep this adapter thin: direct delegation with minimal plumbing.
type tuiStoreAdapter struct{}

func (t *tuiStoreAdapter) GetAccounts() ([]model.Account, error) { return db.GetAllAccounts() }
func (t *tuiStoreAdapter) GetAllActiveAccounts() ([]model.Account, error) {
	return db.GetAllActiveAccounts()
}
func (t *tuiStoreAdapter) GetAllAccounts() ([]model.Account, error) { return db.GetAllAccounts() }
func (t *tuiStoreAdapter) GetAccount(id int) (*model.Account, error) {
	accts, err := db.GetAllAccounts()
	if err != nil {
		return nil, err
	}
	for _, a := range accts {
		if a.ID == id {
			aa := a
			return &aa, nil
		}
	}
	return nil, fmt.Errorf("account not found: %d", id)
}
func (t *tuiStoreAdapter) AddAccount(username, hostname, label, tags string) (int, error) {
	mgr := db.DefaultAccountManager()
	if mgr == nil {
		return 0, fmt.Errorf("no account manager available")
	}
	return mgr.AddAccount(username, hostname, label, tags)
}
func (t *tuiStoreAdapter) DeleteAccount(accountID int) error {
	mgr := db.DefaultAccountManager()
	if mgr == nil {
		return fmt.Errorf("no account manager available")
	}
	return mgr.DeleteAccount(accountID)
}
func (t *tuiStoreAdapter) AssignKeyToAccount(keyID, accountID int) error {
	km := db.DefaultKeyManager()
	if km == nil {
		return fmt.Errorf("no key manager available")
	}
	return km.AssignKeyToAccount(keyID, accountID)
}
func (t *tuiStoreAdapter) UpdateAccountIsDirty(id int, dirty bool) error {
	return db.UpdateAccountIsDirty(id, dirty)
}
func (t *tuiStoreAdapter) CreateSystemKey(publicKey, privateKey string) (int, error) {
	return db.CreateSystemKey(publicKey, privateKey)
}
func (t *tuiStoreAdapter) RotateSystemKey(publicKey, privateKey string) (int, error) {
	return db.RotateSystemKey(publicKey, privateKey)
}
func (t *tuiStoreAdapter) GetActiveSystemKey() (*model.SystemKey, error) {
	return db.GetActiveSystemKey()
}
func (t *tuiStoreAdapter) AddKnownHostKey(hostname, key string) error {
	return db.AddKnownHostKey(hostname, key)
}
func (t *tuiStoreAdapter) ExportDataForBackup() (*model.BackupData, error) {
	return db.ExportDataForBackup()
}
func (t *tuiStoreAdapter) ImportDataFromBackup(d *model.BackupData) error {
	return db.ImportDataFromBackup(d)
}
func (t *tuiStoreAdapter) IntegrateDataFromBackup(d *model.BackupData) error {
	return db.IntegrateDataFromBackup(d)
}

// FindByIdentifier mirrors the lookup helper behavior used by other adapters.
func (t *tuiStoreAdapter) FindByIdentifier(ctx context.Context, identifier string) (*model.Account, error) {
	if identifier == "" {
		return nil, fmt.Errorf("empty identifier")
	}
	// numeric id
	if id, err := strconv.Atoi(identifier); err == nil {
		if a, err := t.GetAccount(id); err == nil {
			return a, nil
		}
	}
	// user@host
	accts, err := db.GetAllAccounts()
	if err != nil {
		return nil, err
	}
	for _, a := range accts {
		if a.Username+"@"+a.Hostname == identifier {
			aa := a
			return &aa, nil
		}
	}
	return nil, fmt.Errorf("account not found: %s", identifier)
}

// SetAccountActiveState sets or clears the IsActive flag for an account.
func (t *tuiStoreAdapter) SetAccountActiveState(ctx context.Context, accountID int, active bool) error {
	accts, err := db.GetAllAccounts()
	if err != nil {
		return err
	}
	var found *model.Account
	for _, a := range accts {
		if a.ID == accountID {
			aa := a
			found = &aa
			break
		}
	}
	if found == nil {
		return fmt.Errorf("account not found: %d", accountID)
	}
	if found.IsActive == active {
		return nil
	}
	return db.ToggleAccountStatus(accountID)
}

// GenerateAuthorizedKeysContent builds authorized_keys content for an account.
func (t *tuiStoreAdapter) GenerateAuthorizedKeysContent(ctx context.Context, accountID int) (string, error) {
	sk, _ := db.GetActiveSystemKey()
	km := db.DefaultKeyManager()
	if km == nil {
		return "", fmt.Errorf("no key manager available")
	}
	gks, err := km.GetGlobalPublicKeys()
	if err != nil {
		return "", err
	}
	aks, err := km.GetKeysForAccount(accountID)
	if err != nil {
		return "", err
	}
	return keys.BuildAuthorizedKeysContent(sk, gks, aks)
}

// ensure tuiStoreAdapter satisfies core.Store at compile time
var _ core.Store = (*tuiStoreAdapter)(nil)

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
	StoreAdapter  = &tuiStoreAdapter{}
	AccountReader = &tuiAccountReader{}
	KeyReader     = &tuiKeyReader{}
	AuditReader   = &tuiAuditReader{}
	KeyLister     = &tuiKeyLister{}
)

// Exported helpers that mirror package-level db helpers but keep the runtime
// files free of direct `internal/db` imports. These are thin delegators.
func DefaultKeyManager() db.KeyManager           { return db.DefaultKeyManager() }
func DefaultKeySearcher() db.KeySearcher         { return db.DefaultKeySearcher() }
func ToggleAccountStatus(accountID int) error    { return db.ToggleAccountStatus(accountID) }
func DefaultAccountSearcher() db.AccountSearcher { return db.DefaultAccountSearcher() }
func DefaultAuditSearcher() db.AuditSearcher     { return db.DefaultAuditSearcher() }
func HasSystemKeys() (bool, error)               { return db.HasSystemKeys() }
