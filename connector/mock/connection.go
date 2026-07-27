// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package mock

import (
	"context"

	"github.com/toeirei/keymaster/connector"
	"github.com/toeirei/keymaster/ui/i18n"
)

// connection is a simulated session. It keeps the hash it last saw so [Close]
// can hand it back the way a real connector hands back what it observed.
type connection struct {
	connector *Connector
	hash      string
	closed    bool
}

// *[connection] implements [connector.Connection]
var _ connector.Connection = (*connection)(nil)

func (n *connection) Close() connector.Cache {
	n.closed = true
	return &cache{n.hash}
}

func (n *connection) Deploy(ctx context.Context, deployment connector.Deployment, progress chan<- connector.Progress) error {
	if err := n.usable(ctx); err != nil {
		return err
	}

	progress <- connector.Progress{Progress: 0.5, Status: i18n.Text("connector.status.uploading_keys")}
	hash, err := n.connector.hash(deployment)
	if err != nil {
		return err
	}
	n.hash = hash

	progress <- connector.Progress{Progress: 1, Status: i18n.Text("connector.status.done")}
	return nil
}

// Verify reports a match whenever the connection is usable: with no remote to
// read there is nothing that could have drifted. A state the mock cannot reach
// is simulated through the secret instead, in [Connector.OpenConnection].
func (n *connection) Verify(ctx context.Context, deployment connector.Deployment, progress chan<- connector.Progress) (bool, error) {
	if err := n.usable(ctx); err != nil {
		return false, err
	}

	progress <- connector.Progress{Progress: 0.5, Status: i18n.Text("connector.status.reading_keys")}
	hash, err := n.connector.hash(deployment)
	if err != nil {
		return false, err
	}
	n.hash = hash

	progress <- connector.Progress{Progress: 1, Status: i18n.Text("connector.status.verified")}
	return true, nil
}

func (n *connection) usable(ctx context.Context) error {
	if n.closed {
		return i18n.NewError("errors.connector.connection_closed", "mock")
	}
	return ctx.Err()
}
