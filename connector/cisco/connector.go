// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package ssh

import (
	"context"

	"github.com/toeirei/keymaster/connector"
)

// register Connector
func init() {
	// connector.Register("cisco", &Connector{})
}

type Connector struct{}

// *[Connector] implements [connector.Connector]
var _ connector.Connector = (*Connector)(nil)

func (c *Connector) Deploy(ctx context.Context, deployData connector.DeployData, connectionData connector.ConnectionData, userRequester connector.UserRequester) (chan connector.Progress, error) {
	panic("unimplemented")
}

func (c *Connector) Verify(ctx context.Context, deployData connector.DeployData, connectionData connector.ConnectionData, userRequester connector.UserRequester) (chan connector.Progress, error) {
	panic("unimplemented")
}

func (c *Connector) VerifyOffline(ctx context.Context, deployData connector.DeployData) (bool, error) {
	panic("unimplemented")
}
