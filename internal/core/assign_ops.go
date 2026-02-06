// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core

import (
	"fmt"

	"github.com/toeirei/keymaster/internal/core/model"
)

// AssignFunc is a callback that performs the actual persistence of an
// assignment (e.g., calling into a KeyManager). It should return an error
// if the underlying operation fails.
type AssignFunc func(keyID, accountID int) error

// AssignKeyToAccount verifies the key exists in the provided keys slice,
// invokes assignFunc to persist the assignment, and returns an updated
// assigned map. It does not perform localization of messages.
func AssignKeyToAccount(keys []model.PublicKey, assigned map[int]struct{}, keyID, accountID int, assignFunc AssignFunc) (map[int]struct{}, error) {
	// Verify existence in-memory
	exists := false
	for _, k := range keys {
		if k.ID == keyID {
			exists = true
			break
		}
	}
	if !exists {
		return assigned, fmt.Errorf("key ID %d not found", keyID)
	}
	if assignFunc == nil {
		return assigned, fmt.Errorf("no assign function provided")
	}
	if err := assignFunc(keyID, accountID); err != nil {
		return assigned, err
	}
	if assigned == nil {
		assigned = make(map[int]struct{})
	}
	assigned[keyID] = struct{}{}
	return assigned, nil
}

// UnassignKeyFromAccount invokes unassignFunc to persist the unassignment
// and removes the key from the assigned map if present.
func UnassignKeyFromAccount(assigned map[int]struct{}, keyID, accountID int, unassignFunc AssignFunc) (map[int]struct{}, error) {
	if unassignFunc == nil {
		return assigned, fmt.Errorf("no unassign function provided")
	}
	if err := unassignFunc(keyID, accountID); err != nil {
		return assigned, err
	}
	if assigned != nil {
		delete(assigned, keyID)
	}
	return assigned, nil
}

// AssignKeys iterates over the provided key IDs and invokes assignFunc for each.
// It returns the first non-nil error encountered.
func AssignKeys(keyIDs []int, accountID int, assignFunc AssignFunc) error {
	if assignFunc == nil {
		return fmt.Errorf("no assign function provided")
	}
	for _, kid := range keyIDs {
		if err := assignFunc(kid, accountID); err != nil {
			return err
		}
	}
	return nil
}
