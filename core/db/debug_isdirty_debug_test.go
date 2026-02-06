package db

import (
	"context"
	"testing"
	"time"
)

// Debug helper to print key_hash around delete path.
func TestDebugDeleteKeyHash(t *testing.T) {
	WithTestStore(t, func(s *BunStore) {
		bdb := s.bun
		a1, _ := AddAccountBun(bdb, "u1", "h1", "lbl1", "")
		_ = UpdateAccountIsDirtyBun(bdb, a1, false)
		_ = AddPublicKeyBun(bdb, "ed25519", "IIII", "ekd", false, time.Now().Add(0))
		pk, _ := GetPublicKeyByCommentBun(bdb, "ekd")
		if pk == nil {
			t.Fatalf("no pk")
		}
		_ = AssignKeyToAccountBun(bdb, pk.ID, a1)
		_ = UpdateAccountIsDirtyBun(bdb, a1, false)

		var khBefore string
		_ = QueryRawInto(context.Background(), bdb, &khBefore, "SELECT key_hash FROM accounts WHERE id = ?", a1)
		t.Logf("key_hash before delete: %q", khBefore)

		if err := DeletePublicKeyBun(bdb, pk.ID); err != nil {
			t.Fatalf("DeletePublicKeyBun failed: %v", err)
		}

		var khAfter string
		_ = QueryRawInto(context.Background(), bdb, &khAfter, "SELECT key_hash FROM accounts WHERE id = ?", a1)
		t.Logf("key_hash after delete: %q", khAfter)

		a, _ := GetAccountByIDBun(bdb, a1)
		t.Logf("account dirty after delete: %v", a.IsDirty)
	})
}

func TestDebugDeleteKeyHashViaManager(t *testing.T) {
	WithTestStore(t, func(s *BunStore) {
		am := DefaultAccountManager()
		km := DefaultKeyManager()
		if am == nil || km == nil {
			t.Skip("managers not set")
		}

		a1, _ := am.AddAccount("u1", "h1", "lbl", "")
		_ = UpdateAccountIsDirty(a1, false)
		ek, _ := km.AddPublicKeyAndGetModel("ssh-ed25519", "ZZZZ", "m-ek", false, time.Now())
		if ek == nil {
			t.Fatalf("no ek")
		}
		_ = km.AssignKeyToAccount(ek.ID, a1)
		_ = UpdateAccountIsDirty(a1, false)

		var khBefore string
		_ = QueryRawInto(context.Background(), s.bun, &khBefore, "SELECT key_hash FROM accounts WHERE id = ?", a1)
		t.Logf("manager key_hash before delete: %q", khBefore)

		if err := km.DeletePublicKey(ek.ID); err != nil {
			t.Fatalf("km.DeletePublicKey failed: %v", err)
		}

		var khAfter string
		_ = QueryRawInto(context.Background(), s.bun, &khAfter, "SELECT key_hash FROM accounts WHERE id = ?", a1)
		t.Logf("manager key_hash after delete: %q", khAfter)
		a, _ := GetAccountByIDBun(s.bun, a1)
		t.Logf("manager account dirty after delete: %v", a.IsDirty)
	})
}
