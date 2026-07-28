// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package ssh

import (
	"context"

	"github.com/toeirei/keymaster/connector"
	"github.com/toeirei/keymaster/core/deploy"
	"github.com/toeirei/keymaster/core/security"
	"github.com/toeirei/keymaster/ui/i18n"
)

type deployerClient interface {
	DeployAuthorizedKeys(content string) error
	GetAuthorizedKeys() ([]byte, error)
	Close()
}

var newDeployer = func(host, user string, privateKey security.Secret, passphrase []byte, config *deploy.ConnectionConfig, isBootstrap bool) (deployerClient, error) {
	return deploy.NewDeployerWithConfig(host, user, privateKey, passphrase, config, isBootstrap)
}

// connection is an open session to one target. It carries the host key resolved
// while opening and the authorized_keys fingerprint it last saw, so [Close] can
// hand both back on every path, including one where an operation failed after
// the connection was already up.
//
// A host key newly trusted by a dial that then fails to authenticate is still
// lost: there is no connection to close in that case. Only what happens after a
// successful dial is recoverable this way.
type connection struct {
	connector *Connector
	client    deployerClient
	knownHost string
	hash      string
	closed    bool
}

// *[connection] implements [connector.Connection]
var _ connector.Connection = (*connection)(nil)

func (n *connection) Close() connector.Cache {
	if !n.closed {
		n.closed = true
		n.client.Close()
	}
	return &cache{n.hash, n.knownHost}
}

func (n *connection) Deploy(ctx context.Context, deployment connector.Deployment, progress chan<- connector.Progress) error {
	if err := n.usable(ctx); err != nil {
		return err
	}

	progress <- connector.Progress{0.3, i18n.Text("connector.status.rendering_keys")}
	authorizedKeys, err := n.connector.renderAuthorizedKeys(deployment)
	if err != nil {
		return err
	}

	progress <- connector.Progress{0.7, i18n.Text("connector.status.uploading_keys")}
	if err := n.client.DeployAuthorizedKeys(authorizedKeys); err != nil {
		return i18n.WrapError(err, "errors.connector.deploy_keys")
	}
	n.hash = n.connector.hashAuthorizedKeys(authorizedKeys)

	progress <- connector.Progress{1, i18n.Text("connector.status.done")}
	return nil
}

func (n *connection) Verify(ctx context.Context, deployment connector.Deployment, progress chan<- connector.Progress) (bool, error) {
	if err := n.usable(ctx); err != nil {
		return false, err
	}

	progress <- connector.Progress{0.3, i18n.Text("connector.status.rendering_keys")}
	expected, err := n.connector.renderAuthorizedKeys(deployment)
	if err != nil {
		return false, err
	}

	progress <- connector.Progress{0.7, i18n.Text("connector.status.reading_keys")}
	remoteBytes, err := n.client.GetAuthorizedKeys()
	if err != nil {
		return false, i18n.WrapError(err, "errors.connector.read_keys")
	}

	// What the target actually holds, not what it was meant to hold: a cache that
	// recorded the expectation would hide the drift it exists to surface.
	n.hash = n.connector.hashAuthorizedKeys(string(remoteBytes))

	ok := n.hash == n.connector.hashAuthorizedKeys(expected)
	if ok {
		progress <- connector.Progress{1, i18n.Text("connector.status.verified")}
	} else {
		progress <- connector.Progress{1, i18n.Text("connector.status.drift_detected")}
	}

	return ok, nil
}

func (n *connection) usable(ctx context.Context) error {
	if n.closed {
		return i18n.NewError("errors.connector.connection_closed", "ssh")
	}
	return ctx.Err()
}
