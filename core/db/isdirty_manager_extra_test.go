// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package db

import (
	"context"
	"testing"
	"time"
)

// Additional manager-level tests to ensure toggle/delete operations mark
// accounts dirty as expected when invoked through DefaultKeyManager.
func TestToggleAndDeleteMarkDirtyViaManagers(t *testing.T) {
	WithTestStore(t, func(s *BunStore) {
		am := DefaultAccountManager()
		if am == nil {
			t.Fatalf("DefaultAccountManager returned nil")
		}
		km := DefaultKeyManager()
		if km == nil {
			t.Fatalf("DefaultKeyManager returned nil")
		}

		// Create accounts and clear dirty
		a1, err := am.AddAccount("u1", "h1", "l1", "")
		if err != nil {
			t.Fatalf("AddAccount failed: %v", err)
		}
		a2, err := am.AddAccount("u2", "h2", "l2", "")
		if err != nil {
			t.Fatalf("AddAccount failed: %v", err)
		}
		if err := UpdateAccountIsDirty(a1, false); err != nil {
			t.Fatalf("clear dirty a1 failed: %v", err)
		}
		if _, err := ExecRaw(context.Background(), s.bun, "UPDATE accounts SET key_hash = NULL WHERE id = ?", a1); err != nil {
			t.Fatalf("clear key_hash a1 failed: %v", err)
		}
		if err := UpdateAccountIsDirty(a2, false); err != nil {
			t.Fatalf("clear dirty a2 failed: %v", err)
		}
		if _, err := ExecRaw(context.Background(), s.bun, "UPDATE accounts SET key_hash = NULL WHERE id = ?", a2); err != nil {
			t.Fatalf("clear key_hash a2 failed: %v", err)
		}

		// Add non-global key, assign to a1
		pk, err := km.AddPublicKeyAndGetModel("ssh-ed25519", "AAAAB3", "t-assign", false, time.Time{})
		if err != nil || pk == nil {
			t.Fatalf("AddPublicKeyAndGetModel failed: %v pk=%v", err, pk)
		}
		if err := km.AssignKeyToAccount(pk.ID, a1); err != nil {
			t.Fatalf("AssignKeyToAccount failed: %v", err)
		}

		// Clear and toggle the key to global via manager -> both accounts dirty
		if err := UpdateAccountIsDirty(a1, false); err != nil {
			t.Fatalf("clear dirty a1 failed: %v", err)
		}
		if _, err := ExecRaw(context.Background(), s.bun, "UPDATE accounts SET key_hash = NULL WHERE id = ?", a1); err != nil {
			t.Fatalf("clear key_hash a1 failed: %v", err)
		}
		if err := UpdateAccountIsDirty(a2, false); err != nil {
			t.Fatalf("clear dirty a2 failed: %v", err)
		}
		if _, err := ExecRaw(context.Background(), s.bun, "UPDATE accounts SET key_hash = NULL WHERE id = ?", a2); err != nil {
			t.Fatalf("clear key_hash a2 failed: %v", err)
		}
		if err := km.TogglePublicKeyGlobal(pk.ID); err != nil {
			t.Fatalf("TogglePublicKeyGlobal failed: %v", err)
		}
		a, _ := GetAccountByIDBun(s.bun, a1)
		b, _ := GetAccountByIDBun(s.bun, a2)
		if !a.IsDirty || !b.IsDirty {
			t.Fatalf("expected all accounts dirty after toggling key global: a1=%v a2=%v", a.IsDirty, b.IsDirty)
		}

		// Clear both, add a key assigned to a1 then delete via manager -> a1 dirty
		if err := UpdateAccountIsDirty(a1, false); err != nil {
			t.Fatalf("clear dirty a1 failed: %v", err)
		}
		if _, err := ExecRaw(context.Background(), s.bun, "UPDATE accounts SET key_hash = NULL WHERE id = ?", a1); err != nil {
			t.Fatalf("clear key_hash a1 failed: %v", err)
		}
		if err := UpdateAccountIsDirty(a2, false); err != nil {
			t.Fatalf("clear dirty a2 failed: %v", err)
		}
		ek, err := km.AddPublicKeyAndGetModel("ssh-ed25519", "BBBBB", "t-delete", false, time.Time{})
		if err != nil || ek == nil {
			t.Fatalf("AddPublicKeyAndGetModel ek failed: %v", err)
		}
		if err := km.AssignKeyToAccount(ek.ID, a1); err != nil {
			t.Fatalf("Assign ek failed: %v", err)
		}
		if err := UpdateAccountIsDirty(a1, false); err != nil {
			t.Fatalf("clear dirty a1 failed: %v", err)
		}
		if err := km.DeletePublicKey(ek.ID); err != nil {
			t.Fatalf("DeletePublicKey failed: %v", err)
		}
		a, _ = GetAccountByIDBun(s.bun, a1)
		if !a.IsDirty {
			t.Fatalf("expected a1 dirty after deleting assigned key")
		}
	})
}
