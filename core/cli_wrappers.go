// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package core

import (
	"context"
	"fmt"

	sshgen "github.com/toeirei/keymaster/core/crypto/ssh"
	"github.com/toeirei/keymaster/core/db"
	"github.com/toeirei/keymaster/core/model"
)

// DefaultKeyManager returns a KeyManager that delegates to the DB layer's
// package-level default. This lets UI code depend on `core` rather than
// importing `core/db` directly.
func DefaultKeyManager() KeyManager {
	km := db.DefaultKeyManager()
	if km == nil {
		return nil
	}
	return km
}

// GetAccountKeyHash reads the raw key_hash column for an account. This is a
// small convenience to avoid UIs importing db helpers directly.
func GetAccountKeyHash(accountID int) (string, error) {
	var stored string
	if err := db.QueryRawInto(context.Background(), db.BunDB(), &stored, "SELECT key_hash FROM accounts WHERE id = ?", accountID); err != nil {
		return "", err
	}
	return stored, nil
}

// InitDB initializes the package-level store via db.New. UI layers may call
// this convenience to avoid importing core/db directly.
func InitDB(dbType, dsn string) error {
	_, err := db.New(dbType, dsn)
	return err
}

// SetDBDebug toggles DB debug logging.
func SetDBDebug(enabled bool) { db.SetDebug(enabled) }

// NewStoreFromDSN creates a new store from DSN and returns a core.Store by
// wrapping the returned db.Store. This is a convenience for UIs.
func NewStoreFromDSN(dbType, dsn string) (Store, error) {
	s, err := db.NewStoreFromDSN(dbType, dsn)
	if err != nil {
		return nil, err
	}
	return &dbStoreWrapper{inner: s}, nil
}

// ResetStoreForTests closes and clears the package-level DB store. Tests may
// call this via the `core` package to avoid importing `core/db` directly.
func ResetStoreForTests() { db.ResetStoreForTests() }

// Convenience wrappers for commonly used DB helpers so UIs/tests can call
// into `core` instead of importing `core/db` directly.
func GetKnownHostKey(hostname string) (string, error) { return db.GetKnownHostKey(hostname) }
func GetActiveSystemKey() (*model.SystemKey, error)   { return db.GetActiveSystemKey() }
func GetSystemKeyBySerial(serial int) (*model.SystemKey, error) {
	return db.GetSystemKeyBySerial(serial)
}
func UpdateAccountSerial(accountID int, serial int) error {
	return db.UpdateAccountSerial(accountID, serial)
}
func GetAllAccounts() ([]model.Account, error)       { return db.GetAllAccounts() }
func ToggleAccountStatus(accountID int) error        { return db.ToggleAccountStatus(accountID) }
func GetAllActiveAccounts() ([]model.Account, error) { return db.GetAllActiveAccounts() }

// DefaultDBMaintainer returns a DBMaintainer implementation that delegates
// to the db package's RunDBMaintenance helper.
type dbMaintainer struct{}

func (d dbMaintainer) RunDBMaintenance(dbType, dsn string) error {
	return db.RunDBMaintenance(dbType, dsn)
}

func DefaultDBMaintainer() DBMaintainer { return dbMaintainer{} }

// DefaultKeyGenerator returns a KeyGenerator backed by the core/crypto/ssh
// package.
type sshKeyGen struct{}

func (s sshKeyGen) GenerateAndMarshalEd25519Key(comment, passphrase string) (string, string, error) {
	return sshgen.GenerateAndMarshalEd25519Key(comment, passphrase)
}

func DefaultKeyGenerator() KeyGenerator { return sshKeyGen{} }

// SetDefaultKeyManager registers a package-level KeyManager implementation
// for tests and initialization code. This delegates to the DB layer's
// SetDefaultKeyManager so callers don't need to import `core/db`.
func SetDefaultKeyManager(m KeyManager) { db.SetDefaultKeyManager(m) }

// (SetDefaultAccountManager is implemented in defaults_db.go and also
// delegates to the DB package; no duplicate implementation here.)

// CloseStore attempts to close resources held by a store created via
// NewStoreFromDSN. This helps tests clean up in-memory or temp-file DBs when
// they create ad-hoc stores for assertions.
func CloseStore(s Store) error {
	if s == nil {
		return nil
	}
	if w, ok := s.(*dbStoreWrapper); ok {
		if w.inner == nil {
			return nil
		}
		if closer, ok := interface{}(w.inner).(interface{ Close() error }); ok {
			return closer.Close()
		}
	}
	return nil
}

// dbStoreWrapper adapts db.Store to core.Store.
type dbStoreWrapper struct{ inner db.Store }

func (w *dbStoreWrapper) GetAccounts() ([]model.Account, error) { return w.inner.GetAllAccounts() }
func (w *dbStoreWrapper) GetAllActiveAccounts() ([]model.Account, error) {
	if w == nil || w.inner == nil {
		return nil, fmt.Errorf("dbStoreWrapper: no inner store available")
	}
	return w.inner.GetAllActiveAccounts()
}
func (w *dbStoreWrapper) GetAllAccounts() ([]model.Account, error) { return w.inner.GetAllAccounts() }
func (w *dbStoreWrapper) GetAccount(id int) (*model.Account, error) {
	if w == nil || w.inner == nil {
		return nil, fmt.Errorf("dbStoreWrapper: no inner store available")
	}
	ac, err := w.inner.GetAllAccounts()
	if err != nil {
		return nil, err
	}
	for _, a := range ac {
		if a.ID == id {
			aa := a
			return &aa, nil
		}
	}
	return nil, fmt.Errorf("not found")
}
func (w *dbStoreWrapper) AddAccount(username, hostname, label, tags string) (int, error) {
	if w == nil || w.inner == nil {
		return 0, fmt.Errorf("dbStoreWrapper: no inner store available")
	}
	return w.inner.AddAccount(username, hostname, label, tags)
}
func (w *dbStoreWrapper) DeleteAccount(accountID int) error { return w.inner.DeleteAccount(accountID) }
func (w *dbStoreWrapper) AssignKeyToAccount(keyID, accountID int) error {
	km := db.DefaultKeyManager()
	if km == nil {
		return fmt.Errorf("no key manager available")
	}
	return km.AssignKeyToAccount(keyID, accountID)
}
func (w *dbStoreWrapper) ToggleAccountStatus(accountID int, enabled bool) error {
	if w == nil || w.inner == nil {
		return fmt.Errorf("dbStoreWrapper: no inner store available")
	}
	return w.inner.ToggleAccountStatus(accountID, enabled)
}
func (w *dbStoreWrapper) UpdateAccountHostname(accountID int, hostname string) error {
	return w.inner.UpdateAccountHostname(accountID, hostname)
}
func (w *dbStoreWrapper) UpdateAccountLabel(accountID int, label string) error {
	return w.inner.UpdateAccountLabel(accountID, label)
}
func (w *dbStoreWrapper) UpdateAccountTags(accountID int, tags string) error {
	return w.inner.UpdateAccountTags(accountID, tags)
}
func (w *dbStoreWrapper) UpdateAccountIsDirty(id int, dirty bool) error {
	return w.inner.UpdateAccountIsDirty(id, dirty)
}
func (w *dbStoreWrapper) CreateSystemKey(publicKey, privateKey string) (int, error) {
	return w.inner.CreateSystemKey(publicKey, privateKey)
}
func (w *dbStoreWrapper) RotateSystemKey(publicKey, privateKey string) (int, error) {
	return w.inner.RotateSystemKey(publicKey, privateKey)
}
func (w *dbStoreWrapper) GetActiveSystemKey() (*model.SystemKey, error) {
	return w.inner.GetActiveSystemKey()
}
func (w *dbStoreWrapper) AddKnownHostKey(hostname, key string) error {
	return w.inner.AddKnownHostKey(hostname, key)
}
func (w *dbStoreWrapper) ExportDataForBackup() (*model.BackupData, error) {
	return w.inner.ExportDataForBackup()
}
func (w *dbStoreWrapper) ImportDataFromBackup(d *model.BackupData) error {
	return w.inner.ImportDataFromBackup(d)
}
func (w *dbStoreWrapper) IntegrateDataFromBackup(d *model.BackupData) error {
	return w.inner.IntegrateDataFromBackup(d)
}
