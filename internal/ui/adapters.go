// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package ui

import (
	"time"

	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/model"
)

// AccountManager provides a small interface for account add/delete operations
// used by UIs. This mirrors the minimal methods from `internal/db` but keeps
// callers decoupled from the DB package.
type AccountManager interface {
	AddAccount(username, hostname, label, tags string) (int, error)
	DeleteAccount(id int) error
}

// KeyManager provides management operations for public keys used by UIs.
type KeyManager interface {
	AddPublicKey(algorithm, keyData, comment string, isGlobal bool, expiresAt time.Time) error
	AddPublicKeyAndGetModel(algorithm, keyData, comment string, isGlobal bool, expiresAt time.Time) (*model.PublicKey, error)
	DeletePublicKey(id int) error
	TogglePublicKeyGlobal(id int) error
	SetPublicKeyExpiry(id int, expiresAt time.Time) error
	GetAllPublicKeys() ([]model.PublicKey, error)
	GetPublicKeyByComment(comment string) (*model.PublicKey, error)
	GetGlobalPublicKeys() ([]model.PublicKey, error)
	AssignKeyToAccount(keyID, accountID int) error
	UnassignKeyFromAccount(keyID, accountID int) error
	GetKeysForAccount(accountID int) ([]model.PublicKey, error)
	GetAccountsForKey(keyID int) ([]model.Account, error)
}

// dbAccountManager adapts a db.AccountManager to the ui.AccountManager interface.
type dbAccountManager struct{ inner db.AccountManager }

func (d *dbAccountManager) AddAccount(username, hostname, label, tags string) (int, error) {
	return d.inner.AddAccount(username, hostname, label, tags)
}

func (d *dbAccountManager) DeleteAccount(id int) error {
	return d.inner.DeleteAccount(id)
}

// dbKeyManager adapts a db.KeyManager to the ui.KeyManager interface.
type dbKeyManager struct{ inner db.KeyManager }

func (d *dbKeyManager) AddPublicKey(algorithm, keyData, comment string, isGlobal bool, expiresAt time.Time) error {
	return d.inner.AddPublicKey(algorithm, keyData, comment, isGlobal, expiresAt)
}

func (d *dbKeyManager) AddPublicKeyAndGetModel(algorithm, keyData, comment string, isGlobal bool, expiresAt time.Time) (*model.PublicKey, error) {
	return d.inner.AddPublicKeyAndGetModel(algorithm, keyData, comment, isGlobal, expiresAt)
}

func (d *dbKeyManager) DeletePublicKey(id int) error       { return d.inner.DeletePublicKey(id) }
func (d *dbKeyManager) TogglePublicKeyGlobal(id int) error { return d.inner.TogglePublicKeyGlobal(id) }
func (d *dbKeyManager) SetPublicKeyExpiry(id int, expiresAt time.Time) error {
	return d.inner.SetPublicKeyExpiry(id, expiresAt)
}
func (d *dbKeyManager) GetAllPublicKeys() ([]model.PublicKey, error) {
	return d.inner.GetAllPublicKeys()
}
func (d *dbKeyManager) GetPublicKeyByComment(comment string) (*model.PublicKey, error) {
	return d.inner.GetPublicKeyByComment(comment)
}
func (d *dbKeyManager) GetGlobalPublicKeys() ([]model.PublicKey, error) {
	return d.inner.GetGlobalPublicKeys()
}
func (d *dbKeyManager) AssignKeyToAccount(keyID, accountID int) error {
	return d.inner.AssignKeyToAccount(keyID, accountID)
}
func (d *dbKeyManager) UnassignKeyFromAccount(keyID, accountID int) error {
	return d.inner.UnassignKeyFromAccount(keyID, accountID)
}
func (d *dbKeyManager) GetKeysForAccount(accountID int) ([]model.PublicKey, error) {
	return d.inner.GetKeysForAccount(accountID)
}
func (d *dbKeyManager) GetAccountsForKey(keyID int) ([]model.Account, error) {
	return d.inner.GetAccountsForKey(keyID)
}

// DefaultAccountManager returns a ui.AccountManager backed by db.DefaultAccountManager().
// Returns nil if the db package does not provide a default manager.
func DefaultAccountManager() AccountManager {
	if am := db.DefaultAccountManager(); am != nil {
		return &dbAccountManager{inner: am}
	}
	return nil
}

// DefaultKeyManager returns a ui.KeyManager backed by db.DefaultKeyManager().
func DefaultKeyManager() KeyManager {
	if km := db.DefaultKeyManager(); km != nil {
		return &dbKeyManager{inner: km}
	}
	return nil
}

