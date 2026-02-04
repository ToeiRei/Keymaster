// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core

import (
	"testing"

	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/model"
)

func TestGetKeyDeployments_GlobalKeys(t *testing.T) {
	// Setup: inject a fake key manager
	originalKM := db.DefaultKeyManager()
	defer func() {
		if originalKM != nil {
			db.SetDefaultKeyManager(originalKM)
		} else {
			db.ClearDefaultKeyManager()
		}
	}()

	fakeKM := &fakeKeyManagerForDeployment{
		allKeys: []model.PublicKey{
			{ID: 1, Comment: "global-key", Algorithm: "ssh-ed25519", IsGlobal: true},
			{ID: 2, Comment: "user-key", Algorithm: "ssh-rsa", IsGlobal: false},
		},
		accountsByKey: map[int][]model.Account{
			// Global key has no explicit assignments in account_keys table
			1: {},
			// Non-global key has explicit assignments (including one inactive)
			2: {
				{ID: 10, Username: "user1", Hostname: "host1", IsActive: true},
				{ID: 12, Username: "user3", Hostname: "host3", IsActive: false}, // inactive
			},
		},
	}
	db.SetDefaultKeyManager(fakeKM)

	// Setup: inject fake store for GetAllAccounts
	originalStore := db.GetStore()
	defer func() {
		if originalStore != nil {
			db.SetStore(originalStore)
		}
	}()

	fakeStore := &fakeStoreForDeployment{
		allAccounts: []model.Account{
			{ID: 10, Username: "user1", Hostname: "host1", IsActive: true},
			{ID: 11, Username: "user2", Hostname: "host2", IsActive: true},
			{ID: 12, Username: "user3", Hostname: "host3", IsActive: false}, // inactive
		},
	}
	db.SetStore(fakeStore)

	// Execute
	deployments, err := GetKeyDeployments()
	if err != nil {
		t.Fatalf("GetKeyDeployments failed: %v", err)
	}

	// Verify: should have 2 deployments (global key + user key)
	if len(deployments) != 2 {
		t.Fatalf("expected 2 deployments, got %d", len(deployments))
	}

	// Verify global key shows all active accounts
	var globalDep *KeyDeploymentInfo
	var userDep *KeyDeploymentInfo
	for i := range deployments {
		if deployments[i].Key.Comment == "global-key" {
			globalDep = &deployments[i]
		} else if deployments[i].Key.Comment == "user-key" {
			userDep = &deployments[i]
		}
	}

	if globalDep == nil {
		t.Fatal("global-key deployment not found")
	}
	if userDep == nil {
		t.Fatal("user-key deployment not found")
	}

	// Global key should have 2 active accounts (not the inactive one)
	if len(globalDep.Accounts) != 2 {
		t.Errorf("global key: expected 2 active accounts, got %d", len(globalDep.Accounts))
	}
	for _, acc := range globalDep.Accounts {
		if !acc.IsActive {
			t.Errorf("global key should only include active accounts, found inactive: %v", acc)
		}
	}

	// Non-global key should have 1 explicitly assigned account (filtering out the inactive one)
	if len(userDep.Accounts) != 1 {
		t.Errorf("user key: expected 1 active account (inactive filtered), got %d", len(userDep.Accounts))
	}
	if len(userDep.Accounts) > 0 {
		if userDep.Accounts[0].Username != "user1" {
			t.Errorf("user key: expected user1, got %s", userDep.Accounts[0].Username)
		}
		if !userDep.Accounts[0].IsActive {
			t.Errorf("user key: account should be active, got inactive")
		}
	}
}

func TestGetKeyDeployments_OnlyIncludesDeployedKeys(t *testing.T) {
	// Setup
	originalKM := db.DefaultKeyManager()
	defer func() {
		if originalKM != nil {
			db.SetDefaultKeyManager(originalKM)
		} else {
			db.ClearDefaultKeyManager()
		}
	}()

	fakeKM := &fakeKeyManagerForDeployment{
		allKeys: []model.PublicKey{
			{ID: 1, Comment: "unused-key", Algorithm: "ssh-ed25519", IsGlobal: false},
		},
		accountsByKey: map[int][]model.Account{
			1: {}, // No accounts assigned
		},
	}
	db.SetDefaultKeyManager(fakeKM)

	originalStore := db.GetStore()
	defer func() {
		if originalStore != nil {
			db.SetStore(originalStore)
		}
	}()
	fakeStore := &fakeStoreForDeployment{allAccounts: []model.Account{}}
	db.SetStore(fakeStore)

	// Execute
	deployments, err := GetKeyDeployments()
	if err != nil {
		t.Fatalf("GetKeyDeployments failed: %v", err)
	}

	// Verify: unused key should not be included
	if len(deployments) != 0 {
		t.Errorf("expected 0 deployments for unused key, got %d", len(deployments))
	}
}

// Fake implementations for testing

type fakeKeyManagerForDeployment struct {
	allKeys       []model.PublicKey
	accountsByKey map[int][]model.Account
}

func (f *fakeKeyManagerForDeployment) GetAllPublicKeys() ([]model.PublicKey, error) {
	return f.allKeys, nil
}

func (f *fakeKeyManagerForDeployment) GetAccountsForKey(keyID int) ([]model.Account, error) {
	return f.accountsByKey[keyID], nil
}

// Stubs for other KeyManager methods
func (f *fakeKeyManagerForDeployment) AddPublicKey(algorithm, keyData, comment string, isGlobal bool, expiresAt interface{}) error {
	return nil
}
func (f *fakeKeyManagerForDeployment) AddPublicKeyAndGetModel(algorithm, keyData, comment string, isGlobal bool, expiresAt interface{}) (*model.PublicKey, error) {
	return nil, nil
}
func (f *fakeKeyManagerForDeployment) DeletePublicKey(id int) error       { return nil }
func (f *fakeKeyManagerForDeployment) TogglePublicKeyGlobal(id int) error { return nil }
func (f *fakeKeyManagerForDeployment) SetPublicKeyExpiry(id int, expiresAt interface{}) error {
	return nil
}
func (f *fakeKeyManagerForDeployment) GetPublicKeyByComment(comment string) (*model.PublicKey, error) {
	return nil, nil
}
func (f *fakeKeyManagerForDeployment) GetGlobalPublicKeys() ([]model.PublicKey, error) {
	return nil, nil
}
func (f *fakeKeyManagerForDeployment) AssignKeyToAccount(keyID, accountID int) error { return nil }
func (f *fakeKeyManagerForDeployment) UnassignKeyFromAccount(keyID, accountID int) error {
	return nil
}
func (f *fakeKeyManagerForDeployment) GetKeysForAccount(accountID int) ([]model.PublicKey, error) {
	return nil, nil
}

type fakeStoreForDeployment struct {
	allAccounts []model.Account
}

func (f *fakeStoreForDeployment) GetAllAccounts() ([]model.Account, error) {
	return f.allAccounts, nil
}

// Stubs for other Store methods (minimal implementation to satisfy interface)
func (f *fakeStoreForDeployment) GetActiveSystemKey() (*model.SystemKey, error) { return nil, nil }
func (f *fakeStoreForDeployment) GetSystemKeyBySerial(serial int) (*model.SystemKey, error) {
	return nil, nil
}
func (f *fakeStoreForDeployment) AddAccount(username, hostname, label, tags string) (int, error) {
	return 0, nil
}
func (f *fakeStoreForDeployment) GetAccountByID(id int) (*model.Account, error) { return nil, nil }
func (f *fakeStoreForDeployment) DeleteAccount(id int) error                    { return nil }
func (f *fakeStoreForDeployment) UpdateAccount(account model.Account) error     { return nil }
func (f *fakeStoreForDeployment) LogAction(action, details string) error        { return nil }
func (f *fakeStoreForDeployment) Close() error                                  { return nil }
func (f *fakeStoreForDeployment) BunDB() interface{}                            { return nil }
