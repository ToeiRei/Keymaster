// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package tui

import (
	"errors"
	"testing"
	"time"

	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/model"
)

// fakeKeyManager implements db.KeyManager minimally for tests
type fakeKeyManager struct{}

func (f *fakeKeyManager) AddPublicKey(algorithm, keyData, comment string, isGlobal bool, expiresAt time.Time) error {
	return nil
}
func (f *fakeKeyManager) AddPublicKeyAndGetModel(algorithm, keyData, comment string, isGlobal bool, expiresAt time.Time) (*model.PublicKey, error) {
	return &model.PublicKey{ID: 1, Algorithm: algorithm, KeyData: keyData, Comment: comment}, nil
}
func (f *fakeKeyManager) DeletePublicKey(id int) error                         { return nil }
func (f *fakeKeyManager) TogglePublicKeyGlobal(id int) error                   { return nil }
func (f *fakeKeyManager) SetPublicKeyExpiry(id int, expiresAt time.Time) error { return nil }
func (f *fakeKeyManager) GetAllPublicKeys() ([]model.PublicKey, error) {
	return []model.PublicKey{{ID: 11, Algorithm: "ssh-ed25519", KeyData: "k11"}}, nil
}
func (f *fakeKeyManager) GetPublicKeyByComment(comment string) (*model.PublicKey, error) {
	return &model.PublicKey{ID: 12, Comment: comment}, nil
}
func (f *fakeKeyManager) GetGlobalPublicKeys() ([]model.PublicKey, error) {
	return []model.PublicKey{{ID: 21, Algorithm: "ssh-rsa", KeyData: "g1", Comment: "g1"}}, nil
}
func (f *fakeKeyManager) AssignKeyToAccount(keyID, accountID int) error     { return nil }
func (f *fakeKeyManager) UnassignKeyFromAccount(keyID, accountID int) error { return nil }
func (f *fakeKeyManager) GetKeysForAccount(accountID int) ([]model.PublicKey, error) {
	return []model.PublicKey{{ID: 31, Algorithm: "ssh-ed25519", KeyData: "a1", Comment: "a1"}}, nil
}
func (f *fakeKeyManager) GetAccountsForKey(keyID int) ([]model.Account, error) {
	return []model.Account{{ID: 99}}, nil
}

func TestCoreDeployAdapter_IsPassphraseRequired(t *testing.T) {
	d := coreDeployAdapter{}
	// Behavior: returns a boolean and should be false for generic errors.
	if d.IsPassphraseRequired(errors.New("other")) {
		t.Fatal("unexpected true for non-passphrase error")
	}
}

func TestCoreKeyReader_GetAllPublicKeys_NoManager_ReturnsNil(t *testing.T) {
	db.ClearDefaultKeyManager()
	kr := coreKeyReader{}
	keys, err := kr.GetAllPublicKeys()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if keys != nil && len(keys) != 0 {
		t.Fatalf("expected nil/empty slice when no key manager, got: %v", keys)
	}
}

func TestCoreKeyStore_WithFakeKeyManager(t *testing.T) {
	prev := db.DefaultKeyManager()
	db.SetDefaultKeyManager(&fakeKeyManager{})
	defer db.SetDefaultKeyManager(prev)

	ks := coreKeyStore{}

	gk, err := ks.GetGlobalPublicKeys()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gk) == 0 || gk[0].ID != 21 {
		t.Fatalf("unexpected global keys: %v", gk)
	}

	ak, err := ks.GetKeysForAccount(5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ak) == 0 || ak[0].ID != 31 {
		t.Fatalf("unexpected account keys: %v", ak)
	}

	if err := ks.AssignKeyToAccount(1, 2); err != nil {
		t.Fatalf("Assign returned error: %v", err)
	}
}
