package deploy

import (
	"errors"

	"github.com/toeirei/keymaster/internal/core"
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
	core.NewBootstrapDeployerFunc = func(hostname, username, privateKey, expectedHostKey string) (core.BootstrapDeployer, error) {
		if expectedHostKey != "" {
			return NewBootstrapDeployerWithExpectedKey(hostname, username, privateKey, expectedHostKey)
		}
		return NewBootstrapDeployer(hostname, username, privateKey)
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
