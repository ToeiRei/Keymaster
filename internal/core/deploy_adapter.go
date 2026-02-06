// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core

import (
	"fmt"

	"github.com/toeirei/keymaster/internal/core/model"
	"github.com/toeirei/keymaster/internal/core/security"
	"github.com/toeirei/keymaster/internal/core/state"
)

type builtinBootstrapDeployer struct{ d BootstrapDeployer }

func (b *builtinBootstrapDeployer) DeployAuthorizedKeys(content string) error {
	return b.d.DeployAuthorizedKeys(content)
}

func (b *builtinBootstrapDeployer) Close() { b.d.Close() }

// NewBootstrapDeployer creates a BootstrapDeployer via the registered hook.
func NewBootstrapDeployer(hostname, username string, privateKey security.Secret, expectedHostKey string) (BootstrapDeployer, error) {
	d, err := NewBootstrapDeployerFunc(hostname, username, privateKey, expectedHostKey)
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
	return RunDeploymentForAccount(account, keepFile)
}

func (builtinDeployerManager) AuditSerial(account model.Account) error {
	return AuditAccountSerial(account)
}
func (builtinDeployerManager) AuditStrict(account model.Account) error {
	return AuditAccountStrict(account)
}

func (builtinDeployerManager) DecommissionAccount(account model.Account, systemPrivateKey security.Secret, options interface{}) (DecommissionResult, error) {
	var opts DecommissionOptions
	if o, ok := options.(DecommissionOptions); ok {
		opts = o
	}
	r := DecommissionAccount(account, systemPrivateKey, opts)
	return r, nil
}

func (builtinDeployerManager) BulkDecommissionAccounts(accounts []model.Account, systemPrivateKey security.Secret, options interface{}) ([]DecommissionResult, error) {
	var opts DecommissionOptions
	if o, ok := options.(DecommissionOptions); ok {
		opts = o
	}
	res := BulkDecommissionAccounts(accounts, systemPrivateKey, opts)
	return res, nil
}

func (builtinDeployerManager) CanonicalizeHostPort(host string) string {
	return CanonicalizeHostPort(host)
}
func (builtinDeployerManager) ParseHostPort(host string) (string, string, error) {
	return ParseHostPort(host)
}
func (builtinDeployerManager) GetRemoteHostKey(host string) (string, error) {
	return GetRemoteHostKey(host)
}

func (builtinDeployerManager) FetchAuthorizedKeys(account model.Account) ([]byte, error) {
	// Use NewDeployerFactory hook which handles agent/passphrase.
	var privateKeySecret security.Secret
	kr := DefaultKeyReader()
	if kr == nil {
		privateKeySecret = nil
	} else {
		if account.Serial == 0 {
			sk, err := kr.GetActiveSystemKey()
			if err != nil {
				return nil, fmt.Errorf("failed to get active system key: %w", err)
			}
			if sk != nil {
				privateKeySecret = SystemKeyToSecret(sk)
			}
		} else {
			sk, err := kr.GetSystemKeyBySerial(account.Serial)
			if err != nil {
				return nil, fmt.Errorf("failed to get system key for serial %d: %w", account.Serial, err)
			}
			if sk == nil {
				return nil, fmt.Errorf("no system key for serial %d", account.Serial)
			}
			privateKeySecret = SystemKeyToSecret(sk)
		}
	}

	passphrase := state.PasswordCache.Get()
	defer func() {
		for i := range passphrase {
			passphrase[i] = 0
		}
	}()

	deployer, err := NewDeployerFactory(account.Hostname, account.Username, privateKeySecret, passphrase)
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
	return ImportRemoteKeys(account)
}

func (builtinDeployerManager) IsPassphraseRequired(err error) bool {
	return IsPassphraseRequired(err)
}

// DefaultDeployerManager is the production implementation used by UIs.
var DefaultDeployerManager DeployerManager = builtinDeployerManager{}
