// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package ui

import (
	"fmt"
	"time"

	"github.com/toeirei/keymaster/internal/core"
	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/model"
)

// coreKeyReader adapts package-level DB helpers to core.KeyReader.
type coreKeyReader struct{}

func (coreKeyReader) GetActiveSystemKey() (*model.SystemKey, error) {
	return db.GetActiveSystemKey()
}

func (coreKeyReader) GetSystemKeyBySerial(serial int) (*model.SystemKey, error) {
	return db.GetSystemKeyBySerial(serial)
}

func (coreKeyReader) GetAllPublicKeys() ([]model.PublicKey, error) {
	km := db.DefaultKeyManager()
	if km == nil {
		return nil, fmt.Errorf("no KeyManager available")
	}
	return km.GetAllPublicKeys()
}

// DefaultCoreKeyReader returns a core.KeyReader that delegates to `internal/db`.
func DefaultCoreKeyReader() core.KeyReader { return coreKeyReader{} }

// coreKeyLister adapts KeyManager to core.KeyLister.
type coreKeyLister struct{}

func (coreKeyLister) GetGlobalPublicKeys() ([]model.PublicKey, error) {
	km := db.DefaultKeyManager()
	if km == nil {
		return nil, fmt.Errorf("no KeyManager available")
	}
	return km.GetGlobalPublicKeys()
}

func (coreKeyLister) GetKeysForAccount(accountID int) ([]model.PublicKey, error) {
	km := db.DefaultKeyManager()
	if km == nil {
		return nil, fmt.Errorf("no KeyManager available")
	}
	return km.GetKeysForAccount(accountID)
}

func (coreKeyLister) GetAllPublicKeys() ([]model.PublicKey, error) {
	km := db.DefaultKeyManager()
	if km == nil {
		return nil, fmt.Errorf("no KeyManager available")
	}
	return km.GetAllPublicKeys()
}

// DefaultCoreKeyLister returns a core.KeyLister backed by the DB.
func DefaultCoreKeyLister() core.KeyLister { return coreKeyLister{} }

// accountSerialUpdater adapts UpdateAccountSerial to core.AccountSerialUpdater.
type accountSerialUpdater struct{}

func (accountSerialUpdater) UpdateAccountSerial(accountID int, serial int) error {
	return db.UpdateAccountSerial(accountID, serial)
}

func DefaultAccountSerialUpdater() core.AccountSerialUpdater { return accountSerialUpdater{} }

// coreKeyImporter adapts DB key manager to core.KeyImporter.
type coreKeyImporter struct{}

func (coreKeyImporter) AddPublicKeyAndGetModel(algorithm, keyData, comment string, isGlobal bool, expiresAt time.Time) (*model.PublicKey, error) {
	km := db.DefaultKeyManager()
	if km == nil {
		return nil, fmt.Errorf("no KeyManager available")
	}
	return km.AddPublicKeyAndGetModel(algorithm, keyData, comment, isGlobal, expiresAt)
}

// coreAuditWriter adapts DB audit writer to core.AuditWriter.
type coreAuditWriter struct{}

func (coreAuditWriter) LogAction(action, details string) error {
	if w := db.DefaultAuditWriter(); w != nil {
		return w.LogAction(action, details)
	}
	return nil
}

// coreAccountManager adapts DB account manager to core.AccountManager.
type coreAccountManager struct{}

func (coreAccountManager) DeleteAccount(id int) error {
	if m := db.DefaultAccountManager(); m != nil {
		return m.DeleteAccount(id)
	}
	return fmt.Errorf("no account manager available")
}

// InitializeDefaults registers UI-facing default implementations into
// `internal/core`. This function makes the wiring explicit so callers and
// tests can choose to invoke it directly. Calling it from package `init()`
// preserves existing implicit behavior.
//
// InitializeDefaults is safe to call multiple times.
func InitializeDefaults() {
	core.SetDefaultKeyReader(DefaultCoreKeyReader())
	core.SetDefaultKeyLister(DefaultCoreKeyLister())
	core.SetDefaultAccountSerialUpdater(DefaultAccountSerialUpdater())
	core.SetDefaultKeyImporter(coreKeyImporter{})
	core.SetDefaultAuditWriter(coreAuditWriter{})
	core.SetDefaultAccountManager(coreAccountManager{})
	core.SetDefaultDBInit(func(dbType, dsn string) error {
		_, err := db.New(dbType, dsn)
		return err
	})
	core.SetDefaultDBIsInitialized(db.IsInitialized)
}

// Preserve existing init-time wiring for backward compatibility.
func init() {
	InitializeDefaults()
}
