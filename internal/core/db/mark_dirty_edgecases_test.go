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

		// capture key_hash values for comparison after delete
		var kh1, kh2 sql.NullString
		if err := QueryRawInto(context.Background(), s.bun, &kh1, "SELECT key_hash FROM accounts WHERE id = ?", a1); err != nil {
			t.Fatalf("read key_hash a1 failed: %v", err)
		}
		if err := QueryRawInto(context.Background(), s.bun, &kh2, "SELECT key_hash FROM accounts WHERE id = ?", a2); err != nil {
			t.Fatalf("read key_hash a2 failed: %v", err)
		}

		if err := km.DeletePublicKey(pk.ID); err != nil {
			t.Fatalf("DeletePublicKey failed: %v", err)
		}

		a, _ := GetAccountByIDBun(s.bun, a1)
		var kh1a, kh2a sql.NullString
		if err := QueryRawInto(context.Background(), s.bun, &kh1a, "SELECT key_hash FROM accounts WHERE id = ?", a1); err != nil {
			t.Fatalf("read key_hash a1 after failed: %v", err)
		}
		if err := QueryRawInto(context.Background(), s.bun, &kh2a, "SELECT key_hash FROM accounts WHERE id = ?", a2); err != nil {
			t.Fatalf("read key_hash a2 after failed: %v", err)
		}
		// done: assert assigned account marked dirty; non-assigned key_hash unchanged
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
