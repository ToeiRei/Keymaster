package main

import (
	"fmt"

	"github.com/toeirei/keymaster/internal/core"
	crypto_ssh "github.com/toeirei/keymaster/internal/crypto/ssh"
	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/deploy"
	"github.com/toeirei/keymaster/internal/model"
	"github.com/toeirei/keymaster/internal/state"
)

// cliStoreAdapter adapts package-level db helpers to core.Store.
type cliStoreAdapter struct{}

func (c *cliStoreAdapter) GetAccounts() ([]model.Account, error) {
	return db.GetAllAccounts()
}
func (c *cliStoreAdapter) GetAllActiveAccounts() ([]model.Account, error) {
	return db.GetAllActiveAccounts()
}
func (c *cliStoreAdapter) GetAllAccounts() ([]model.Account, error) {
	return db.GetAllAccounts()
}
func (c *cliStoreAdapter) GetAccount(id int) (*model.Account, error) {
	accts, err := db.GetAllAccounts()
	if err != nil {
		return nil, err
	}
	for _, a := range accts {
		if a.ID == id {
			return &a, nil
		}
	}
	return nil, fmt.Errorf("account not found: %d", id)
}
func (c *cliStoreAdapter) AddAccount(username, hostname, label, tags string) (int, error) {
	mgr := db.DefaultAccountManager()
	if mgr == nil {
		return 0, fmt.Errorf("no account manager available")
	}
	return mgr.AddAccount(username, hostname, label, tags)
}
func (c *cliStoreAdapter) DeleteAccount(accountID int) error {
	mgr := db.DefaultAccountManager()
	if mgr == nil {
		return fmt.Errorf("no account manager available")
	}
	return mgr.DeleteAccount(accountID)
}
func (c *cliStoreAdapter) AssignKeyToAccount(keyID, accountID int) error {
	km := db.DefaultKeyManager()
	if km == nil {
		return fmt.Errorf("no key manager available")
	}
	return km.AssignKeyToAccount(keyID, accountID)
}
func (c *cliStoreAdapter) CreateSystemKey(publicKey, privateKey string) (int, error) {
	return db.CreateSystemKey(publicKey, privateKey)
}
func (c *cliStoreAdapter) RotateSystemKey(publicKey, privateKey string) (int, error) {
	return db.RotateSystemKey(publicKey, privateKey)
}
func (c *cliStoreAdapter) GetActiveSystemKey() (*model.SystemKey, error) {
	return db.GetActiveSystemKey()
}
func (c *cliStoreAdapter) AddKnownHostKey(hostname, key string) error {
	return db.AddKnownHostKey(hostname, key)
}
func (c *cliStoreAdapter) ExportDataForBackup() (*model.BackupData, error) {
	return db.ExportDataForBackup()
}
func (c *cliStoreAdapter) ImportDataFromBackup(d *model.BackupData) error {
	return db.ImportDataFromBackup(d)
}
func (c *cliStoreAdapter) IntegrateDataFromBackup(d *model.BackupData) error {
	return db.IntegrateDataFromBackup(d)
}

// cliDeployerManager adapts deploy package helpers to core.DeployerManager.
type cliDeployerManager struct{}

func (c *cliDeployerManager) DeployForAccount(account model.Account, keepFile bool) error {
	// deploy.RunDeploymentForAccount expects (account, isTUI)
	return deploy.RunDeploymentForAccount(account, false)
}
func (c *cliDeployerManager) AuditSerial(account model.Account) error {
	return deploy.AuditAccountSerial(account)
}
func (c *cliDeployerManager) AuditStrict(account model.Account) error {
	return deploy.AuditAccountStrict(account)
}
func (c *cliDeployerManager) DecommissionAccount(account model.Account, systemPrivateKey string, options interface{}) (core.DecommissionResult, error) {
	// options come from deploy.DecommissionOptions in CLI path; try to assert
	var opts deploy.DecommissionOptions
	if o, ok := options.(deploy.DecommissionOptions); ok {
		opts = o
	}
	res := deploy.DecommissionAccount(account, systemPrivateKey, opts)
	// map deploy.DecommissionResult -> core.DecommissionResult
	cr := core.DecommissionResult{
		Account:             account,
		AccountID:           res.AccountID,
		AccountString:       res.AccountString,
		RemoteCleanupDone:   res.RemoteCleanupDone,
		RemoteCleanupError:  res.RemoteCleanupError,
		DatabaseDeleteDone:  res.DatabaseDeleteDone,
		DatabaseDeleteError: res.DatabaseDeleteError,
		BackupPath:          res.BackupPath,
		Skipped:             res.Skipped,
		SkipReason:          res.SkipReason,
	}
	return cr, nil
}
func (c *cliDeployerManager) BulkDecommissionAccounts(accounts []model.Account, systemPrivateKey string, options interface{}) ([]core.DecommissionResult, error) {
	var opts deploy.DecommissionOptions
	if o, ok := options.(deploy.DecommissionOptions); ok {
		opts = o
	}
	res := deploy.BulkDecommissionAccounts(accounts, systemPrivateKey, opts)
	out := make([]core.DecommissionResult, 0, len(res))
	for i, r := range res {
		var acc model.Account
		if i < len(accounts) {
			acc = accounts[i]
		}
		cr := core.DecommissionResult{
			Account:             acc,
			AccountID:           r.AccountID,
			AccountString:       r.AccountString,
			RemoteCleanupDone:   r.RemoteCleanupDone,
			RemoteCleanupError:  r.RemoteCleanupError,
			DatabaseDeleteDone:  r.DatabaseDeleteDone,
			DatabaseDeleteError: r.DatabaseDeleteError,
			BackupPath:          r.BackupPath,
			Skipped:             r.Skipped,
			SkipReason:          r.SkipReason,
		}
		out = append(out, cr)
	}
	return out, nil
}
func (c *cliDeployerManager) CanonicalizeHostPort(host string) string {
	return deploy.CanonicalizeHostPort(host)
}
func (c *cliDeployerManager) ParseHostPort(host string) (string, string, error) {
	return deploy.ParseHostPort(host)
}
func (c *cliDeployerManager) GetRemoteHostKey(host string) (string, error) {
	pk, err := deploy.GetRemoteHostKey(host)
	if err != nil {
		return "", err
	}
	return string(crypto_ssh.MarshalAuthorizedKey(pk)), nil
}

// FetchAuthorizedKeys fetches the raw authorized_keys content bytes for the account.
func (c *cliDeployerManager) FetchAuthorizedKeys(account model.Account) ([]byte, error) {
	var privateKey string
	if account.Serial == 0 {
		sk, err := db.GetActiveSystemKey()
		if err != nil {
			return nil, fmt.Errorf("failed to get active system key: %w", err)
		}
		if sk != nil {
			privateKey = sk.PrivateKey
		} else {
			privateKey = ""
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
	return w.inner.GetAllActiveAccounts()
}
func (w *dbStoreWrapper) GetAllAccounts() ([]model.Account, error) { return w.inner.GetAllAccounts() }
func (w *dbStoreWrapper) GetAccount(id int) (*model.Account, error) {
	ac, _ := w.inner.GetAllAccounts()
	for _, a := range ac {
		if a.ID == id {
			return &a, nil
		}
	}
	return nil, fmt.Errorf("not found")
}
func (w *dbStoreWrapper) AddAccount(username, hostname, label, tags string) (int, error) {
	return w.inner.AddAccount(username, hostname, label, tags)
}
func (w *dbStoreWrapper) DeleteAccount(accountID int) error             { return w.inner.DeleteAccount(accountID) }
func (w *dbStoreWrapper) AssignKeyToAccount(keyID, accountID int) error { return nil }
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

// cliKeyGenerator delegates to the package-level generator function.
type cliKeyGenerator struct{}

func (c *cliKeyGenerator) GenerateAndMarshalEd25519Key(comment, passphrase string) (string, string, error) {
	return crypto_ssh.GenerateAndMarshalEd25519Key(comment, passphrase)
}

// ensure cliStoreAdapter satisfies core.Store at compile time
var _ core.Store = (*cliStoreAdapter)(nil)
var _ core.DeployerManager = (*cliDeployerManager)(nil)
var _ core.DBMaintainer = (*cliDBMaintainer)(nil)
var _ core.StoreFactory = (*cliStoreFactory)(nil)
var _ core.KeyGenerator = (*cliKeyGenerator)(nil)

// cliReporter implements core.Reporter by printing to stdout.
type cliReporter struct{}

func (r *cliReporter) Reportf(format string, args ...any) {
	fmt.Printf(format, args...)
}

var _ core.Reporter = (*cliReporter)(nil)

// cliAuditWriter adapts the package-level DB AuditWriter to core.AuditWriter.
type cliAuditWriter struct{}

func (a *cliAuditWriter) LogAction(action, details string) error {
	if w := db.DefaultAuditWriter(); w != nil {
		return w.LogAction(action, details)
	}
	return nil
}

var _ core.AuditWriter = (*cliAuditWriter)(nil)
