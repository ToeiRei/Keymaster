// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package connector

import (
	"context"
	"fmt"
)

type Connection interface {
	// Close closes the active connection which renders it unusable, and returns
	// the connector's record of the target: seeded from the cache passed to
	// [Connector.OpenConnection] and advanced by every operation that reached the
	// remote. Never nil, and safe to call more than once.
	Close() Cache

	// Deploy deploys the provided deployment to the connected remote.
	Deploy(ctx context.Context, deployment Deployment, progress chan<- Progress) error

	// Verify verifies the provided deployment matches the remote's state.
	Verify(ctx context.Context, deployment Deployment, progress chan<- Progress) (ok bool, err error)
}

type Progress struct {
	Progress float64
	Status   fmt.Stringer
}
