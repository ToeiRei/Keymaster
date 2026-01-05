package tui

import (
	"errors"
	"fmt"
	"time"

	"github.com/toeirei/keymaster/internal/config"
	"github.com/toeirei/keymaster/internal/core"
	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/deploy"
	"github.com/toeirei/keymaster/internal/keys"
	"github.com/toeirei/keymaster/internal/model"
	"github.com/toeirei/keymaster/internal/state"
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

func (coreDeployAdapter) GetRemoteHostKey(hostname string) (string, error) {
	pk, err := deploy.GetRemoteHostKey(hostname)
	if err != nil {
		return "", err
	}
	return string(ssh.MarshalAuthorizedKey(pk)), nil
}

func (coreDeployAdapter) CanonicalizeHostPort(host string) string {
	return deploy.CanonicalizeHostPort(host)
}

func (coreDeployAdapter) ParseHostPort(host string) (string, string, error) {
	return deploy.ParseHostPort(host)
}

// FetchAuthorizedKeys returns the raw authorized_keys content from the remote host.
func (coreDeployAdapter) FetchAuthorizedKeys(account model.Account) ([]byte, error) {
	var privateKey string
	if account.Serial == 0 {
		sk, err := db.GetActiveSystemKey()
		if err != nil {
			return nil, fmt.Errorf("failed to get active system key: %w", err)
		}
		if sk != nil {
			privateKey = sk.PrivateKey
		}
	} else {
		sk, err := db.GetSystemKeyBySerial(account.Serial)
		if err != nil {
			return nil, fmt.Errorf("failed to get system key for serial %d: %w", account.Serial, err)
		}
		if sk == nil {
			return nil, fmt.Errorf("no system key for serial %d", account.Serial)
		}
		privateKey = sk.PrivateKey
	}

	passphrase := state.PasswordCache.Get()
	defer func() {
		for i := range passphrase {
			passphrase[i] = 0
		}
	}()

	deployer, err := deploy.NewDeployerFunc(account.Hostname, account.Username, privateKey, passphrase)
	if err != nil {
		return nil, err
	}
	defer deployer.Close()
	state.PasswordCache.Clear()

	content, err := deployer.GetAuthorizedKeys()
	if err != nil {
		return nil, err
	}
	return content, nil
}

func (coreDeployAdapter) ImportRemoteKeys(account model.Account) ([]model.PublicKey, int, string, error) {
	return deploy.ImportRemoteKeys(account)
}

func (coreDeployAdapter) DecommissionAccount(account model.Account, systemKey string, options interface{}) (core.DecommissionResult, error) {
	var opts deploy.DecommissionOptions
	if o, ok := options.(deploy.DecommissionOptions); ok {
		opts = o
	}
	r := deploy.DecommissionAccount(account, systemKey, opts)
	return core.DecommissionResult{Account: account, Skipped: r.Skipped, DatabaseDeleteError: r.DatabaseDeleteError}, nil
}

func (coreDeployAdapter) DeployForAccount(account model.Account, keepFile bool) error {
	// TUI always runs in interactive/TUI mode
	return deploy.RunDeploymentForAccount(account, true)
}

func (coreDeployAdapter) RunDeploymentForAccount(account model.Account, isTUI bool) error {
	return deploy.RunDeploymentForAccount(account, isTUI)
}

func (coreDeployAdapter) AuditSerial(account model.Account) error {
	return deploy.AuditAccountSerial(account)
}

func (coreDeployAdapter) AuditStrict(account model.Account) error {
	return deploy.AuditAccountStrict(account)
}

func (coreDeployAdapter) BulkDecommissionAccounts(accounts []model.Account, systemKey string, options interface{}) ([]core.DecommissionResult, error) {
	var opts deploy.DecommissionOptions
	if o, ok := options.(deploy.DecommissionOptions); ok {
		opts = o
	}
	res := deploy.BulkDecommissionAccounts(accounts, systemKey, opts)
	out := make([]core.DecommissionResult, 0, len(res))
	for i, r := range res {
		var acc model.Account
		if i < len(accounts) {
			acc = accounts[i]
		}
		out = append(out, core.DecommissionResult{Account: acc, Skipped: r.Skipped, DatabaseDeleteError: r.DatabaseDeleteError})
	}
	return out, nil
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
