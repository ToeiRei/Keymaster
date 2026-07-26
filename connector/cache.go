// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package connector

// Cache holds a connector's record of a target's last known deployed state.
type Cache interface {
	Serialize() (string, error)
}
