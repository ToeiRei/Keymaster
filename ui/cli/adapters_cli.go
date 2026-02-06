// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package cli

import (
	"fmt"

	log "github.com/charmbracelet/log"

	"github.com/toeirei/keymaster/core"
	crypto_ssh "github.com/toeirei/keymaster/core/crypto/ssh"
	"github.com/toeirei/keymaster/core/db"
	"github.com/toeirei/keymaster/core/model"
	"github.com/toeirei/keymaster/core/security"
)

// (cliStoreAdapter removed; CLI now uses uiadapters.NewStoreAdapter())

// cliDeployerManager adapts deploy package helpers to core.DeployerManager.
type cliDeployerManager struct{}

func (c *cliDeployerManager) DeployForAccount(account model.Account, keepFile bool) error {
	if core.DefaultDeployerManager == nil {
		return fmt.Errorf("no deployer manager available")
	}
	return core.DefaultDeployerManager.DeployForAccount(account, keepFile)
}
func (c *cliDeployerManager) AuditSerial(account model.Account) error {
	if core.DefaultDeployerManager == nil {
		return nil
	}
	return core.DefaultDeployerManager.AuditSerial(account)
}
func (c *cliDeployerManager) AuditStrict(account model.Account) error {
	if core.DefaultDeployerManager == nil {
		return nil
	}
	return core.DefaultDeployerManager.AuditStrict(account)
}
func (c *cliDeployerManager) DecommissionAccount(account model.Account, systemPrivateKey security.Secret, options interface{}) (core.DecommissionResult, error) {
	if core.DefaultDeployerManager == nil {
		return core.DecommissionResult{}, fmt.Errorf("no deployer manager available")
	}
	return core.DefaultDeployerManager.DecommissionAccount(account, systemPrivateKey, options)
}
func (c *cliDeployerManager) BulkDecommissionAccounts(accounts []model.Account, systemPrivateKey security.Secret, options interface{}) ([]core.DecommissionResult, error) {
	if core.DefaultDeployerManager == nil {
		return nil, fmt.Errorf("no deployer manager available")
	}
	return core.DefaultDeployerManager.BulkDecommissionAccounts(accounts, systemPrivateKey, options)
}
func (c *cliDeployerManager) CanonicalizeHostPort(host string) string {
	if core.DefaultDeployerManager == nil {
		return host
	}
	return core.DefaultDeployerManager.CanonicalizeHostPort(host)
}
func (c *cliDeployerManager) ParseHostPort(host string) (string, string, error) {
	if core.DefaultDeployerManager == nil {
		return host, "22", nil
	}
	return core.DefaultDeployerManager.ParseHostPort(host)
}
func (c *cliDeployerManager) GetRemoteHostKey(host string) (string, error) {
	if core.DefaultDeployerManager == nil {
		return "", nil
	}
	return core.DefaultDeployerManager.GetRemoteHostKey(host)
}

// FetchAuthorizedKeys fetches the raw authorized_keys content bytes for the account.
func (c *cliDeployerManager) FetchAuthorizedKeys(account model.Account) ([]byte, error) {
	if core.DefaultDeployerManager == nil {
		return nil, fmt.Errorf("no deployer manager available")
	}
	return core.DefaultDeployerManager.FetchAuthorizedKeys(account)
}

func (c *cliDeployerManager) ImportRemoteKeys(account model.Account) ([]model.PublicKey, int, string, error) {
	if core.DefaultDeployerManager == nil {
		return nil, 0, "", fmt.Errorf("no deployer manager available")
	}
	return core.DefaultDeployerManager.ImportRemoteKeys(account)
}

func (c *cliDeployerManager) IsPassphraseRequired(err error) bool {
	if core.DefaultDeployerManager == nil {
		return false
	}
	return core.DefaultDeployerManager.IsPassphraseRequired(err)
}

// cliDBMaintainer adapts db.RunDBMaintenance to core.DBMaintainer.
type cliDBMaintainer struct{}

func (c *cliDBMaintainer) RunDBMaintenance(dbType, dsn string) error {
	return db.RunDBMaintenance(dbType, dsn)
}

// cliStoreFactory creates a new store for migration targets via db.NewStoreFromDSN.
type cliStoreFactory struct{}

func (c *cliStoreFactory) NewStoreFromDSN(dbType, dsn string) (core.Store, error) {
	s, err := db.NewStoreFromDSN(dbType, dsn)
	if err != nil {
		return nil, err
	}
	// wrap the returned db.Store into a thin adapter that implements core.Store
	return &dbStoreWrapper{inner: s}, nil
}

// dbStoreWrapper adapts db.Store to core.Store for migration targets.
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
func (w *dbStoreWrapper) UpdateAccountIsDirty(id int, dirty bool) error {
	if w == nil || w.inner == nil {
		return fmt.Errorf("dbStoreWrapper: no inner store available")
	}
	return w.inner.UpdateAccountIsDirty(id, dirty)
}
func (w *dbStoreWrapper) CreateSystemKey(publicKey, privateKey string) (int, error) {
	if w == nil || w.inner == nil {
		return 0, fmt.Errorf("dbStoreWrapper: no inner store available")
	}
	return w.inner.CreateSystemKey(publicKey, privateKey)
}
func (w *dbStoreWrapper) RotateSystemKey(publicKey, privateKey string) (int, error) {
	if w == nil || w.inner == nil {
		return 0, fmt.Errorf("dbStoreWrapper: no inner store available")
	}
	return w.inner.RotateSystemKey(publicKey, privateKey)
}
func (w *dbStoreWrapper) GetActiveSystemKey() (*model.SystemKey, error) {
	if w == nil || w.inner == nil {
		return nil, fmt.Errorf("dbStoreWrapper: no inner store available")
	}
	return w.inner.GetActiveSystemKey()
}
func (w *dbStoreWrapper) AddKnownHostKey(hostname, key string) error {
	if w == nil || w.inner == nil {
		return fmt.Errorf("dbStoreWrapper: no inner store available")
	}
	return w.inner.AddKnownHostKey(hostname, key)
}
func (w *dbStoreWrapper) ExportDataForBackup() (*model.BackupData, error) {
	if w == nil || w.inner == nil {
		return nil, fmt.Errorf("dbStoreWrapper: no inner store available")
	}
	return w.inner.ExportDataForBackup()
}
func (w *dbStoreWrapper) ImportDataFromBackup(d *model.BackupData) error {
	if w == nil || w.inner == nil {
		return fmt.Errorf("dbStoreWrapper: no inner store available")
	}
	return w.inner.ImportDataFromBackup(d)
}
func (w *dbStoreWrapper) IntegrateDataFromBackup(d *model.BackupData) error {
	if w == nil || w.inner == nil {
		return fmt.Errorf("dbStoreWrapper: no inner store available")
	}
	return w.inner.IntegrateDataFromBackup(d)
}

// cliKeyGenerator delegates to the package-level generator function.
type cliKeyGenerator struct{}

func (c *cliKeyGenerator) GenerateAndMarshalEd25519Key(comment, passphrase string) (string, string, error) {
	return crypto_ssh.GenerateAndMarshalEd25519Key(comment, passphrase)
}

// ensure adapters satisfy core interfaces at compile time
var _ core.DBMaintainer = (*cliDBMaintainer)(nil)
var _ core.StoreFactory = (*cliStoreFactory)(nil)
var _ core.KeyGenerator = (*cliKeyGenerator)(nil)

// cliReporter implements core.Reporter by printing to stdout.
type cliReporter struct{}

func (r *cliReporter) Reportf(format string, args ...any) {
	log.Infof(format, args...)
}

var _ core.Reporter = (*cliReporter)(nil)

// cliReporter removed; pass nil Reporter where appropriate.

// cliAuditWriter removed; use db.DefaultAuditWriter() where needed.
