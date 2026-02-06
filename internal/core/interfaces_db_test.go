// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core

import (
	"testing"

	"github.com/toeirei/keymaster/internal/core/model"
)

type fakeKeyLister struct{}

func (f fakeKeyLister) GetGlobalPublicKeys() ([]model.PublicKey, error) {
	return []model.PublicKey{{ID: 1, Algorithm: "ssh-ed25519", KeyData: "X"}}, nil
}
func (f fakeKeyLister) GetKeysForAccount(accountID int) ([]model.PublicKey, error) {
	return []model.PublicKey{}, nil
}
func (f fakeKeyLister) GetAllPublicKeys() ([]model.PublicKey, error) {
	return []model.PublicKey{}, nil
}

type fakeUpdater struct{}

func (f fakeUpdater) UpdateAccountSerial(accountID int, serial int) error { return nil }

func TestCoreDBInterfacesCompile(t *testing.T) {
	// Ensure static assignment satisfies the interfaces.
	var _ KeyLister = fakeKeyLister{}
	var _ AccountSerialUpdater = fakeUpdater{}

	kl := fakeKeyLister{}
	keys, err := kl.GetGlobalPublicKeys()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}
}
