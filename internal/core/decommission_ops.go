// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core

import (
	"github.com/toeirei/keymaster/internal/model"
)

// PerformDecommissionWithKeys performs a selective decommission for the
// provided account using the selectedKeysToKeep map to determine which
// Key IDs should be preserved. The actual deployment/decommission work is
// delegated to the provided decommander function so that core remains
// side-effect free and callers can inject the environment-specific logic.
//
// The function works with core.DecommissionResult to avoid leaking the
// lower-level deploy package into UI code.
func PerformDecommissionWithKeys(account model.Account, selectedKeysToKeep map[int]bool, decommander func(model.Account, map[int]bool) (DecommissionResult, error)) (DecommissionResult, error) {
	if decommander == nil {
		return DecommissionResult{}, nil
	}
	return decommander(account, selectedKeysToKeep)
}
