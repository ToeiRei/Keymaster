// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core

import (
	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/deploy"
	"github.com/toeirei/keymaster/internal/model"
)

// PerformDecommissionWithKeys performs a selective decommission for the
// provided account using the selectedKeysToKeep map to determine which
// Key IDs should be preserved. It returns the DecommissionResult or an
// error if a precondition (like an active system key) is missing.
func PerformDecommissionWithKeys(account model.Account, selectedKeysToKeep map[int]bool) (deploy.DecommissionResult, error) {
	// Get active system key
	systemKey, err := db.GetActiveSystemKey()
	if err != nil || systemKey == nil {
		return deploy.DecommissionResult{}, err
	}

	// Build list of key IDs to remove (inverse of keys to keep)
	var keysToRemove []int
	for keyID, shouldKeep := range selectedKeysToKeep {
		if !shouldKeep {
			keysToRemove = append(keysToRemove, keyID)
		}
	}

	options := deploy.DecommissionOptions{
		SkipRemoteCleanup: false,
		KeepFile:          true,
		Force:             false,
		DryRun:            false,
		SelectiveKeys:     keysToRemove,
	}

	result := deploy.DecommissionAccount(account, systemKey.PrivateKey, options)
	return result, nil
}

