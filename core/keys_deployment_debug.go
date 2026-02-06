// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package core

import (
	"fmt"

	"github.com/toeirei/keymaster/core/db"
	"github.com/toeirei/keymaster/core/model"
)

// DebugKeyAssignments returns the raw account IDs assigned to a key from the database.
// This is a diagnostic function to verify what's actually in account_keys table.
func DebugKeyAssignments(keyID int) ([]int, error) {
	km := db.DefaultKeyManager()
	if km == nil {
		return nil, fmt.Errorf("no key manager available")
	}

	accounts, err := km.GetAccountsForKey(keyID)
	if err != nil {
		return nil, err
	}

	accountIDs := make([]int, len(accounts))
	for i, acc := range accounts {
		accountIDs[i] = acc.ID
	}
	return accountIDs, nil
}

// DebugAllKeyStats returns stats about all keys and their assignments.
// This shows which keys are actually assigned to how many accounts.
func DebugAllKeyStats() (map[string]interface{}, error) {
	km := db.DefaultKeyManager()
	if km == nil {
		return nil, fmt.Errorf("no key manager available")
	}

	allKeys, err := km.GetAllPublicKeys()
	if err != nil {
		return nil, err
	}

	result := make(map[string]interface{})

	for _, key := range allKeys {
		accounts, err := km.GetAccountsForKey(key.ID)
		if err != nil {
			result[fmt.Sprintf("key_%d_error", key.ID)] = err.Error()
			continue
		}

		result[key.Comment] = map[string]interface{}{
			"key_id":        key.ID,
			"is_global":     key.IsGlobal,
			"algorithm":     key.Algorithm,
			"account_count": len(accounts),
			"account_ids":   getAccountIDs(accounts),
		}
	}

	return result, nil
}

func getAccountIDs(accounts []model.Account) []int {
	ids := make([]int, len(accounts))
	for i, acc := range accounts {
		ids[i] = acc.ID
	}
	return ids
}
