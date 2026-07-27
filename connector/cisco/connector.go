// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package cisco

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

func (c *Connector) OpenConnection(ctx context.Context, secret connector.Secret, cache connector.Cache, user string, host string, port int, userRequester connector.UserRequester) (connector.Connection, error) {
	panic("unimplemented")
}

func (c *Connector) SecretFields() []connector.SecretField {
	panic("unimplemented")
}

func (c *Connector) ParseSecretFromFields(fieldValues map[string]string) (connector.Secret, error) {
	panic("unimplemented")
}

func (c *Connector) ParseSecret(raw string) (connector.Secret, error) {
	panic("unimplemented")
}

func (c *Connector) ParseCache(raw string) (connector.Cache, error) {
	panic("unimplemented")
}

func (c *Connector) VerifyOffline(ctx context.Context, cache connector.Cache, deployment connector.Deployment) (bool, error) {
	panic("unimplemented")
}
