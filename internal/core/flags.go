// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core

import "github.com/toeirei/keymaster/internal/model"

// GetAccountIsDirty returns whether the given account is marked dirty.
func GetAccountIsDirty(a model.Account) bool {
	return a.IsDirty
}

// WithAccountIsDirty returns a copy of the account with IsDirty set to the
// provided value. This is a pure helper useful for core-level operations
// that should not mutate caller state directly.
func WithAccountIsDirty(a model.Account, dirty bool) model.Account {
	a.IsDirty = dirty
	return a
}

// SetAccountsIsDirtyByID returns a new slice of accounts where accounts with
// IDs in the `ids` map have their IsDirty flag set to `dirty`.
// The input slice is not mutated.
func SetAccountsIsDirtyByID(accounts []model.Account, ids map[int]struct{}, dirty bool) []model.Account {
	out := make([]model.Account, 0, len(accounts))
	for _, a := range accounts {
		if _, ok := ids[a.ID]; ok {
			a.IsDirty = dirty
		}
		out = append(out, a)
	}
	return out
}

