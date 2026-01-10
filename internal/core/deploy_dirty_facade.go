// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core

import (
	"context"
	"fmt"
)

// DeployDirtyAccounts fetches all active accounts from the store, selects
// accounts marked `IsDirty`, deploys to each using the provided DeployerManager,
// and clears the `is_dirty` flag for accounts that deployed successfully.
// It returns the per-account DeployResult slice and an error if fetching
// accounts failed.
func DeployDirtyAccounts(ctx context.Context, st Store, dm DeployerManager, rep Reporter) ([]DeployResult, error) {
	accounts, err := st.GetAllActiveAccounts()
	if err != nil {
		return nil, fmt.Errorf("get accounts: %w", err)
	}

	dirty := DirtyAccounts(accounts)
	results := make([]DeployResult, 0, len(dirty))
	for _, acc := range dirty {
		err := dm.DeployForAccount(acc, false)
		results = append(results, DeployResult{Account: acc, Error: err})
		if err == nil {
			// Best-effort: clear is_dirty; log/store error ignored for now
			_ = st.UpdateAccountIsDirty(acc.ID, false)
		}
	}
	return results, nil
}
