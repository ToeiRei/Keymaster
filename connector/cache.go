// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package connector

// Cache holds a connector's record of a target's last known deployed state and
// other potentially relevant data needed by the connector.
type Cache interface {
	// Serialize serializes the cache for persistence, to be parsed back by [Connector.ParseCache].
	Serialize() (string, error)
}
