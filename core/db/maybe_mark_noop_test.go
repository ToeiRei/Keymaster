// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package db

import (
	"context"
	"database/sql"
	"testing"
)

func TestMaybeMarkAccountDirtyTx_NoopWhenHashUnchanged(t *testing.T) {
	WithTestStore(t, func(s *BunStore) {
		ctx := context.Background()
		bdb := s.BunDB()

		// Insert an account with known id
		if _, err := ExecRaw(ctx, bdb, "INSERT INTO accounts (id, username, hostname, label, tags, serial, is_active, is_dirty) VALUES (?, ?, ?, ?, ?, ?, ?, ?)", 301, "u3", "h3", "", "", 1, true, false); err != nil {
			t.Fatalf("failed inserting account: %v", err)
		}

		// Compute current fingerprint and set it on the account
		h, err := computeAccountKeyHashTx(ctx, bdb, 301)
		if err != nil {
			t.Fatalf("computeAccountKeyHashTx failed: %v", err)
		}
		if _, err := ExecRaw(ctx, bdb, "UPDATE accounts SET key_hash = ?, is_dirty = ? WHERE id = ?", h, false, 301); err != nil {
			t.Fatalf("failed setting key_hash: %v", err)
		}

		// Count audit entries before
		var before int
		if err := QueryRawInto(ctx, bdb, &before, "SELECT COUNT(id) FROM audit_log WHERE action = ?", "ACCOUNT_KEY_HASH_UPDATED"); err != nil {
			t.Fatalf("count before failed: %v", err)
		}

		// Call MaybeMarkAccountDirtyTx which should be a no-op
		if err := MaybeMarkAccountDirtyTx(ctx, bdb, 301); err != nil {
			t.Fatalf("MaybeMarkAccountDirtyTx returned error: %v", err)
		}

		// Verify key_hash unchanged and is_dirty remains false
		var kh sql.NullString
		if err := QueryRawInto(ctx, bdb, &kh, "SELECT key_hash FROM accounts WHERE id = ?", 301); err != nil {
			t.Fatalf("select key_hash failed: %v", err)
		}
		if !kh.Valid || kh.String != h {
			t.Fatalf("expected key_hash to remain %q, got %q", h, kh.String)
		}
		var dirty bool
		if err := QueryRawInto(ctx, bdb, &dirty, "SELECT is_dirty FROM accounts WHERE id = ?", 301); err != nil {
			t.Fatalf("select is_dirty failed: %v", err)
		}
		if dirty {
			t.Fatalf("expected is_dirty to remain false")
		}

		// Count audit entries after: should be unchanged
		var after int
		if err := QueryRawInto(ctx, bdb, &after, "SELECT COUNT(id) FROM audit_log WHERE action = ?", "ACCOUNT_KEY_HASH_UPDATED"); err != nil {
			t.Fatalf("count after failed: %v", err)
		}
		if after != before {
			t.Fatalf("expected no new audit rows, before=%d after=%d", before, after)
		}
	})
}
