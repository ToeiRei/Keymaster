// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package deploy

import (
	"fmt"
	"time"

	"github.com/toeirei/keymaster/internal/core"
	"github.com/toeirei/keymaster/internal/core/db"
	"github.com/toeirei/keymaster/internal/model"
)

// Compile-time interface checks
var (
	_ core.KeyLister            = (*coreKeyLister)(nil)        // coreKeyLister implements core.KeyLister
	_ core.AccountSerialUpdater = (*accountSerialUpdater)(nil) // accountSerialUpdater implements core.AccountSerialUpdater
	_ core.KeyImporter          = (*keyImporter)(nil)          // keyImporter implements core.KeyImporter
	_ core.AccountManager       = (*coreAccountManager)(nil)   // coreAccountManager implements core.AccountManager
	_ core.AuditWriter          = (*coreAuditWriter)(nil)      // coreAuditWriter implements core.AuditWriter
)

// Wire DB-backed adapters into core defaults for packages that import
// internal/deploy (many programs/tests import deploy but may not import UI packages).

type coreKeyReader struct{}

func (coreKeyReader) GetActiveSystemKey() (*model.SystemKey, error) { return db.GetActiveSystemKey() }
func (coreKeyReader) GetSystemKeyBySerial(serial int) (*model.SystemKey, error) {
	return db.GetSystemKeyBySerial(serial)
}
func (coreKeyReader) GetAllPublicKeys() ([]model.PublicKey, error) {
	if km := db.DefaultKeyManager(); km != nil {
		return km.GetAllPublicKeys()
	}
	return nil, fmt.Errorf("no key manager available")
}

type coreKeyLister struct{}

func (coreKeyLister) GetGlobalPublicKeys() ([]model.PublicKey, error) {
	if km := db.DefaultKeyManager(); km != nil {
		return km.GetGlobalPublicKeys()
	}
	return nil, fmt.Errorf("no key manager available")
}
func (coreKeyLister) GetKeysForAccount(accountID int) ([]model.PublicKey, error) {
	if km := db.DefaultKeyManager(); km != nil {
		return km.GetKeysForAccount(accountID)
	}
	return nil, fmt.Errorf("no key manager available")
}
func (coreKeyLister) GetAllPublicKeys() ([]model.PublicKey, error) {
	return coreKeyReader{}.GetAllPublicKeys()
}

type accountSerialUpdater struct{}

func (accountSerialUpdater) UpdateAccountSerial(accountID int, serial int) error {
	return db.UpdateAccountSerial(accountID, serial)
}

type keyImporter struct{}

func (keyImporter) AddPublicKeyAndGetModel(algorithm, keyData, comment string, isGlobal bool, expiresAt time.Time) (*model.PublicKey, error) {
	if km := db.DefaultKeyManager(); km != nil {
		return km.AddPublicKeyAndGetModel(algorithm, keyData, comment, isGlobal, expiresAt)
	}
	return nil, fmt.Errorf("no key manager available")
}

type coreAuditWriter struct{}

func (coreAuditWriter) LogAction(action, details string) error {
	if w := db.DefaultAuditWriter(); w != nil {
		return w.LogAction(action, details)
	}
	return nil
}

type coreAccountManager struct{}

func (coreAccountManager) DeleteAccount(id int) error {
	if m := db.DefaultAccountManager(); m != nil {
		return m.DeleteAccount(id)
	}
	return fmt.Errorf("no account manager available")
}

// InitializeDefaults registers deploy-specific default implementations into
// `internal/core`. This makes wiring explicit for tests and allows callers to
// opt-in to deploy wiring without relying on package init.
//
// InitializeDefaults is safe to call multiple times.
func InitializeDefaults() {
	core.SetDefaultKeyReader(coreKeyReader{})
	core.SetDefaultKeyLister(coreKeyLister{})
	core.SetDefaultAccountSerialUpdater(accountSerialUpdater{})
	core.SetDefaultKeyImporter(keyImporter{})
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
