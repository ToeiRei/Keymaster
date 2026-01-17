// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package tui

import (
	"context"
	"fmt"
	"time"

	"github.com/toeirei/keymaster/internal/config"
	"github.com/toeirei/keymaster/internal/core"
	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/keys"
	"github.com/toeirei/keymaster/internal/model"
	"github.com/toeirei/keymaster/internal/security"
	_ "github.com/toeirei/keymaster/internal/ui"
	"github.com/toeirei/keymaster/internal/uiadapters"
)

// coreAccountReader adapts UI helpers to core.AccountReader.
type coreAccountReader struct{}

func (coreAccountReader) GetAllAccounts() ([]model.Account, error) {
	// Delegate to canonical store adapter when available.
	if s := uiadapters.NewStoreAdapter(); s != nil {
		return s.GetAllAccounts()
	}
	return db.GetAllAccounts()
}

// coreKeyReader adapts UI key manager to core.KeyReader.
type coreKeyReader struct{}

func (coreKeyReader) GetAllPublicKeys() ([]model.PublicKey, error) {
	km := db.DefaultKeyManager()
	if km == nil {
		return nil, nil
	}
	return km.GetAllPublicKeys()
}

func (coreKeyReader) GetActiveSystemKey() (*model.SystemKey, error) { return db.GetActiveSystemKey() }

func (coreKeyReader) GetSystemKeyBySerial(serial int) (*model.SystemKey, error) {
	return db.GetSystemKeyBySerial(serial)
}

// coreAuditReader adapts UI audit helpers to core.AuditReader.
type coreAuditReader struct{}

func (coreAuditReader) GetAllAuditLogEntries() ([]model.AuditLogEntry, error) {
	return db.GetAllAuditLogEntries()
}

// coreAuditor adapts the TUI package-level audit helper to core.Auditor.
type coreAuditor struct{}

func (coreAuditor) LogAction(action, details string) error {
	return logAction(action, details)
}

// coreSystemKeyStore adapts UI system key helpers to core.SystemKeyStore.
type coreSystemKeyStore struct{}

func (coreSystemKeyStore) CreateSystemKey(publicKey, privateKey string) (int, error) {
	return db.CreateSystemKey(publicKey, privateKey)
}

func (coreSystemKeyStore) RotateSystemKey(publicKey, privateKey string) (int, error) {
	return db.RotateSystemKey(publicKey, privateKey)
}

// coreAccountStore adapts the UI account manager to core.AccountStore.
type coreAccountStore struct{}

func (coreAccountStore) AddAccount(username, hostname, label, tags string) (int, error) {
	mgr := db.DefaultAccountManager()
	if mgr == nil {
		return 0, fmt.Errorf("no account manager configured")
	}
	return mgr.AddAccount(username, hostname, label, tags)
}

func (coreAccountStore) DeleteAccount(accountID int) error {
	mgr := db.DefaultAccountManager()
	if mgr == nil {
		return fmt.Errorf("no account manager configured")
	}
	return mgr.DeleteAccount(accountID)
}

// coreKeyStore adapts the UI key manager to core.KeyStore.
type coreKeyStore struct{}

func (coreKeyStore) GetGlobalPublicKeys() ([]model.PublicKey, error) {
	km := db.DefaultKeyManager()
	if km == nil {
		return nil, fmt.Errorf("no key manager configured")
	}
	return km.GetGlobalPublicKeys()
}

func (coreKeyStore) GetKeysForAccount(accountID int) ([]model.PublicKey, error) {
	km := db.DefaultKeyManager()
	if km == nil {
		return nil, fmt.Errorf("no key manager configured")
	}
	return km.GetKeysForAccount(accountID)
}

func (coreKeyStore) AssignKeyToAccount(keyID, accountID int) error {
	km := db.DefaultKeyManager()
	if km == nil {
		return fmt.Errorf("no key manager configured")
	}
	return km.AssignKeyToAccount(keyID, accountID)
}

// coreKeysContentBuilder builds authorized_keys content using UI key manager
// and system key helpers.
type coreKeysContentBuilder struct{}

func (coreKeysContentBuilder) Generate(accountID int) (string, error) {
	// Use canonical store adapter's builder to keep logic centralized.
	if s := uiadapters.NewStoreAdapter(); s != nil {
		return s.GenerateAuthorizedKeysContent(context.Background(), accountID)
	}
	sk, _ := db.GetActiveSystemKey()
	km := db.DefaultKeyManager()
	if km == nil {
		return "", fmt.Errorf("no key manager available")
	}
	globalKeys, err := km.GetGlobalPublicKeys()
	if err != nil {
		return "", err
	}
	accountKeys, err := km.GetKeysForAccount(accountID)
	if err != nil {
		return "", err
	}
	return keys.BuildAuthorizedKeysContent(sk, globalKeys, accountKeys)
}

// accountSerialUpdater adapts UpdateAccountSerial to core.AccountSerialUpdater.
type accountSerialUpdater struct{}

func (accountSerialUpdater) UpdateAccountSerial(accountID int, serial int) error {
	return db.UpdateAccountSerial(accountID, serial)
}

// keyImporter adapts the db.KeyManager AddPublicKeyAndGetModel to core.KeyImporter.
type keyImporter struct{}

func (keyImporter) AddPublicKeyAndGetModel(algorithm, keyData, comment string, isGlobal bool, expiresAt time.Time) (*model.PublicKey, error) {
	km := db.DefaultKeyManager()
	if km == nil {
		return nil, fmt.Errorf("no KeyManager available")
	}
	return km.AddPublicKeyAndGetModel(algorithm, keyData, comment, isGlobal, expiresAt)
}

type coreKeyListerAdapter struct{}

func (coreKeyListerAdapter) GetGlobalPublicKeys() ([]model.PublicKey, error) {
	return coreKeyStore{}.GetGlobalPublicKeys()
}
func (coreKeyListerAdapter) GetKeysForAccount(accountID int) ([]model.PublicKey, error) {
	return coreKeyStore{}.GetKeysForAccount(accountID)
}
func (coreKeyListerAdapter) GetAllPublicKeys() ([]model.PublicKey, error) {
	return coreKeyReader{}.GetAllPublicKeys()
}

func init() {
	core.SetDefaultKeyReader(coreKeyReader{})
	core.SetDefaultKeyLister(coreKeyListerAdapter{})
	core.SetDefaultAccountSerialUpdater(accountSerialUpdater{})
	core.SetDefaultKeyImporter(keyImporter{})
	core.SetDefaultAuditWriter(coreAuditor{})
	core.SetDefaultAccountManager(coreAccountStore{})
	core.SetDefaultDBInit(func(dbType, dsn string) error {
		_, err := db.New(dbType, dsn)
		return err
	})
	core.SetDefaultDBIsInitialized(db.IsInitialized)
}

// coreBootstrapDeployerFactory adapts core bootstrap factory to a simple type used by TUI.
type coreBootstrapDeployerFactory struct{}

func (coreBootstrapDeployerFactory) New(hostname, username string, privateKey interface{}, expectedHostKey string) (core.BootstrapDeployer, error) {
	// Accept either a string or a security.Secret and normalize to security.Secret
	var sk security.Secret
	switch v := privateKey.(type) {
	case security.Secret:
		sk = v
	case string:
		sk = security.FromString(v)
	case []byte:
		sk = security.FromBytes(v)
	default:
		sk = nil
	}
	return core.NewBootstrapDeployer(hostname, username, sk, expectedHostKey)
}

// coreDeployAdapter delegates deploy-related operations to core.DefaultDeployerManager.
type coreDeployAdapter struct{}

func (coreDeployAdapter) GetRemoteHostKey(hostname string) (string, error) {
	return core.DefaultDeployerManager.GetRemoteHostKey(hostname)
}

func (coreDeployAdapter) CanonicalizeHostPort(host string) string {
	return core.DefaultDeployerManager.CanonicalizeHostPort(host)
}

func (coreDeployAdapter) ParseHostPort(host string) (string, string, error) {
	return core.DefaultDeployerManager.ParseHostPort(host)
}

// FetchAuthorizedKeys returns the raw authorized_keys content from the remote host.
func (coreDeployAdapter) FetchAuthorizedKeys(account model.Account) ([]byte, error) {
	return core.DefaultDeployerManager.FetchAuthorizedKeys(account)
}

func (coreDeployAdapter) ImportRemoteKeys(account model.Account) ([]model.PublicKey, int, string, error) {
	return core.DefaultDeployerManager.ImportRemoteKeys(account)
}

func (coreDeployAdapter) DecommissionAccount(account model.Account, systemKey security.Secret, options interface{}) (core.DecommissionResult, error) {
	return core.DefaultDeployerManager.DecommissionAccount(account, systemKey, options)
}

func (coreDeployAdapter) DeployForAccount(account model.Account, keepFile bool) error {
	return core.DefaultDeployerManager.DeployForAccount(account, keepFile)
}

func (coreDeployAdapter) RunDeploymentForAccount(account model.Account, isTUI bool) error {
	// Keep the same semantic: TUI uses interactive mode.
	return core.DefaultDeployerManager.DeployForAccount(account, isTUI)
}

func (coreDeployAdapter) AuditSerial(account model.Account) error {
	return core.DefaultDeployerManager.AuditSerial(account)
}
func (coreDeployAdapter) AuditStrict(account model.Account) error {
	return core.DefaultDeployerManager.AuditStrict(account)
}

func (coreDeployAdapter) BulkDecommissionAccounts(accounts []model.Account, systemKey security.Secret, options interface{}) ([]core.DecommissionResult, error) {
	return core.DefaultDeployerManager.BulkDecommissionAccounts(accounts, systemKey, options)
}

func (coreDeployAdapter) IsPassphraseRequired(err error) bool {
	return core.DefaultDeployerManager.IsPassphraseRequired(err)
}

var deployAdapter = coreDeployAdapter{}

// coreSessionStore adapts UI session helpers to core.SessionStore.
type coreSessionStore struct{}

func (coreSessionStore) SaveBootstrapSession(id, username, hostname, label, tags, tempPublicKey string, expiresAt time.Time, status string) error {
	return db.SaveBootstrapSession(id, username, hostname, label, tags, tempPublicKey, expiresAt, status)
}

func (coreSessionStore) GetBootstrapSession(id string) (*model.BootstrapSession, error) {
	return db.GetBootstrapSession(id)
}

func (coreSessionStore) DeleteBootstrapSession(id string) error {
	return db.DeleteBootstrapSession(id)
}

func (coreSessionStore) UpdateBootstrapSessionStatus(id string, status string) error {
	return db.UpdateBootstrapSessionStatus(id, status)
}

func (coreSessionStore) GetExpiredBootstrapSessions() ([]*model.BootstrapSession, error) {
	return db.GetExpiredBootstrapSessions()
}

func (coreSessionStore) GetOrphanedBootstrapSessions() ([]*model.BootstrapSession, error) {
	return db.GetOrphanedBootstrapSessions()
}

// coreConfigSaver adapts package-level config persistence to a small adapter
// used by the TUI so core and UI code don't directly call the config package.
type coreConfigSaver struct{}

func (coreConfigSaver) Save() error {
	return config.Save()
}

var configSaver = coreConfigSaver{}
