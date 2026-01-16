// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package db

import (
	"context"
	"testing"
	"time"
)

func TestAssignAndUnassignMarksDirty(t *testing.T) {
	WithTestStore(t, func(s *BunStore) {
		ctx := context.Background()
		bdb := s.BunDB()

		// Create account and public key
		if _, err := ExecRaw(ctx, bdb, "INSERT INTO accounts (id, username, hostname, label, tags, serial, is_active, is_dirty) VALUES (?, ?, ?, ?, ?, ?, ?, ?)", 401, "ua", "ha", "", "", 1, true, false); err != nil {
			t.Fatalf("insert account failed: %v", err)
		}
		res, err := ExecRaw(ctx, bdb, "INSERT INTO public_keys (algorithm, key_data, comment, is_global) VALUES (?, ?, ?, ?)", "ssh-ed25519", "AAAAB3NzaC1yc2E...", "c1", false)
		if err != nil {
			t.Fatalf("insert key failed: %v", err)
		}
		id64, _ := res.LastInsertId()
		keyID := int(id64)

		// Assign key -> account
		if err := AssignKeyToAccountBun(bdb, keyID, 401); err != nil {
			t.Fatalf("AssignKeyToAccountBun failed: %v", err)
		}

		var dirty bool
		if err := QueryRawInto(ctx, bdb, &dirty, "SELECT is_dirty FROM accounts WHERE id = ?", 401); err != nil {
			t.Fatalf("select is_dirty failed: %v", err)
		}
		if !dirty {
			t.Fatalf("expected account to be marked dirty after assign")
		}

		// Clear dirty and unassign
		if _, err := ExecRaw(ctx, bdb, "UPDATE accounts SET is_dirty = ? WHERE id = ?", false, 401); err != nil {
			t.Fatalf("clear dirty failed: %v", err)
		}
		if err := UnassignKeyFromAccountBun(bdb, keyID, 401); err != nil {
			t.Fatalf("UnassignKeyFromAccountBun failed: %v", err)
		}
		if err := QueryRawInto(ctx, bdb, &dirty, "SELECT is_dirty FROM accounts WHERE id = ?", 401); err != nil {
			t.Fatalf("select is_dirty after unassign failed: %v", err)
		}
		if !dirty {
			t.Fatalf("expected account to be marked dirty after unassign")
		}
	})
}

func TestTogglePublicKeyGlobal_MarksAllAccountsDirty(t *testing.T) {
	WithTestStore(t, func(s *BunStore) {
		ctx := context.Background()
		bdb := s.BunDB()

		// Two accounts
		if _, err := ExecRaw(ctx, bdb, "INSERT INTO accounts (id, username, hostname, label, tags, serial, is_active, is_dirty) VALUES (?, ?, ?, ?, ?, ?, ?, ?)", 501, "a1", "h1", "", "", 1, true, false); err != nil {
			t.Fatalf("insert account1 failed: %v", err)
		}
		if _, err := ExecRaw(ctx, bdb, "INSERT INTO accounts (id, username, hostname, label, tags, serial, is_active, is_dirty) VALUES (?, ?, ?, ?, ?, ?, ?, ?)", 502, "a2", "h2", "", "", 1, true, false); err != nil {
			t.Fatalf("insert account2 failed: %v", err)
		}

		// Insert key (initially non-global)
		res, err := ExecRaw(ctx, bdb, "INSERT INTO public_keys (algorithm, key_data, comment, is_global) VALUES (?, ?, ?, ?)", "ssh-ed25519", "AAAAB3NzaC1yc2E...", "g1", false)
		if err != nil {
			t.Fatalf("insert key failed: %v", err)
		}
		id64, _ := res.LastInsertId()
		keyID := int(id64)

		// Toggle to global: should mark all accounts dirty via MaybeMarkAccountDirtyTx
		if err := TogglePublicKeyGlobalBun(bdb, keyID); err != nil {
			t.Fatalf("TogglePublicKeyGlobalBun failed: %v", err)
		}

		var dirty1, dirty2 bool
		if err := QueryRawInto(ctx, bdb, &dirty1, "SELECT is_dirty FROM accounts WHERE id = ?", 501); err != nil {
			t.Fatalf("select is_dirty a1 failed: %v", err)
		}
		if err := QueryRawInto(ctx, bdb, &dirty2, "SELECT is_dirty FROM accounts WHERE id = ?", 502); err != nil {
			t.Fatalf("select is_dirty a2 failed: %v", err)
		}
		if !dirty1 || !dirty2 {
			t.Fatalf("expected both accounts to be marked dirty after toggling global")
		}

		// Also expect audit entries for ACCOUNT_KEY_HASH_UPDATED (>=2)
		var count int
		if err := QueryRawInto(ctx, bdb, &count, "SELECT COUNT(id) FROM audit_log WHERE action = ?", "ACCOUNT_KEY_HASH_UPDATED"); err != nil {
			t.Fatalf("count audit failed: %v", err)
		}
		if count < 2 {
			t.Fatalf("expected at least 2 ACCOUNT_KEY_HASH_UPDATED audit rows, got %d", count)
		}
	})
}

func TestSetPublicKeyExpiry_AssignedKey_MarksAssignedAccounts(t *testing.T) {
	WithTestStore(t, func(s *BunStore) {
		ctx := context.Background()
		bdb := s.BunDB()

		// Create account
		if _, err := ExecRaw(ctx, bdb, "INSERT INTO accounts (id, username, hostname, label, tags, serial, is_active, is_dirty) VALUES (?, ?, ?, ?, ?, ?, ?, ?)", 601, "ux", "hx", "", "", 1, true, false); err != nil {
			t.Fatalf("insert account failed: %v", err)
		}
		// Create key and assign to account
		res, err := ExecRaw(ctx, bdb, "INSERT INTO public_keys (algorithm, key_data, comment, is_global) VALUES (?, ?, ?, ?)", "ssh-ed25519", "AAAAB3NzaC1yc2E...", "e1", false)
		if err != nil {
			t.Fatalf("insert key failed: %v", err)
		}
		id64, _ := res.LastInsertId()
		keyID := int(id64)
		if _, err := ExecRaw(ctx, bdb, "INSERT INTO account_keys (key_id, account_id) VALUES (?, ?)", keyID, 601); err != nil {
			t.Fatalf("assign key failed: %v", err)
		}

		// Clear any prior dirty
		if _, err := ExecRaw(ctx, bdb, "UPDATE accounts SET is_dirty = ? WHERE id = ?", false, 601); err != nil {
			t.Fatalf("clear dirty failed: %v", err)
		}

		// Set expiry to now (non-zero) -- should mark assigned accounts dirty
		if err := SetPublicKeyExpiryBun(bdb, keyID, time.Now().Add(24*time.Hour)); err != nil {
			t.Fatalf("SetPublicKeyExpiryBun failed: %v", err)
		}

		var dirty bool
		if err := QueryRawInto(ctx, bdb, &dirty, "SELECT is_dirty FROM accounts WHERE id = ?", 601); err != nil {
			t.Fatalf("select is_dirty failed: %v", err)
		}
		if !dirty {
			t.Fatalf("expected assigned account to be marked dirty after expiry change")
		}
	})
}
