// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package deploy

import (
	"fmt"
	"time"

	"github.com/toeirei/keymaster/internal/core"
	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/model"
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

func init() {
	// NOTE: Deploy package init registers deploy-focused adapters into core.
	// These defaults are set for programs or tests that import
	// `internal/deploy` and expect deploy-specific behavior wired into
	// `internal/core`.
	//
	// Defaults registered by Deploy init():
	// - KeyReader (coreKeyReader)
	// - KeyLister (coreKeyLister)
	// - AccountSerialUpdater (accountSerialUpdater)
	// - KeyImporter (keyImporter)
	// - AuditWriter (coreAuditWriter)
	//
	// Subsystems depending on these: deploy logic, bootstrap helpers, and
	// tests that rely on deploy semantics.
	//
	// TODO: consider centralizing documentation about which package's init()
	// registers which `core` defaults to make it easier to reason about
	// global wiring and to avoid import-domain surprises.

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
