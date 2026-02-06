// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core

import (
	"fmt"

	"github.com/toeirei/keymaster/internal/core/db"
	"github.com/toeirei/keymaster/internal/core/model"
)

// KeyDeploymentInfo holds a public key and the accounts it's deployed to.
type KeyDeploymentInfo struct {
	Key      model.PublicKey
	Accounts []model.Account
}

// GetKeyDeployments returns all public keys with their assigned accounts.
// For global keys (IsGlobal=true), returns all active accounts since global
// keys are automatically deployed everywhere. For non-global keys, returns
// only accounts with explicit assignments.
// Only includes keys that have at least one account assigned.
func GetKeyDeployments() ([]KeyDeploymentInfo, error) {
	km := db.DefaultKeyManager()
	if km == nil {
		return nil, fmt.Errorf("no key manager available")
	}

	allKeys, err := km.GetAllPublicKeys()
	if err != nil {
		return nil, err
	}

	var deployments []KeyDeploymentInfo
	for _, key := range allKeys {
		var accounts []model.Account

		if key.IsGlobal {
			// Global keys are deployed to all active accounts
			allAccounts, err := db.GetAllAccounts()
			if err != nil {
				return nil, err
			}
			// Filter to only active accounts
			for _, acc := range allAccounts {
				if acc.IsActive {
					accounts = append(accounts, acc)
				}
			}
		} else {
			// Non-global keys: only get explicitly assigned active accounts
			assigned, err := km.GetAccountsForKey(key.ID)
			if err != nil {
				return nil, err
			}
			// Filter to only active accounts (matching global key behavior)
			for _, acc := range assigned {
				if acc.IsActive {
					accounts = append(accounts, acc)
				}
			}
		}

		// Only include keys that are actually deployed somewhere
		if len(accounts) > 0 {
			deployments = append(deployments, KeyDeploymentInfo{
				Key:      key,
				Accounts: accounts,
			})
		}
	}

	return deployments, nil
}

// BuildAccountsByKey groups accounts by their assigned public keys.
// Returns a map of key comment to the list of accounts that have that key.
func BuildAccountsByKey(deployments []KeyDeploymentInfo) map[string][]model.Account {
	m := make(map[string][]model.Account)
	for _, dep := range deployments {
		m[dep.Key.Comment] = dep.Accounts
	}
	return m
}

// GetKeysWithAccounts returns a sorted list of key comments that have accounts assigned.
func GetKeysWithAccounts(deployments []KeyDeploymentInfo) []string {
	keys := make([]string, 0, len(deployments))
	for _, dep := range deployments {
		keys = append(keys, dep.Key.Comment)
	}
	return keys
}

// GetKeyByComment finds a public key in the deployments by its comment.
func GetKeyByComment(deployments []KeyDeploymentInfo, comment string) *model.PublicKey {
	for _, dep := range deployments {
		if dep.Key.Comment == comment {
			return &dep.Key
		}
	}
	return nil
}
