// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core

import "github.com/toeirei/keymaster/internal/model"

// DirtyAccounts returns the subset of accounts whose `IsDirty` flag is true.
// This is a pure helper and performs no side-effects.
func DirtyAccounts(accts []model.Account) []model.Account {
	var out []model.Account
	for _, a := range accts {
		if a.IsDirty {
			out = append(out, a)
		}
	}
	return out
}

// DeployList deploys the provided accounts using the given DeployerManager.
// It returns a slice of `DeployResult` (the core-level type) preserving the
// order of the input accounts. Core intentionally does not clear `IsDirty` or
// update the database; callers are responsible for persisting any desired
// post-deploy side-effects.
func DeployList(dm DeployerManager, accounts []model.Account) []DeployResult {
	results := make([]DeployResult, 0, len(accounts))
	for _, a := range accounts {
		err := dm.DeployForAccount(a, false)
		results = append(results, DeployResult{Account: a, Error: err})
	}
	return results
}

