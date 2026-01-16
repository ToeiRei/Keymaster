// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package db

import (
	"testing"
	"time"
)

// Test that various mutators mark accounts as dirty when they affect authorized_keys.
func TestIsDirtyFlags(t *testing.T) {
	WithTestStore(t, func(s *BunStore) {
		bdb := s.bun

		// Create two accounts
		a1, err := AddAccountBun(bdb, "u1", "h1", "lbl1", "")
		if err != nil {
			t.Fatalf("AddAccountBun a1 failed: %v", err)
		}
		a2, err := AddAccountBun(bdb, "u2", "h2", "lbl2", "")
		if err != nil {
			t.Fatalf("AddAccountBun a2 failed: %v", err)
		}

		// New accounts are marked dirty by AddAccountBun; clear for controlled tests
		if err := UpdateAccountIsDirtyBun(bdb, a1, false); err != nil {
			t.Fatalf("clear dirty a1 failed: %v", err)
		}
		if err := UpdateAccountIsDirtyBun(bdb, a2, false); err != nil {
			t.Fatalf("clear dirty a2 failed: %v", err)
		}

		// Add a public key (non-global) and assign to a1
		if err := AddPublicKeyBun(bdb, "ed25519", "AAAAB3", "k1", false, time.Time{}); err != nil {
			t.Fatalf("AddPublicKeyBun failed: %v", err)
		}
		pk, err := GetPublicKeyByCommentBun(bdb, "k1")
		if err != nil || pk == nil {
			t.Fatalf("GetPublicKeyByCommentBun failed: %v pk=%v", err, pk)
		}

		// Assign key -> should mark a1 dirty
		if err := AssignKeyToAccountBun(bdb, pk.ID, a1); err != nil {
			t.Fatalf("AssignKeyToAccountBun failed: %v", err)
		}
		a, err := GetAccountByIDBun(bdb, a1)
		if err != nil {
			t.Fatalf("GetAccountByIDBun failed: %v", err)
		}
		if !a.IsDirty {
			t.Fatalf("expected account a1 to be dirty after assign")
		}

		// Clear and then unassign -> should mark dirty again
		if err := UpdateAccountIsDirtyBun(bdb, a1, false); err != nil {
			t.Fatalf("clear dirty a1 failed: %v", err)
		}
		if err := UnassignKeyFromAccountBun(bdb, pk.ID, a1); err != nil {
			t.Fatalf("UnassignKeyFromAccountBun failed: %v", err)
		}
		a, _ = GetAccountByIDBun(bdb, a1)
		if !a.IsDirty {
			t.Fatalf("expected account a1 to be dirty after unassign")
		}

		// Clear both; add a global key -> should mark all accounts dirty
		if err := UpdateAccountIsDirtyBun(bdb, a1, false); err != nil {
			t.Fatalf("clear dirty a1 failed: %v", err)
		}
		if err := UpdateAccountIsDirtyBun(bdb, a2, false); err != nil {
			t.Fatalf("clear dirty a2 failed: %v", err)
		}
		if err := AddPublicKeyBun(bdb, "ed25519", "GGGG", "gk", true, time.Time{}); err != nil {
			t.Fatalf("AddPublicKeyBun global failed: %v", err)
		}
		a, _ = GetAccountByIDBun(bdb, a1)
		b, _ := GetAccountByIDBun(bdb, a2)
		if !a.IsDirty || !b.IsDirty {
			t.Fatalf("expected all accounts dirty after adding global key: a1=%v a2=%v", a.IsDirty, b.IsDirty)
		}

		// Clear and test toggle global
		if err := UpdateAccountIsDirtyBun(bdb, a1, false); err != nil {
			t.Fatalf("clear dirty a1 failed: %v", err)
		}
		if err := UpdateAccountIsDirtyBun(bdb, a2, false); err != nil {
			t.Fatalf("clear dirty a2 failed: %v", err)
		}
		// Create a non-global key then toggle it to global
		if err := AddPublicKeyBun(bdb, "ed25519", "HHHH", "tk", false, time.Time{}); err != nil {
			t.Fatalf("AddPublicKeyBun tk failed: %v", err)
		}
		tpk, _ := GetPublicKeyByCommentBun(bdb, "tk")
		if tpk == nil {
			t.Fatalf("expected tk key present")
		}
		if err := TogglePublicKeyGlobalBun(bdb, tpk.ID); err != nil {
			t.Fatalf("TogglePublicKeyGlobalBun failed: %v", err)
		}
		a, _ = GetAccountByIDBun(bdb, a1)
		b, _ = GetAccountByIDBun(bdb, a2)
		if !a.IsDirty || !b.IsDirty {
			t.Fatalf("expected all accounts dirty after toggling key global: a1=%v a2=%v", a.IsDirty, b.IsDirty)
		}

		// Clear and test expiry marking for assigned accounts
		if err := UpdateAccountIsDirtyBun(bdb, a1, false); err != nil {
			t.Fatalf("clear dirty a1 failed: %v", err)
		}
		if err := UpdateAccountIsDirtyBun(bdb, a2, false); err != nil {
			t.Fatalf("clear dirty a2 failed: %v", err)
		}
		// Add key assign to a1
		if err := AddPublicKeyBun(bdb, "ed25519", "IIII", "ek", false, time.Time{}); err != nil {
			t.Fatalf("AddPublicKeyBun ek failed: %v", err)
		}
		ek, _ := GetPublicKeyByCommentBun(bdb, "ek")
		if ek == nil {
			t.Fatalf("expected ek present")
		}
		if err := AssignKeyToAccountBun(bdb, ek.ID, a1); err != nil {
			t.Fatalf("Assign ek failed: %v", err)
		}
		// Clear a1 then set expiry -> should mark a1 dirty
		if err := UpdateAccountIsDirtyBun(bdb, a1, false); err != nil {
			t.Fatalf("clear dirty a1 failed: %v", err)
		}
		if err := SetPublicKeyExpiryBun(bdb, ek.ID, time.Now().Add(24*time.Hour)); err != nil {
			t.Fatalf("SetPublicKeyExpiryBun failed: %v", err)
		}
		a, _ = GetAccountByIDBun(bdb, a1)
		if !a.IsDirty {
			t.Fatalf("expected a1 dirty after key expiry change")
		}

		// Clear and test delete marks assigned accounts
		if err := UpdateAccountIsDirtyBun(bdb, a1, false); err != nil {
			t.Fatalf("clear dirty a1 failed: %v", err)
		}
		// Delete ek
		if err := DeletePublicKeyBun(bdb, ek.ID); err != nil {
			t.Fatalf("DeletePublicKeyBun failed: %v", err)
		}
		a, _ = GetAccountByIDBun(bdb, a1)
		if !a.IsDirty {
			t.Fatalf("expected a1 dirty after deleting assigned key")
		}

		// Clear and test system key rotation marks all dirty
		if err := UpdateAccountIsDirtyBun(bdb, a1, false); err != nil {
			t.Fatalf("clear dirty a1 failed: %v", err)
		}
		if err := UpdateAccountIsDirtyBun(bdb, a2, false); err != nil {
			t.Fatalf("clear dirty a2 failed: %v", err)
		}
		if _, err := RotateSystemKeyBun(bdb, "npub", "npriv"); err != nil {
			t.Fatalf("RotateSystemKeyBun failed: %v", err)
		}
		a, _ = GetAccountByIDBun(bdb, a1)
		b, _ = GetAccountByIDBun(bdb, a2)
		if !a.IsDirty || !b.IsDirty {
			t.Fatalf("expected all accounts dirty after rotation: a1=%v a2=%v", a.IsDirty, b.IsDirty)
		}
	})
}
