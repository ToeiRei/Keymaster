// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package db

import (
	"testing"
	"time"
)

func TestPublicKeyExpiryToggleDeleteAssignFlow(t *testing.T) {
	WithTestStore(t, func(s *BunStore) {
		bdb := s.BunDB()

		// Create an account
		aid, err := AddAccountBun(bdb, "u", "h", "lbl", "")
		if err != nil {
			t.Fatalf("AddAccountBun: %v", err)
		}

		// Add a non-global key
		if err := AddPublicKeyBun(bdb, "ssh-ed25519", "DATA", "k1", false, time.Time{}); err != nil {
			t.Fatalf("AddPublicKeyBun: %v", err)
		}
		pk, err := GetPublicKeyByCommentBun(bdb, "k1")
		if err != nil {
			t.Fatalf("GetPublicKeyByCommentBun: %v", err)
		}
		if pk == nil {
			t.Fatalf("expected pk")
		}

		// Assign key to account and ensure account marked dirty
		if err := AssignKeyToAccountBun(bdb, pk.ID, aid); err != nil {
			t.Fatalf("AssignKeyToAccountBun: %v", err)
		}
		acc, err := GetAccountByIDBun(bdb, aid)
		if err != nil {
			t.Fatalf("GetAccountByIDBun: %v", err)
		}
		if acc == nil || !acc.IsDirty {
			t.Fatalf("expected account dirty after assign: %+v", acc)
		}

		// Clear dirty for test
		if err := UpdateAccountIsDirtyBun(bdb, aid, false); err != nil {
			t.Fatalf("clear dirty: %v", err)
		}

		// Set expiry on non-global key: should mark account dirty
		if err := SetPublicKeyExpiryBun(bdb, pk.ID, time.Now().Add(24*time.Hour)); err != nil {
			t.Fatalf("SetPublicKeyExpiryBun: %v", err)
		}
		acc2, _ := GetAccountByIDBun(bdb, aid)
		if acc2 == nil || !acc2.IsDirty {
			t.Fatalf("expected account dirty after expiry set: %+v", acc2)
		}

		// Make key global
		if err := TogglePublicKeyGlobalBun(bdb, pk.ID); err != nil {
			t.Fatalf("TogglePublicKeyGlobalBun: %v", err)
		}
		// Toggle should mark all accounts dirty (account remains dirty)
		acc3, _ := GetAccountByIDBun(bdb, aid)
		if acc3 == nil || !acc3.IsDirty {
			t.Fatalf("expected account dirty after toggle global: %+v", acc3)
		}

		// Delete key: should remove key and keep accounts dirty
		if err := DeletePublicKeyBun(bdb, pk.ID); err != nil {
			t.Fatalf("DeletePublicKeyBun: %v", err)
		}
		pk2, err := GetPublicKeyByIDBun(bdb, pk.ID)
		if err != nil {
			t.Fatalf("GetPublicKeyByIDBun: %v", err)
		}
		if pk2 != nil {
			t.Fatalf("expected pk deleted")
		}

		// Unassigning non-existing key should not error (idempotent)
		if err := UnassignKeyFromAccountBun(bdb, pk.ID, aid); err != nil {
			t.Fatalf("UnassignKeyFromAccountBun: %v", err)
		}

		// Ensure account dirty flag still present
		acc4, _ := GetAccountByIDBun(bdb, aid)
		if acc4 == nil || !acc4.IsDirty {
			t.Fatalf("expected account dirty at end: %+v", acc4)
		}

		// Basic GetKeysForAccount/ GetAccountsForKey return without panics
		if _, err := GetKeysForAccountBun(bdb, aid); err != nil {
			t.Fatalf("GetKeysForAccountBun: %v", err)
		}
		if _, err := GetAccountsForKeyBun(bdb, pk.ID); err != nil {
			t.Fatalf("GetAccountsForKeyBun: %v", err)
		}

		// Clean up: Unassign was called; DeleteAccount should succeed
		if err := DeleteAccountBun(bdb, aid); err != nil {
			t.Fatalf("DeleteAccountBun: %v", err)
		}
	})
}
