package core

import (
	"github.com/toeirei/keymaster/internal/deploy"
)

// deployerIface is a minimal interface used by core to interact with remote
// deployers. Tests can provide fakes that implement this interface by
// overriding newDeployerFactory.
// Deployer is an exported minimal interface used by core to interact with
// remote deployers. Tests and external packages can implement this for
// injection.
type RemoteDeployer interface {
	DeployAuthorizedKeys(content string) error
	GetAuthorizedKeys() ([]byte, error)
	Close()
}

// NewDeployerFactory creates a Deployer for a host/user/privateKey and
// passphrase. By default it delegates to deploy.NewDeployerFunc and returns
// an adapter wrapping *deploy.Deployer. Tests may set this variable to inject
// fakes.
var NewDeployerFactory = func(host, user, privateKey string, passphrase []byte) (RemoteDeployer, error) {
	d, err := deploy.NewDeployerFunc(host, user, privateKey, passphrase)
	if err != nil {
		return nil, err
	}
	return &deployAdapter{inner: d}, nil
}

type deployAdapter struct{ inner *deploy.Deployer }

func (a *deployAdapter) DeployAuthorizedKeys(content string) error {
	return a.inner.DeployAuthorizedKeys(content)
}
func (a *deployAdapter) GetAuthorizedKeys() ([]byte, error) { return a.inner.GetAuthorizedKeys() }
func (a *deployAdapter) Close()                             { a.inner.Close() }
