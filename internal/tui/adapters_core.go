package tui

import (
	"errors"
	"fmt"
	"time"

	"github.com/toeirei/keymaster/internal/config"
	"github.com/toeirei/keymaster/internal/core"
	"github.com/toeirei/keymaster/internal/deploy"
	"github.com/toeirei/keymaster/internal/keys"
	"github.com/toeirei/keymaster/internal/model"
	"github.com/toeirei/keymaster/internal/ui"
	"golang.org/x/crypto/ssh"
)

// coreAccountReader adapts UI helpers to core.AccountReader.
type coreAccountReader struct{}

func (coreAccountReader) GetAllAccounts() ([]model.Account, error) { return ui.GetAllAccounts() }

// coreKeyReader adapts UI key manager to core.KeyReader.
type coreKeyReader struct{}

func (coreKeyReader) GetAllPublicKeys() ([]model.PublicKey, error) {
	km := ui.DefaultKeyManager()
	if km == nil {
		return nil, nil
	}
	return km.GetAllPublicKeys()
}

func (coreKeyReader) GetActiveSystemKey() (*model.SystemKey, error) { return ui.GetActiveSystemKey() }

// coreAuditReader adapts UI audit helpers to core.AuditReader.
type coreAuditReader struct{}

func (coreAuditReader) GetAllAuditLogEntries() ([]model.AuditLogEntry, error) {
	return ui.GetAllAuditLogEntries()
}

// coreAuditor adapts the TUI package-level audit helper to core.Auditor.
type coreAuditor struct{}

func (coreAuditor) LogAction(action, details string) error {
	return logAction(action, details)
}

// coreSystemKeyStore adapts UI system key helpers to core.SystemKeyStore.
type coreSystemKeyStore struct{}

func (coreSystemKeyStore) CreateSystemKey(publicKey, privateKey string) (int, error) {
	return ui.CreateSystemKey(publicKey, privateKey)
}

func (coreSystemKeyStore) RotateSystemKey(publicKey, privateKey string) (int, error) {
	return ui.RotateSystemKey(publicKey, privateKey)
}

// coreAccountStore adapts the UI account manager to core.AccountStore.
type coreAccountStore struct{}

func (coreAccountStore) AddAccount(username, hostname, label, tags string) (int, error) {
	mgr := ui.DefaultAccountManager()
	if mgr == nil {
		return 0, fmt.Errorf("no account manager configured")
	}
	return mgr.AddAccount(username, hostname, label, tags)
}

func (coreAccountStore) DeleteAccount(accountID int) error {
	mgr := ui.DefaultAccountManager()
	if mgr == nil {
		return fmt.Errorf("no account manager configured")
	}
	return mgr.DeleteAccount(accountID)
}

// coreKeyStore adapts the UI key manager to core.KeyStore.
type coreKeyStore struct{}

func (coreKeyStore) GetGlobalPublicKeys() ([]model.PublicKey, error) {
	km := ui.DefaultKeyManager()
	if km == nil {
		return nil, fmt.Errorf("no key manager configured")
	}
	return km.GetGlobalPublicKeys()
}

func (coreKeyStore) GetKeysForAccount(accountID int) ([]model.PublicKey, error) {
	km := ui.DefaultKeyManager()
	if km == nil {
		return nil, fmt.Errorf("no key manager configured")
	}
	return km.GetKeysForAccount(accountID)
}

func (coreKeyStore) AssignKeyToAccount(keyID, accountID int) error {
	km := ui.DefaultKeyManager()
	if km == nil {
		return fmt.Errorf("no key manager configured")
	}
	return km.AssignKeyToAccount(keyID, accountID)
}

// coreKeysContentBuilder builds authorized_keys content using UI key manager
// and system key helpers.
type coreKeysContentBuilder struct{}

func (coreKeysContentBuilder) Generate(accountID int) (string, error) {
	sk, _ := ui.GetActiveSystemKey()
	km := ui.DefaultKeyManager()
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

// coreBootstrapDeployerFactory adapts the deploy package to core.NewBootstrapDeployer.
type coreBootstrapDeployerFactory struct{}

func (coreBootstrapDeployerFactory) New(hostname, username, privateKey, expectedHostKey string) (core.BootstrapDeployer, error) {
	d, err := deploy.NewBootstrapDeployerWithExpectedKey(hostname, username, privateKey, expectedHostKey)
	if err != nil {
		return nil, err
	}
	return d, nil
}

// coreDeployAdapter wraps the deploy package for use by the TUI.
type coreDeployAdapter struct{}

func (coreDeployAdapter) GetRemoteHostKey(hostname string) (ssh.PublicKey, error) {
	return deploy.GetRemoteHostKey(hostname)
}

func (coreDeployAdapter) ImportRemoteKeys(account model.Account) ([]model.PublicKey, int, string, error) {
	return deploy.ImportRemoteKeys(account)
}

func (coreDeployAdapter) DecommissionAccount(account model.Account, systemKey string, options deploy.DecommissionOptions) (deploy.DecommissionResult, error) {
	res := deploy.DecommissionAccount(account, systemKey, options)
	return res, nil
}

func (coreDeployAdapter) RunDeploymentForAccount(account model.Account, isTUI bool) error {
	return deploy.RunDeploymentForAccount(account, isTUI)
}

func (coreDeployAdapter) IsPassphraseRequired(err error) bool {
	return errors.Is(err, deploy.ErrPassphraseRequired)
}

var deployAdapter = coreDeployAdapter{}

// coreSessionStore adapts UI session helpers to core.SessionStore.
type coreSessionStore struct{}

func (coreSessionStore) SaveBootstrapSession(id, username, hostname, label, tags, tempPublicKey string, expiresAt time.Time, status string) error {
	return ui.SaveBootstrapSession(id, username, hostname, label, tags, tempPublicKey, expiresAt, status)
}

func (coreSessionStore) GetBootstrapSession(id string) (*model.BootstrapSession, error) {
	return ui.GetBootstrapSession(id)
}

func (coreSessionStore) DeleteBootstrapSession(id string) error {
	return ui.DeleteBootstrapSession(id)
}

func (coreSessionStore) UpdateBootstrapSessionStatus(id string, status string) error {
	return ui.UpdateBootstrapSessionStatus(id, status)
}

func (coreSessionStore) GetExpiredBootstrapSessions() ([]*model.BootstrapSession, error) {
	return ui.GetExpiredBootstrapSessions()
}

func (coreSessionStore) GetOrphanedBootstrapSessions() ([]*model.BootstrapSession, error) {
	return ui.GetOrphanedBootstrapSessions()
}

// coreConfigSaver adapts package-level config persistence to a small adapter
// used by the TUI so core and UI code don't directly call the config package.
type coreConfigSaver struct{}

func (coreConfigSaver) Save() error {
	return config.Save()
}

var configSaver = coreConfigSaver{}
