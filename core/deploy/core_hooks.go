// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package deploy

import (
	"errors"

	"github.com/toeirei/keymaster/core"
	"github.com/toeirei/keymaster/core/security"
	"golang.org/x/crypto/ssh"
)

func init() {
	// Wire the core.NewDeployerFactory to produce adapters over *deploy.Deployer.
	core.NewDeployerFactory = func(host, user string, privateKey security.Secret, passphrase []byte) (core.RemoteDeployer, error) {
		d, err := NewDeployerFunc(host, user, privateKey, passphrase)
		if err != nil {
			return nil, err
		}
		return &deployAdapter{inner: d}, nil
	}

	// Wire bootstrap deployer creation hooks.
	core.NewBootstrapDeployerFunc = func(hostname, username string, privateKey interface{}, expectedHostKey string) (core.BootstrapDeployer, error) {
		// Normalize to security.Secret when possible.
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
		if expectedHostKey != "" {
			return NewBootstrapDeployerWithExpectedKey(hostname, username, sk, expectedHostKey)
		}
		return NewBootstrapDeployer(hostname, username, sk)
	}

	// Network helper passthroughs.
	core.CanonicalizeHostPort = CanonicalizeHostPort
	core.ParseHostPort = ParseHostPort
	core.GetRemoteHostKey = func(host string) (string, error) {
		pk, err := GetRemoteHostKey(host)
		if err != nil {
			return "", err
		}
		return string(ssh.MarshalAuthorizedKey(pk)), nil
	}

	core.IsPassphraseRequired = func(err error) bool {
		return errors.Is(err, ErrPassphraseRequired)
	}
}

type deployAdapter struct{ inner *Deployer }

func (a *deployAdapter) DeployAuthorizedKeys(content string) error {
	return a.inner.DeployAuthorizedKeys(content)
}
func (a *deployAdapter) GetAuthorizedKeys() ([]byte, error) { return a.inner.GetAuthorizedKeys() }
func (a *deployAdapter) Close()                             { a.inner.Close() }
