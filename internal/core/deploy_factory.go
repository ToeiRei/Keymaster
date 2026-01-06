package core

import (
	"fmt"
)

// RemoteDeployer is a minimal interface used by core to interact with remote
// deployers. Tests can provide fakes by overriding `NewDeployerFactory`.
type RemoteDeployer interface {
	DeployAuthorizedKeys(content string) error
	GetAuthorizedKeys() ([]byte, error)
	Close()
}

// NewDeployerFactory creates a RemoteDeployer for a host/user/privateKey and
// passphrase. Production code in `internal/deploy` will register a working
// factory at init time. Tests may override this variable to inject fakes.
var NewDeployerFactory = func(host, user, privateKey string, passphrase []byte) (RemoteDeployer, error) {
	return nil, fmt.Errorf("no deployer factory configured")
}

// NewBootstrapDeployerFunc is a hook that production code may set to create
// bootstrap deployers without core importing the deploy package.
var NewBootstrapDeployerFunc = func(hostname, username, privateKey, expectedHostKey string) (BootstrapDeployer, error) {
	return nil, fmt.Errorf("no bootstrap deployer configured")
}

// Network helpers / feature hooks that may be implemented by the lower-level
// deploy package. These are set by `internal/deploy` during program init so
// core does not import the deploy package directly.
var CanonicalizeHostPort = func(host string) string { return host }
var ParseHostPort = func(host string) (string, string, error) { return host, "", nil }
var GetRemoteHostKey = func(host string) (string, error) { return "", fmt.Errorf("host key fetcher not configured") }
var IsPassphraseRequired = func(err error) bool { return false }
