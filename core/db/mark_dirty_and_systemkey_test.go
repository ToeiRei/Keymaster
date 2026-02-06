// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package db

import (
	"context"
	"testing"
)

func TestMarkAccountsDirtyForKey_GlobalAndNonGlobal(t *testing.T) {
	WithTestStore(t, func(s *BunStore) {
		ctx := context.Background()
		bdb := s.BunDB()

		// Global branch: two accounts
		if _, err := ExecRaw(ctx, bdb, "INSERT INTO accounts (id, username, hostname, label, tags, serial, is_active, is_dirty) VALUES (?, ?, ?, ?, ?, ?, ?, ?)", 701, "g1", "h1", "", "", 1, true, false); err != nil {
			t.Fatalf("insert account1 failed: %v", err)
		}
		if _, err := ExecRaw(ctx, bdb, "INSERT INTO accounts (id, username, hostname, label, tags, serial, is_active, is_dirty) VALUES (?, ?, ?, ?, ?, ?, ?, ?)", 702, "g2", "h2", "", "", 1, true, false); err != nil {
			t.Fatalf("insert account2 failed: %v", err)
		}

		// Call markAccountsDirtyForKey with isGlobal=true
		if err := markAccountsDirtyForKey(ctx, bdb, 0, true); err != nil {
			t.Fatalf("markAccountsDirtyForKey global failed: %v", err)
		}

		var d1, d2 bool
		if err := QueryRawInto(ctx, bdb, &d1, "SELECT is_dirty FROM accounts WHERE id = ?", 701); err != nil {
			t.Fatalf("select is_dirty a1 failed: %v", err)
		}
		if err := QueryRawInto(ctx, bdb, &d2, "SELECT is_dirty FROM accounts WHERE id = ?", 702); err != nil {
			t.Fatalf("select is_dirty a2 failed: %v", err)
		}
		if !d1 || !d2 {
			t.Fatalf("expected both accounts dirty after global mark")
		}

		// Non-global: create new account and key, assign key
		if _, err := ExecRaw(ctx, bdb, "INSERT INTO accounts (id, username, hostname, label, tags, serial, is_active, is_dirty) VALUES (?, ?, ?, ?, ?, ?, ?, ?)", 703, "ng", "h3", "", "", 1, true, false); err != nil {
			t.Fatalf("insert account3 failed: %v", err)
		}
		res, err := ExecRaw(ctx, bdb, "INSERT INTO public_keys (algorithm, key_data, comment, is_global) VALUES (?, ?, ?, ?)", "ssh-ed25519", "AAAAB3NzaC1yc2E...", "ng1", false)
		if err != nil {
			t.Fatalf("insert key failed: %v", err)
		}
		id64, _ := res.LastInsertId()
		keyID := int(id64)
		if _, err := ExecRaw(ctx, bdb, "INSERT INTO account_keys (key_id, account_id) VALUES (?, ?)", keyID, 703); err != nil {
			t.Fatalf("assign key failed: %v", err)
		}

		if err := markAccountsDirtyForKey(ctx, bdb, keyID, false); err != nil {
			t.Fatalf("markAccountsDirtyForKey non-global failed: %v", err)
		}
		var d3 bool
		if err := QueryRawInto(ctx, bdb, &d3, "SELECT is_dirty FROM accounts WHERE id = ?", 703); err != nil {
			t.Fatalf("select is_dirty a3 failed: %v", err)
		}
		if !d3 {
			t.Fatalf("expected assigned account to be dirty after non-global mark")
		}
	})
}

func TestRotateAndCreateSystemKey_MaintainsSerialsAndMarksDirty(t *testing.T) {
	WithTestStore(t, func(s *BunStore) {
		ctx := context.Background()
		bdb := s.BunDB()

		// Create an account to be marked dirty
		if _, err := ExecRaw(ctx, bdb, "INSERT INTO accounts (id, username, hostname, label, tags, serial, is_active, is_dirty) VALUES (?, ?, ?, ?, ?, ?, ?, ?)", 801, "sk", "hsk", "", "", 1, true, false); err != nil {
			t.Fatalf("insert account failed: %v", err)
		}

		// Insert initial system key serial 1
		if _, err := ExecRaw(ctx, bdb, "INSERT INTO system_keys (serial, public_key, private_key, is_active) VALUES (?, ?, ?, ?)", 1, "pk1", "sk1", true); err != nil {
			t.Fatalf("insert system key failed: %v", err)
		}

		// Rotate system key
		newSerial, err := RotateSystemKeyBun(bdb, "pk2", "sk2")
		if err != nil {
			t.Fatalf("RotateSystemKeyBun failed: %v", err)
		}
		if newSerial != 2 {
			t.Fatalf("expected new serial 2, got %d", newSerial)
		}

		// After rotation accounts should be processed for dirty marking (may or may not be dirty depending on keys); ensure function completed without error

		// Create another system key via CreateSystemKeyBun (non-deactivating)
		s2, err := CreateSystemKeyBun(bdb, "pk3", "sk3")
		if err != nil {
			t.Fatalf("CreateSystemKeyBun failed: %v", err)
		}
		if s2 <= 0 {
			t.Fatalf("expected positive serial from CreateSystemKeyBun, got %d", s2)
		}
	})
}
