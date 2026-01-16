package db

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"
)

// Test that MaybeMarkAccountDirtyTx updates accounts.key_hash, sets is_dirty,
// and inserts an ACCOUNT_KEY_HASH_UPDATED row into audit_log when the hash changes.
func TestMaybeMarkAccountDirtyTx_InsertsAuditAndMarksDirty(t *testing.T) {
	WithTestStore(t, func(s *BunStore) {
		am := DefaultAccountManager()
		km := DefaultKeyManager()

		aid, err := am.AddAccount("u", "h", "lbl", "")
		if err != nil {
			t.Fatalf("AddAccount failed: %v", err)
		}

		// Ensure clean initial state
		if err := UpdateAccountIsDirty(aid, false); err != nil {
			t.Fatalf("clear dirty failed: %v", err)
		}
		if _, err := ExecRaw(context.Background(), s.bun, "UPDATE accounts SET key_hash = NULL WHERE id = ?", aid); err != nil {
			t.Fatalf("clear key_hash failed: %v", err)
		}

		// Compute and store initial fingerprint
		if err := MaybeMarkAccountDirtyTx(context.Background(), s.bun, aid); err != nil {
			t.Fatalf("initial MaybeMarkAccountDirtyTx failed: %v", err)
		}

		// Clear audit_log so we can observe the next insert only
		if _, err := ExecRaw(context.Background(), s.bun, "DELETE FROM audit_log"); err != nil {
			t.Fatalf("clear audit_log failed: %v", err)
		}

		// Clear dirty flag so a real change will set it
		if err := UpdateAccountIsDirty(aid, false); err != nil {
			t.Fatalf("clear dirty failed: %v", err)
		}

		// Add a global key to cause the account fingerprint to change
		pk, err := km.AddPublicKeyAndGetModel("ssh-ed25519", "CHANGEKEY", "g-change", true, time.Time{})
		if err != nil || pk == nil {
			t.Fatalf("AddPublicKeyAndGetModel failed: %v pk=%v", err, pk)
		}

		// Now run MaybeMarkAccountDirtyTx; this should update key_hash, set is_dirty, and insert audit row
		if err := MaybeMarkAccountDirtyTx(context.Background(), s.bun, aid); err != nil {
			t.Fatalf("MaybeMarkAccountDirtyTx failed: %v", err)
		}

		// Verify account is_dirty
		a, err := GetAccountByIDBun(s.bun, aid)
		if err != nil {
			t.Fatalf("GetAccountByIDBun failed: %v", err)
		}
		if a == nil {
			t.Fatalf("account missing after update")
		}
		if !a.IsDirty {
			t.Fatalf("expected account to be marked dirty")
		}

		// Verify key_hash stored and audit row exists and matches
		var kh sql.NullString
		if err := QueryRawInto(context.Background(), s.bun, &kh, "SELECT key_hash FROM accounts WHERE id = ?", aid); err != nil {
			t.Fatalf("read key_hash failed: %v", err)
		}
		if !kh.Valid || strings.TrimSpace(kh.String) == "" {
			t.Fatalf("expected non-empty key_hash stored")
		}

		// Read most recent audit row
		type auditRow struct {
			Username string
			Action   string
			Details  string
		}
		var rows []auditRow
		if err := QueryRawInto(context.Background(), s.bun, &rows, "SELECT username, action, details FROM audit_log WHERE action = 'ACCOUNT_KEY_HASH_UPDATED' ORDER BY id DESC LIMIT 1"); err != nil {
			t.Fatalf("query audit_log failed: %v", err)
		}
		if len(rows) != 1 {
			t.Fatalf("expected 1 audit row, got %d", len(rows))
		}
		r := rows[0]
		if r.Action != "ACCOUNT_KEY_HASH_UPDATED" {
			t.Fatalf("expected action ACCOUNT_KEY_HASH_UPDATED, got %s", r.Action)
		}
		if !strings.Contains(r.Details, "account:") || !strings.Contains(r.Details, "key_hash:") {
			t.Fatalf("unexpected audit details: %s", r.Details)
		}
		// Ensure the stored key_hash matches the one recorded in audit details
		if !strings.Contains(r.Details, kh.String) {
			t.Fatalf("audit details key_hash doesn't match stored key_hash; details=%s stored=%s", r.Details, kh.String)
		}
	})
}
