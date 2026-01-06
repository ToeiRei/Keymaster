package core

import (
	"errors"
	"fmt"

	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/deploy"
	"github.com/toeirei/keymaster/internal/model"
	"github.com/toeirei/keymaster/internal/state"
	"golang.org/x/crypto/ssh"
)

type builtinBootstrapDeployer struct{ d *deploy.Deployer }

func (b *builtinBootstrapDeployer) DeployAuthorizedKeys(content string) error {
	return b.d.DeployAuthorizedKeys(content)
}

func (b *builtinBootstrapDeployer) Close() { b.d.Close() }

// NewBootstrapDeployer creates a BootstrapDeployer using the deploy package.
func NewBootstrapDeployer(hostname, username, privateKey, expectedHostKey string) (BootstrapDeployer, error) {
	var d *deploy.Deployer
	var err error
	if expectedHostKey != "" {
		d, err = deploy.NewBootstrapDeployerWithExpectedKey(hostname, username, privateKey, expectedHostKey)
	} else {
		d, err = deploy.NewBootstrapDeployer(hostname, username, privateKey)
	}
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, nil
	}
	return &builtinBootstrapDeployer{d: d}, nil
}

type builtinDeployerManager struct{}

func (builtinDeployerManager) DeployForAccount(account model.Account, keepFile bool) error {
	return deploy.RunDeploymentForAccount(account, keepFile)
}

func (builtinDeployerManager) AuditSerial(account model.Account) error {
	return deploy.AuditAccountSerial(account)
}
func (builtinDeployerManager) AuditStrict(account model.Account) error {
	return deploy.AuditAccountStrict(account)
}

func (builtinDeployerManager) DecommissionAccount(account model.Account, systemPrivateKey string, options interface{}) (DecommissionResult, error) {
	var opts deploy.DecommissionOptions
	if o, ok := options.(deploy.DecommissionOptions); ok {
		opts = o
	} else if o2, ok := options.(DecommissionOptions); ok {
		opts = deploy.DecommissionOptions{
			SkipRemoteCleanup: o2.SkipRemoteCleanup,
			KeepFile:          o2.KeepFile,
			Force:             o2.Force,
			DryRun:            o2.DryRun,
			SelectiveKeys:     o2.SelectiveKeys,
		}
	}
	r := deploy.DecommissionAccount(account, systemPrivateKey, opts)
	return DecommissionResult{
		Account:             account,
		AccountID:           r.AccountID,
		AccountString:       r.AccountString,
		RemoteCleanupDone:   r.RemoteCleanupDone,
		RemoteCleanupError:  r.RemoteCleanupError,
		DatabaseDeleteDone:  r.DatabaseDeleteDone,
		DatabaseDeleteError: r.DatabaseDeleteError,
		BackupPath:          r.BackupPath,
		Skipped:             r.Skipped,
		SkipReason:          r.SkipReason,
	}, nil
}

func (builtinDeployerManager) BulkDecommissionAccounts(accounts []model.Account, systemPrivateKey string, options interface{}) ([]DecommissionResult, error) {
	var opts deploy.DecommissionOptions
	if o, ok := options.(deploy.DecommissionOptions); ok {
		opts = o
	} else if o2, ok := options.(DecommissionOptions); ok {
		opts = deploy.DecommissionOptions{
			SkipRemoteCleanup: o2.SkipRemoteCleanup,
			KeepFile:          o2.KeepFile,
			Force:             o2.Force,
			DryRun:            o2.DryRun,
			SelectiveKeys:     o2.SelectiveKeys,
		}
	}
	res := deploy.BulkDecommissionAccounts(accounts, systemPrivateKey, opts)
	out := make([]DecommissionResult, 0, len(res))
	for i, r := range res {
		var acc model.Account
		if i < len(accounts) {
			acc = accounts[i]
		}
		out = append(out, DecommissionResult{
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
		})
	}
	return out, nil
}

func (builtinDeployerManager) CanonicalizeHostPort(host string) string {
	return deploy.CanonicalizeHostPort(host)
}
func (builtinDeployerManager) ParseHostPort(host string) (string, string, error) {
	return deploy.ParseHostPort(host)
}
func (builtinDeployerManager) GetRemoteHostKey(host string) (string, error) {
	pk, err := deploy.GetRemoteHostKey(host)
	if err != nil {
		return "", err
	}
	return string(ssh.MarshalAuthorizedKey(pk)), nil
}

func (builtinDeployerManager) FetchAuthorizedKeys(account model.Account) ([]byte, error) {
	// Use deploy.NewDeployerFunc which handles agent/passphrase.
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

func (builtinDeployerManager) ImportRemoteKeys(account model.Account) ([]model.PublicKey, int, string, error) {
	return deploy.ImportRemoteKeys(account)
}

func (builtinDeployerManager) IsPassphraseRequired(err error) bool {
	return errors.Is(err, deploy.ErrPassphraseRequired)
}

// DefaultDeployerManager is the production implementation used by UIs.
var DefaultDeployerManager DeployerManager = builtinDeployerManager{}
