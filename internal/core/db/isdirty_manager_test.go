// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package db

import (
	"testing"
	"time"
)

// Verify that using the package-level managers (DefaultKeyManager / DefaultAccountManager)
// results in the expected is_dirty semantics at the account level.
func TestIsDirtyFlagsViaManagers(t *testing.T) {
	WithTestStore(t, func(s *BunStore) {
		// Use managers rather than direct Bun helpers
		am := DefaultAccountManager()
		if am == nil {
			t.Fatalf("DefaultAccountManager returned nil")
		}
		km := DefaultKeyManager()
		if km == nil {
			t.Fatalf("DefaultKeyManager returned nil")
		}

		// Create two accounts
		a1, err := am.AddAccount("um1", "h1", "l1", "")
		if err != nil {
			t.Fatalf("AddAccount failed: %v", err)
		}
		a2, err := am.AddAccount("um2", "h2", "l2", "")
		if err != nil {
			t.Fatalf("AddAccount failed: %v", err)
		}

		// Clear dirty for controlled checks
		if err := UpdateAccountIsDirty(a1, false); err != nil {
			t.Fatalf("clear dirty a1 failed: %v", err)
		}
		if err := UpdateAccountIsDirty(a2, false); err != nil {
			t.Fatalf("clear dirty a2 failed: %v", err)
		}

		// Add a non-global key and assign to a1 via managers
		pk, err := km.AddPublicKeyAndGetModel("ed25519", "AAAAB3", "m1", false, time.Time{})
		if err != nil || pk == nil {
			t.Fatalf("AddPublicKeyAndGetModel failed: %v pk=%v", err, pk)
		}
		if err := km.AssignKeyToAccount(pk.ID, a1); err != nil {
			t.Fatalf("AssignKeyToAccount failed: %v", err)
		}

		// Fetch account via Bun helper and assert dirty
		a, err := GetAccountByIDBun(s.bun, a1)
		if err != nil {
			t.Fatalf("GetAccountByIDBun failed: %v", err)
		}
		if !a.IsDirty {
			t.Fatalf("expected account a1 to be dirty after assign")
		}

		// Clear both, then add a global key via manager -> both should be dirty
		if err := UpdateAccountIsDirty(a1, false); err != nil {
			t.Fatalf("clear dirty a1 failed: %v", err)
		}
		if err := UpdateAccountIsDirty(a2, false); err != nil {
			t.Fatalf("clear dirty a2 failed: %v", err)
		}
		if _, err := km.AddPublicKeyAndGetModel("ed25519", "GGGG", "gk", true, time.Time{}); err != nil {
			t.Fatalf("AddPublicKeyAndGetModel global failed: %v", err)
		}
		a, _ = GetAccountByIDBun(s.bun, a1)
		b, _ := GetAccountByIDBun(s.bun, a2)
		if !a.IsDirty || !b.IsDirty {
			t.Fatalf("expected all accounts dirty after adding global key: a1=%v a2=%v", a.IsDirty, b.IsDirty)
		}

		// Clear both, create a key assigned to a1 then SetPublicKeyExpiry via manager
		if err := UpdateAccountIsDirty(a1, false); err != nil {
			t.Fatalf("clear dirty a1 failed: %v", err)
		}
		if err := UpdateAccountIsDirty(a2, false); err != nil {
			t.Fatalf("clear dirty a2 failed: %v", err)
		}
		ek, err := km.AddPublicKeyAndGetModel("ed25519", "IIII", "ek", false, time.Time{})
		if err != nil || ek == nil {
			t.Fatalf("AddPublicKeyAndGetModel ek failed: %v", err)
		}
		if err := km.AssignKeyToAccount(ek.ID, a1); err != nil {
			t.Fatalf("Assign ek failed: %v", err)
		}
		if err := UpdateAccountIsDirty(a1, false); err != nil {
			t.Fatalf("clear dirty a1 failed: %v", err)
		}
		if err := km.SetPublicKeyExpiry(ek.ID, time.Now().Add(24*time.Hour)); err != nil {
			t.Fatalf("SetPublicKeyExpiry failed: %v", err)
		}
		a, _ = GetAccountByIDBun(s.bun, a1)
		if !a.IsDirty {
			t.Fatalf("expected a1 dirty after key expiry change")
		}
	})
}
