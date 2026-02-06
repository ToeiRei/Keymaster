// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core

import (
	"fmt"

	"github.com/toeirei/keymaster/core/security"
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
var NewDeployerFactory = func(host, user string, privateKey security.Secret, passphrase []byte) (RemoteDeployer, error) {
	return nil, fmt.Errorf("no deployer factory configured")
}

// NewBootstrapDeployerFunc is a hook that production code may set to create
// bootstrap deployers without core importing the deploy package.
var NewBootstrapDeployerFunc = func(hostname, username string, privateKey interface{}, expectedHostKey string) (BootstrapDeployer, error) {
	// Default is conservative: accept interface{} to avoid early breakage; callers
	// should provide a `security.Secret` in normal usage.
	return nil, fmt.Errorf("no bootstrap deployer configured")
}

// Network helpers / feature hooks that may be implemented by the lower-level
// deploy package. These are set by `internal/deploy` during program init so
// core does not import the deploy package directly.

// CanonicalizeHostPort normalizes a host string into a canonical host[:port]
// representation used by deployers.
var CanonicalizeHostPort = func(host string) string { return host }

// ParseHostPort extracts host and port from a canonical host[:port] string.
var ParseHostPort = func(host string) (string, string, error) { return host, "", nil }

// GetRemoteHostKey fetches the remote host key for a given host (used for trust-on-first-use).
var GetRemoteHostKey = func(host string) (string, error) { return "", fmt.Errorf("host key fetcher not configured") }

// IsPassphraseRequired examines an error returned while accessing a key and
// returns true when the error indicates that a passphrase is required.
var IsPassphraseRequired = func(err error) bool { return false }
