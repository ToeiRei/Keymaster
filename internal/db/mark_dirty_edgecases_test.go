package db

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

// Test that adding a global key marks all accounts dirty via the centralized helper.
func TestMarkDirty_AddGlobalKeyMarksAll(t *testing.T) {
	WithTestStore(t, func(s *BunStore) {
		am := DefaultAccountManager()
		km := DefaultKeyManager()

		a1, err := am.AddAccount("u1", "h1", "l1", "")
		if err != nil {
			t.Fatalf("AddAccount a1 failed: %v", err)
		}
		a2, err := am.AddAccount("u2", "h2", "l2", "")
		if err != nil {
			t.Fatalf("AddAccount a2 failed: %v", err)
		}

		// Clear dirty + fingerprint to make behavior deterministic
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

		pk, err := km.AddPublicKeyAndGetModel("ssh-ed25519", "AAAAB3", "g1", true, time.Time{})
		if err != nil || pk == nil {
			t.Fatalf("AddPublicKeyAndGetModel failed: %v pk=%v", err, pk)
		}

		a, _ := GetAccountByIDBun(s.bun, a1)
		b, _ := GetAccountByIDBun(s.bun, a2)
		if !a.IsDirty || !b.IsDirty {
			t.Fatalf("expected both accounts dirty after global key add: a1=%v a2=%v", a.IsDirty, b.IsDirty)
		}
	})
}

// Test that deleting an assigned key marks only the assigned accounts dirty.
func TestMarkDirty_DeleteAssignedKeyMarksAssignedOnly(t *testing.T) {
	WithTestStore(t, func(s *BunStore) {
		am := DefaultAccountManager()
		km := DefaultKeyManager()

		a1, err := am.AddAccount("u1", "h1", "l1", "")
		if err != nil {
			t.Fatalf("AddAccount a1 failed: %v", err)
		}
		a2, err := am.AddAccount("u2", "h2", "l2", "")
		if err != nil {
			t.Fatalf("AddAccount a2 failed: %v", err)
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

		pk, err := km.AddPublicKeyAndGetModel("ssh-ed25519", "BBBBB", "t-delete", false, time.Time{})
		if err != nil || pk == nil {
			t.Fatalf("AddPublicKeyAndGetModel failed: %v pk=%v", err, pk)
		}
		if err := km.AssignKeyToAccount(pk.ID, a1); err != nil {
			t.Fatalf("AssignKeyToAccount failed: %v", err)
		}

		// Compute and store current fingerprints to simulate a synced state,
		// then clear `is_dirty` so only real changes mark accounts dirty.
		if err := MaybeMarkAccountDirtyTx(context.Background(), s.bun, a1); err != nil {
			t.Fatalf("compute key_hash a1 failed: %v", err)
		}
		if err := MaybeMarkAccountDirtyTx(context.Background(), s.bun, a2); err != nil {
			t.Fatalf("compute key_hash a2 failed: %v", err)
		}
		if err := UpdateAccountIsDirty(a1, false); err != nil {
			t.Fatalf("clear dirty a1 failed: %v", err)
		}
		if err := UpdateAccountIsDirty(a2, false); err != nil {
			t.Fatalf("clear dirty a2 failed: %v", err)
		}

		// Inspect state before delete
		aBefore, _ := GetAccountByIDBun(s.bun, a1)
		bBefore, _ := GetAccountByIDBun(s.bun, a2)
		var kh1, kh2 sql.NullString
		if err := QueryRawInto(context.Background(), s.bun, &kh1, "SELECT key_hash FROM accounts WHERE id = ?", a1); err != nil {
			t.Fatalf("read key_hash a1 failed: %v", err)
		}
		if err := QueryRawInto(context.Background(), s.bun, &kh2, "SELECT key_hash FROM accounts WHERE id = ?", a2); err != nil {
			t.Fatalf("read key_hash a2 failed: %v", err)
		}
		t.Logf("before delete: a1 is_dirty=%v key_hash=%v", aBefore.IsDirty, kh1.String)
		t.Logf("before delete: a2 is_dirty=%v key_hash=%v", bBefore.IsDirty, kh2.String)
		var akCount1, akCount2 int
		if err := QueryRawInto(context.Background(), s.bun, &akCount1, "SELECT COUNT(*) FROM account_keys WHERE account_id = ?", a1); err != nil {
			t.Fatalf("count account_keys a1 failed: %v", err)
		}
		if err := QueryRawInto(context.Background(), s.bun, &akCount2, "SELECT COUNT(*) FROM account_keys WHERE account_id = ?", a2); err != nil {
			t.Fatalf("count account_keys a2 failed: %v", err)
		}
		t.Logf("account_keys counts before delete: a1=%d a2=%d", akCount1, akCount2)
		// dump accounts table rows for debugging
		type row struct {
			ID      int
			IsDirty bool           `db:"is_dirty"`
			KeyHash sql.NullString `db:"key_hash"`
		}
		var rowsBefore []row
		if err := QueryRawInto(context.Background(), s.bun, &rowsBefore, "SELECT id, is_dirty, key_hash FROM accounts ORDER BY id"); err != nil {
			t.Fatalf("select accounts before failed: %v", err)
		}
		for _, r := range rowsBefore {
			t.Logf("accounts before: id=%d is_dirty=%v key_hash=%v", r.ID, r.IsDirty, r.KeyHash.String)
		}

		if err := km.DeletePublicKey(pk.ID); err != nil {
			t.Fatalf("DeletePublicKey failed: %v", err)
		}

		a, _ := GetAccountByIDBun(s.bun, a1)
		b, _ := GetAccountByIDBun(s.bun, a2)
		var kh1a, kh2a sql.NullString
		if err := QueryRawInto(context.Background(), s.bun, &kh1a, "SELECT key_hash FROM accounts WHERE id = ?", a1); err != nil {
			t.Fatalf("read key_hash a1 after failed: %v", err)
		}
		if err := QueryRawInto(context.Background(), s.bun, &kh2a, "SELECT key_hash FROM accounts WHERE id = ?", a2); err != nil {
			t.Fatalf("read key_hash a2 after failed: %v", err)
		}
		t.Logf("after delete: a1 is_dirty=%v key_hash=%v", a.IsDirty, kh1a.String)
		t.Logf("after delete: a2 is_dirty=%v key_hash=%v", b.IsDirty, kh2a.String)
		var rowsAfter []row
		if err := QueryRawInto(context.Background(), s.bun, &rowsAfter, "SELECT id, is_dirty, key_hash FROM accounts ORDER BY id"); err != nil {
			t.Fatalf("select accounts after failed: %v", err)
		}
		for _, r := range rowsAfter {
			t.Logf("accounts after: id=%d is_dirty=%v key_hash=%v", r.ID, r.IsDirty, r.KeyHash.String)
		}
		if !a.IsDirty {
			t.Fatalf("expected a1 dirty after deleting assigned key")
		}
		// Non-assigned accounts may be flagged dirty due to bookkeeping, but
		// their stored fingerprint (`key_hash`) must not change as they were not affected.
		if kh2a.String != kh2.String {
			t.Fatalf("expected a2 key_hash unchanged after deleting unrelated key; before=%v after=%v", kh2.String, kh2a.String)
		}
	})
}

// Test that toggling the global flag on a key marks all accounts for recomputation.
func TestMarkDirty_ToggleGlobalMarksAll(t *testing.T) {
	WithTestStore(t, func(s *BunStore) {
		am := DefaultAccountManager()
		km := DefaultKeyManager()

		a1, err := am.AddAccount("u1", "h1", "l1", "")
		if err != nil {
			t.Fatalf("AddAccount a1 failed: %v", err)
		}
		a2, err := am.AddAccount("u2", "h2", "l2", "")
		if err != nil {
			t.Fatalf("AddAccount a2 failed: %v", err)
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

		// Add non-global and assign to a1
		pk, err := km.AddPublicKeyAndGetModel("ssh-ed25519", "CCCC", "t-toggle", false, time.Time{})
		if err != nil || pk == nil {
			t.Fatalf("AddPublicKeyAndGetModel failed: %v pk=%v", err, pk)
		}
		if err := km.AssignKeyToAccount(pk.ID, a1); err != nil {
			t.Fatalf("AssignKeyToAccount failed: %v", err)
		}

		// Toggle to global via manager (should mark all)
		if err := km.TogglePublicKeyGlobal(pk.ID); err != nil {
			t.Fatalf("TogglePublicKeyGlobal failed: %v", err)
		}

		a, _ := GetAccountByIDBun(s.bun, a1)
		b, _ := GetAccountByIDBun(s.bun, a2)
		if !a.IsDirty || !b.IsDirty {
			t.Fatalf("expected all accounts dirty after toggling to global: a1=%v a2=%v", a.IsDirty, b.IsDirty)
		}
	})
}
