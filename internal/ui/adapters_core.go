// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package ui

import (
	"fmt"

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
