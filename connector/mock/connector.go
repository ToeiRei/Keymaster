// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package mock

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/toeirei/keymaster/connector"
	"github.com/toeirei/keymaster/ui/i18n"
)

// register Connector
func init() {
	connector.Register("mock", &Connector{})
}

// Connector is a network-free stand-in for a real connector, intended for
// exercising client-layer functionality without a real remote host. It
// deterministically succeeds or fails based on DeployData.Secret: exactly
// "true" simulates success, any other value simulates a connection failure.
type Connector struct{}

// *[Connector] implements [connector.Connector]
var _ connector.Connector = (*Connector)(nil)

var errSimulatedFailure = errors.New("mock connector: simulated failure (secret != \"true\")")

func (c *Connector) Deploy(ctx context.Context, deployData connector.DeployData, connectionData connector.ConnectionData, userRequester connector.UserRequester, progress chan<- connector.Progress) (string, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	progress <- connector.Progress{Progress: 0.25, Status: i18n.Text("connector.status.connecting")}
	if deployData.Secret != "true" {
		return "", i18n.WrapError(errSimulatedFailure, "errors.connector.connect", connectionData.Username, connectionData.Host)
	}

	progress <- connector.Progress{Progress: 0.6, Status: i18n.Text("connector.status.uploading_keys")}
	newCache := c.hash(deployData)

	progress <- connector.Progress{Progress: 1, Status: i18n.Text("connector.status.done")}
	return newCache, nil
}

func (c *Connector) Verify(ctx context.Context, deployData connector.DeployData, connectionData connector.ConnectionData, userRequester connector.UserRequester, progress chan<- connector.Progress) (bool, string, error) {
	if ctx.Err() != nil {
		return false, "", ctx.Err()
	}

	progress <- connector.Progress{Progress: 0.3, Status: i18n.Text("connector.status.connecting")}
	if deployData.Secret != "true" {
		return false, "", i18n.WrapError(errSimulatedFailure, "errors.connector.connect", connectionData.Username, connectionData.Host)
	}

	progress <- connector.Progress{Progress: 0.6, Status: i18n.Text("connector.status.reading_keys")}
	hash := c.hash(deployData)

	progress <- connector.Progress{Progress: 1, Status: i18n.Text("connector.status.verified")}
	return true, hash, nil
}

func (c *Connector) VerifyOffline(ctx context.Context, deployData connector.DeployData) (bool, error) {
	if deployData.Cache == "" {
		return false, nil
	}
	return c.hash(deployData) == deployData.Cache, nil
}

// hash returns a deterministic SHA256 hex fingerprint of the deploy data, so
// repeated calls with the same records/serial produce the same cache value.
func (c *Connector) hash(deployData connector.DeployData) string {
	var b strings.Builder
	fmt.Fprintf(&b, "serial:%d\n", deployData.SystemKeySerial)
	for _, r := range deployData.Records {
		fmt.Fprintf(&b, "%s %s %s global=%v expires=%s\n", r.Algorithm, r.Data, r.Comment, r.IsGlobal, r.ExpiresAt.UTC())
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}
