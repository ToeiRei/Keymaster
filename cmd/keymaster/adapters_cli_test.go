// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package main

import (
	"testing"
	"time"

	"github.com/toeirei/keymaster/internal/core"
	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/model"
)

// Test cliKeyGenerator delegates to crypto/ssh generator
func TestCliKeyGenerator_Generate(t *testing.T) {
	gen := &cliKeyGenerator{}
	pub, priv, err := gen.GenerateAndMarshalEd25519Key("test", "")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if pub == "" || priv == "" {
		t.Fatalf("expected non-empty key material")
	}
}

// Minimal fake AccountManager for testing cliStoreAdapter Add/Delete/Assign
type fakeAccountManager struct{}

func (f *fakeAccountManager) AddAccount(username, hostname, label, tags string) (int, error) {
	return 42, nil
}
func (f *fakeAccountManager) DeleteAccount(id int) error { return nil }

// Minimal fake KeyManager for assign
// fakeKeyManager implements db.KeyManager with minimal stubs for tests.
type fakeKeyManager struct{}

func (f *fakeKeyManager) AddPublicKey(algorithm, keyData, comment string, isGlobal bool, expiresAt time.Time) error {
	return nil
}
func (f *fakeKeyManager) AddPublicKeyAndGetModel(algorithm, keyData, comment string, isGlobal bool, expiresAt time.Time) (*model.PublicKey, error) {
	return &model.PublicKey{Comment: comment}, nil
}
func (f *fakeKeyManager) DeletePublicKey(id int) error                         { return nil }
func (f *fakeKeyManager) TogglePublicKeyGlobal(id int) error                   { return nil }
func (f *fakeKeyManager) SetPublicKeyExpiry(id int, expiresAt time.Time) error { return nil }
func (f *fakeKeyManager) GetAllPublicKeys() ([]model.PublicKey, error)         { return nil, nil }
func (f *fakeKeyManager) GetPublicKeyByComment(comment string) (*model.PublicKey, error) {
	return &model.PublicKey{Comment: comment}, nil
}
func (f *fakeKeyManager) GetGlobalPublicKeys() ([]model.PublicKey, error)   { return nil, nil }
func (f *fakeKeyManager) AssignKeyToAccount(keyID, accountID int) error     { return nil }
func (f *fakeKeyManager) UnassignKeyFromAccount(keyID, accountID int) error { return nil }
func (f *fakeKeyManager) GetKeysForAccount(accountID int) ([]model.PublicKey, error) {
	return nil, nil
}
func (f *fakeKeyManager) GetAccountsForKey(keyID int) ([]model.Account, error) { return nil, nil }

func TestCliStoreAdapter_AddDeleteAssign(t *testing.T) {
	// inject fakes into db package defaults
	prevAcct := db.DefaultAccountManager()
	prevKey := db.DefaultKeyManager()
	db.SetDefaultAccountManager(&fakeAccountManager{})
	db.SetDefaultKeyManager(&fakeKeyManager{})
	defer db.SetDefaultAccountManager(prevAcct)
	defer db.SetDefaultKeyManager(prevKey)

	a := &cliStoreAdapter{}
	id, err := a.AddAccount("u", "h", "l", "t")
	if err != nil {
		t.Fatalf("AddAccount failed: %v", err)
	}
	if id != 42 {
		t.Fatalf("unexpected id %d", id)
	}

	if err := a.DeleteAccount(42); err != nil {
		t.Fatalf("DeleteAccount failed: %v", err)
	}

	if err := a.AssignKeyToAccount(1, 2); err != nil {
		t.Fatalf("AssignKeyToAccount failed: %v", err)
	}
}

// Fake deployer functions used to test cliDeployerManager adapters
func TestCliDeployerManager_DeployAndAudit_Decommission(t *testing.T) {
	dm := &cliDeployerManager{}

	// Use a simple account value
	acct := model.Account{ID: 1, Username: "u", Hostname: "h", Label: "l"}

	// Test purely local helpers
	if host := dm.CanonicalizeHostPort("example.com"); host == "" {
		t.Fatalf("CanonicalizeHostPort returned empty")
	}
	if _, _, err := dm.ParseHostPort("example.com:22"); err != nil {
		t.Fatalf("ParseHostPort failed: %v", err)
	}

	// Test decommission path that avoids remote cleanup by injecting fake account manager
	prevAcct := db.DefaultAccountManager()
	db.SetDefaultAccountManager(&fakeAccountManager{})
	defer db.SetDefaultAccountManager(prevAcct)
	// Skip remote cleanup to avoid network operations
	res, err := dm.DecommissionAccount(acct, "syskey", core.DecommissionOptions{SkipRemoteCleanup: true})
	if err != nil {
		t.Fatalf("DecommissionAccount adapter returned error: %v", err)
	}
	_ = res
}
