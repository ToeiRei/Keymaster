// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package uiadapters

import (
	"context"
	"fmt"
	"strconv"

	"github.com/toeirei/keymaster/internal/core"
	"github.com/toeirei/keymaster/internal/core/db"
	"github.com/toeirei/keymaster/internal/keys"
	"github.com/toeirei/keymaster/internal/logging"
	"github.com/toeirei/keymaster/internal/model"
)

// Package uiadapters provides thin, canonical adapters that adapt package-level
// `internal/db` helpers to `core` interfaces used by UI layers (TUI/CLI/UI).
//
// Notes:
// - These adapters are intentionally thin delegators: behavior remains in
//   `internal/db` and other authoritative packages.
// - This file contains no `init()` registrations and introduces no global
//   state. It is considered low-risk to update for clarity; avoid changing
//   exported signatures without explicit approval.

// uiStoreAdapter is a canonical, thin adapter that adapts package-level db
// helpers to the `core.Store` interface. This file intentionally mirrors the
// existing CLI/TUI adapters without behavioral changes so UIs can migrate to a
// single shared implementation.
type storeAdapter struct{}

func (s *storeAdapter) GetAccounts() ([]model.Account, error) { return db.GetAllAccounts() }
func (s *storeAdapter) GetAllActiveAccounts() ([]model.Account, error) {
	return db.GetAllActiveAccounts()
}
func (s *storeAdapter) GetAllAccounts() ([]model.Account, error) { return db.GetAllAccounts() }
func (s *storeAdapter) GetAccount(id int) (*model.Account, error) {
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
func (s *storeAdapter) AddAccount(username, hostname, label, tags string) (int, error) {
	mgr := db.DefaultAccountManager()
	if mgr == nil {
		return 0, fmt.Errorf("no account manager available")
	}
	return mgr.AddAccount(username, hostname, label, tags)
}
func (s *storeAdapter) DeleteAccount(accountID int) error {
	mgr := db.DefaultAccountManager()
	if mgr == nil {
		return fmt.Errorf("no account manager available")
	}
	return mgr.DeleteAccount(accountID)
}
func (s *storeAdapter) AssignKeyToAccount(keyID, accountID int) error {
	km := db.DefaultKeyManager()
	if km == nil {
		return fmt.Errorf("no key manager available")
	}
	return km.AssignKeyToAccount(keyID, accountID)
}
func (s *storeAdapter) UpdateAccountIsDirty(id int, dirty bool) error {
	return db.UpdateAccountIsDirty(id, dirty)
}
func (s *storeAdapter) CreateSystemKey(publicKey, privateKey string) (int, error) {
	return db.CreateSystemKey(publicKey, privateKey)
}
func (s *storeAdapter) RotateSystemKey(publicKey, privateKey string) (int, error) {
	return db.RotateSystemKey(publicKey, privateKey)
}
func (s *storeAdapter) GetActiveSystemKey() (*model.SystemKey, error) {
	return db.GetActiveSystemKey()
}
func (s *storeAdapter) AddKnownHostKey(hostname, key string) error {
	return db.AddKnownHostKey(hostname, key)
}
func (s *storeAdapter) ExportDataForBackup() (*model.BackupData, error) {
	return db.ExportDataForBackup()
}
func (s *storeAdapter) ImportDataFromBackup(d *model.BackupData) error {
	return db.ImportDataFromBackup(d)
}
func (s *storeAdapter) IntegrateDataFromBackup(d *model.BackupData) error {
	return db.IntegrateDataFromBackup(d)
}

// FindByIdentifier mirrors existing logic used in other adapters.
func (s *storeAdapter) FindByIdentifier(ctx context.Context, identifier string) (*model.Account, error) {
	if identifier == "" {
		return nil, fmt.Errorf("empty identifier")
	}
	// numeric id
	if id, err := strconv.Atoi(identifier); err == nil {
		if a, err := s.GetAccount(id); err == nil {
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
func (s *storeAdapter) SetAccountActiveState(ctx context.Context, accountID int, active bool) error {
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
func (s *storeAdapter) GenerateAuthorizedKeysContent(ctx context.Context, accountID int) (string, error) {
	// Note: This builds authorized_keys content by combining the active
	// system key, global keys, and account keys via `keys.BuildAuthorizedKeysContent`.
	// Any future refactor that changes the content format must be validated
	// against existing deployments and tests.
	sk, _ := db.GetActiveSystemKey()
	if sk == nil {
		// Warn but continue: existing callers expect content generation to
		// succeed even when no system key is present (e.g., during bootstrap).
		logging.Warnf("GenerateAuthorizedKeysContent: no active system key found for account %d", accountID)
	}
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
	// Centralize call via a single unexported helper to make it easier to
	// extend or variant-implement the authorized_keys generation in one
	// place without duplicating call sites. This is a mechanical consolidation
	// only and does not change behavior.
	return s.buildAuthorizedKeysContent(sk, gks, aks)
}

// buildAuthorizedKeysContent centralizes the call to `keys.BuildAuthorizedKeysContent`.
// Keep this unexported wrapper to provide a single place to add future
// variants (headers, signing metadata) without changing the higher-level
// control flow in `GenerateAuthorizedKeysContent`.
func (s *storeAdapter) buildAuthorizedKeysContent(sk *model.SystemKey, gks, aks []model.PublicKey) (string, error) {
	return keys.BuildAuthorizedKeysContent(sk, gks, aks)
}

// ensure uiStoreAdapter satisfies core.Store at compile time
var _ core.Store = (*storeAdapter)(nil)

// NewStoreAdapter returns a new thin adapter implementing `core.Store` that
// delegates to package-level `internal/db` helpers. Consumers should call the
// constructor and use the returned `core.Store` rather than relying on a
// package-level instance.
func NewStoreAdapter() *storeAdapter { return &storeAdapter{} }
