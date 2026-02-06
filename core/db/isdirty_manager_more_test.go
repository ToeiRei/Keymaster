// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package db

import (
	"testing"
	"time"
)

// Edge-case tests: toggling a key back to non-global, deleting a global key,
// and rotating the system key through package-level APIs should mark accounts
// dirty conservatively.
func TestIsDirtyEdgeCases(t *testing.T) {
	WithTestStore(t, func(s *BunStore) {
		am := DefaultAccountManager()
		if am == nil {
			t.Fatalf("DefaultAccountManager returned nil")
		}
		km := DefaultKeyManager()
		if km == nil {
			t.Fatalf("DefaultKeyManager returned nil")
		}

		// Setup: two accounts
		a1, err := am.AddAccount("edge1", "h1", "l1", "")
		if err != nil {
			t.Fatalf("AddAccount failed: %v", err)
		}
		a2, err := am.AddAccount("edge2", "h2", "l2", "")
		if err != nil {
			t.Fatalf("AddAccount failed: %v", err)
		}
		// Clear dirty flags
		if err := UpdateAccountIsDirty(a1, false); err != nil {
			t.Fatalf("clear dirty a1 failed: %v", err)
		}
		if err := UpdateAccountIsDirty(a2, false); err != nil {
			t.Fatalf("clear dirty a2 failed: %v", err)
		}

		// 1) Toggle back to non-global: create global key then toggle twice
		pk, err := km.AddPublicKeyAndGetModel("ssh-ed25519", "TG1", "toggle-edge", true, time.Time{})
		if err != nil || pk == nil {
			t.Fatalf("AddPublicKeyAndGetModel failed: %v pk=%v", err, pk)
		}
		// After adding global key both accounts should be dirty
		a, _ := GetAccountByIDBun(s.bun, a1)
		b, _ := GetAccountByIDBun(s.bun, a2)
		if !a.IsDirty || !b.IsDirty {
			t.Fatalf("expected dirty after adding global key: a1=%v a2=%v", a.IsDirty, b.IsDirty)
		}
		// Clear and toggle twice: global -> non-global -> global? We'll toggle off then on
		if err := UpdateAccountIsDirty(a1, false); err != nil {
			t.Fatalf("clear dirty a1 failed: %v", err)
		}
		if err := UpdateAccountIsDirty(a2, false); err != nil {
			t.Fatalf("clear dirty a2 failed: %v", err)
		}
		if err := km.TogglePublicKeyGlobal(pk.ID); err != nil {
			t.Fatalf("TogglePublicKeyGlobal failed: %v", err)
		}
		// Toggling to non-global should conservatively mark assigned accounts dirty;
		// since this key is global and not assigned, treat global->non-global as marking all dirty.
		a, _ = GetAccountByIDBun(s.bun, a1)
		b, _ = GetAccountByIDBun(s.bun, a2)
		if !a.IsDirty || !b.IsDirty {
			t.Fatalf("expected dirty after toggling global->non-global: a1=%v a2=%v", a.IsDirty, b.IsDirty)
		}

		// 2) Delete a global key -> should mark all accounts dirty
		if err := UpdateAccountIsDirty(a1, false); err != nil {
			t.Fatalf("clear dirty a1 failed: %v", err)
		}
		if err := UpdateAccountIsDirty(a2, false); err != nil {
			t.Fatalf("clear dirty a2 failed: %v", err)
		}
		// Ensure key is global again (toggle back)
		if err := km.TogglePublicKeyGlobal(pk.ID); err != nil {
			t.Fatalf("TogglePublicKeyGlobal to global failed: %v", err)
		}
		if err := km.DeletePublicKey(pk.ID); err != nil {
			t.Fatalf("DeletePublicKey failed: %v", err)
		}
		a, _ = GetAccountByIDBun(s.bun, a1)
		b, _ = GetAccountByIDBun(s.bun, a2)
		if !a.IsDirty || !b.IsDirty {
			t.Fatalf("expected all accounts dirty after deleting global key: a1=%v a2=%v", a.IsDirty, b.IsDirty)
		}

		// 3) Rotate system key via package API -> should mark all accounts dirty
		if err := UpdateAccountIsDirty(a1, false); err != nil {
			t.Fatalf("clear dirty a1 failed: %v", err)
		}
		if err := UpdateAccountIsDirty(a2, false); err != nil {
			t.Fatalf("clear dirty a2 failed: %v", err)
		}
		if _, err := RotateSystemKey("npub-edge", "npriv-edge"); err != nil {
			t.Fatalf("RotateSystemKey failed: %v", err)
		}
		a, _ = GetAccountByIDBun(s.bun, a1)
		b, _ = GetAccountByIDBun(s.bun, a2)
		if !a.IsDirty || !b.IsDirty {
			t.Fatalf("expected all accounts dirty after RotateSystemKey: a1=%v a2=%v", a.IsDirty, b.IsDirty)
		}
	})
}
