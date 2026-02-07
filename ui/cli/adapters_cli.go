// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package cli

import (
	"fmt"

	log "github.com/charmbracelet/log"

	"github.com/toeirei/keymaster/core"
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
	return core.DefaultDBMaintainer().RunDBMaintenance(dbType, dsn)
}

// cliStoreFactory creates a new store for migration targets via db.NewStoreFromDSN.
type cliStoreFactory struct{}

func (c *cliStoreFactory) NewStoreFromDSN(dbType, dsn string) (core.Store, error) {
	return core.NewStoreFromDSN(dbType, dsn)
}

// dbStoreWrapper adapts db.Store to core.Store for migration targets.
// dbStoreWrapper moved into core.NewStoreFromDSN wrapper; no local wrapper needed.

// cliKeyGenerator delegates to the package-level generator function.
type cliKeyGenerator struct{}

func (c *cliKeyGenerator) GenerateAndMarshalEd25519Key(comment, passphrase string) (string, string, error) {
	return core.DefaultKeyGenerator().GenerateAndMarshalEd25519Key(comment, passphrase)
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
