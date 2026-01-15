// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package deploy

import (
	"errors"

	"github.com/toeirei/keymaster/internal/core"
	"github.com/toeirei/keymaster/internal/security"
	"golang.org/x/crypto/ssh"
)

func init() {
	// Wire the core.NewDeployerFactory to produce adapters over *deploy.Deployer.
	core.NewDeployerFactory = func(host, user, privateKey string, passphrase []byte) (core.RemoteDeployer, error) {
		d, err := NewDeployerFunc(host, user, privateKey, passphrase)
		if err != nil {
			return nil, err
		}
		return &deployAdapter{inner: d}, nil
	}

	// Wire bootstrap deployer creation hooks.
	core.NewBootstrapDeployerFunc = func(hostname, username string, privateKey interface{}, expectedHostKey string) (core.BootstrapDeployer, error) {
		// Accept either string (legacy) or security.Secret
		var pkStr string
		switch v := privateKey.(type) {
		case string:
			pkStr = v
		case security.Secret:
			// Copy bytes into a string for the existing deployer API.
			b := v.Bytes()
			pkStr = string(b)
			// zero the temporary copy
			for i := range b {
				b[i] = 0
			}
		default:
			pkStr = ""
		}
		if expectedHostKey != "" {
			return NewBootstrapDeployerWithExpectedKey(hostname, username, pkStr, expectedHostKey)
		}
		return NewBootstrapDeployer(hostname, username, pkStr)
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
