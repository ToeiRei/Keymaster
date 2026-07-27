// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package ssh2

import (
	"context"
	"net"
	"strconv"
	"strings"

	"github.com/toeirei/keymaster/connector"
	"github.com/toeirei/keymaster/core/deploy"
	"github.com/toeirei/keymaster/core/security"
	"github.com/toeirei/keymaster/ui/i18n"
	"golang.org/x/crypto/ssh"
)

// register Connector
func init() {
	connector.Register("ssh", &Connector{})
}

type Connector struct{}

// *[Connector] implements [connector.Connector]
var _ connector.Connector = (*Connector)(nil)

func (c *Connector) OpenConnection(ctx context.Context, secret connector.Secret, cache connector.Cache, user string, host string, port int, userRequester connector.UserRequester) (connector.Connection, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	connectorSecret, err := narrowSecret(secret)
	if err != nil {
		return nil, err
	}
	connectorCache, err := narrowCache(cache)
	if err != nil {
		return nil, err
	}

	config := deploy.DefaultConnectionConfig()
	addr := canonicalSSHAddress(host, port)

	hostKey, knownHost, err := resolveKnownHost(addr, connectorCache.KnownHost, userRequester, config.ConnectionTimeout)
	if err != nil {
		return nil, err
	}
	config.HostKeyCallback = ssh.FixedHostKey(hostKey)

	client, err := newDeployer(addr, user, security.FromString(connectorSecret.PrivateKey), connectorSecret.passphraseBytes(), config, false)
	if err != nil {
		return nil, i18n.WrapError(err, "errors.connector.connect", user, addr)
	}

	return &connection{c, client, knownHost, connectorCache.AuthorizedKeysHash, false}, nil
}

func (c *Connector) VerifyOffline(ctx context.Context, cache connector.Cache, deployment connector.Deployment) (bool, error) {
	connectorCache, err := narrowCache(cache)
	if err != nil {
		return false, err
	}
	if connectorCache.AuthorizedKeysHash == "" {
		return false, nil
	}

	authorizedKeys, err := c.renderAuthorizedKeys(deployment)
	if err != nil {
		return false, err
	}

	return c.hashAuthorizedKeys(authorizedKeys) == connectorCache.AuthorizedKeysHash, nil
}

func canonicalSSHAddress(host string, port int) string {
	if port <= 0 {
		port = 22
	}
	return net.JoinHostPort(strings.TrimSpace(host), strconv.Itoa(port))
}
