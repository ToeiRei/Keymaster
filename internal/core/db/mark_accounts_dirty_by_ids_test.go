// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package db

import (
	"context"
	"database/sql"
	"testing"
)

func TestMarkAccountsDirtyByIDs_MarksMultipleAccountsAndInsertsAudit(t *testing.T) {
	WithTestStore(t, func(s *BunStore) {
		ctx := context.Background()
		bdb := s.BunDB()

		// Insert two accounts directly so they start with is_dirty = false and no key_hash
		if _, err := ExecRaw(ctx, bdb, "INSERT INTO accounts (id, username, hostname, label, tags, serial, is_active, is_dirty) VALUES (?, ?, ?, ?, ?, ?, ?, ?)", 101, "u1", "h1", "", "", 1, true, false); err != nil {
			t.Fatalf("failed inserting account1: %v", err)
		}
		if _, err := ExecRaw(ctx, bdb, "INSERT INTO accounts (id, username, hostname, label, tags, serial, is_active, is_dirty) VALUES (?, ?, ?, ?, ?, ?, ?, ?)", 102, "u2", "h2", "", "", 1, true, false); err != nil {
			t.Fatalf("failed inserting account2: %v", err)
		}

		// Call markAccountsDirtyByIDs for both
		if err := markAccountsDirtyByIDs(ctx, bdb, []int{101, 102}); err != nil {
			t.Fatalf("markAccountsDirtyByIDs failed: %v", err)
		}

		// Verify both accounts are marked dirty and have non-empty key_hash
		var dirty1 bool
		var kh1 sql.NullString
		if err := QueryRawInto(ctx, bdb, &kh1, "SELECT key_hash FROM accounts WHERE id = ?", 101); err != nil {
			t.Fatalf("select key_hash account1 failed: %v", err)
		}
		if err := QueryRawInto(ctx, bdb, &dirty1, "SELECT is_dirty FROM accounts WHERE id = ?", 101); err != nil {
			t.Fatalf("select is_dirty account1 failed: %v", err)
		}
		if !kh1.Valid || kh1.String == "" {
			t.Fatalf("expected account1 key_hash set, got empty")
		}
		if !dirty1 {
			t.Fatalf("expected account1 is_dirty = true")
		}

		var dirty2 bool
		var kh2 sql.NullString
		if err := QueryRawInto(ctx, bdb, &kh2, "SELECT key_hash FROM accounts WHERE id = ?", 102); err != nil {
			t.Fatalf("select key_hash account2 failed: %v", err)
		}
		if err := QueryRawInto(ctx, bdb, &dirty2, "SELECT is_dirty FROM accounts WHERE id = ?", 102); err != nil {
			t.Fatalf("select is_dirty account2 failed: %v", err)
		}
		if !kh2.Valid || kh2.String == "" {
			t.Fatalf("expected account2 key_hash set, got empty")
		}
		if !dirty2 {
			t.Fatalf("expected account2 is_dirty = true")
		}

		// Verify two audit entries for ACCOUNT_KEY_HASH_UPDATED exist
		var count int
		if err := QueryRawInto(ctx, bdb, &count, "SELECT COUNT(id) FROM audit_log WHERE action = ?", "ACCOUNT_KEY_HASH_UPDATED"); err != nil {
			t.Fatalf("count audit entries failed: %v", err)
		}
		if count < 2 {
			t.Fatalf("expected at least 2 ACCOUNT_KEY_HASH_UPDATED audit rows, got %d", count)
		}
	})
}
