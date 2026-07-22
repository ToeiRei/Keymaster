// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package bunrewrite

import (
	"context"
	"log"
	"testing"
	"time"
)

// TestSmokeCRUD exercises the basic CRUD happy path against an in-memory SQLite
// database created by the vendored migrations.
func TestSmokeCRUD(t *testing.T) {
	ctx := context.Background()
	c, err := NewDefaultBunClient(log.Default())
	if err != nil {
		t.Fatalf("NewDefaultBunClient: %v", err)
	}
	defer c.Close(ctx)

	// account
	acc, err := c.CreateAccount(ctx, "root", "example.com", 22, "ssh", "")
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}
	if acc.Host != "example.com" || acc.Port != 22 {
		t.Fatalf("account host/port not persisted: %+v", acc)
	}

	// non-global key with an expiry
	future := time.Now().Add(24 * time.Hour)
	key1, err := c.CreatePublicKey(ctx, "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5key1 alice", "alice", false, future)
	if err != nil {
		t.Fatalf("CreatePublicKey key1: %v", err)
	}
	if key1.IsGlobal || key1.ExpiresAt.IsZero() || key1.Data == "" {
		t.Fatalf("key1 not persisted correctly: %+v", key1)
	}

	// global key with no expiry
	key2, err := c.CreatePublicKey(ctx, "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5key2 bob", "bob", true, time.Time{})
	if err != nil {
		t.Fatalf("CreatePublicKey key2: %v", err)
	}
	if !key2.IsGlobal {
		t.Fatalf("key2 should be global: %+v", key2)
	}

	// link account <-> key1 (direct 1:1)
	link, err := c.CreateLink(ctx, acc.Id, key1.Id, future)
	if err != nil {
		t.Fatalf("CreateLink: %v", err)
	}
	if link.PublicKeyId != key1.Id || link.AccountId != acc.Id {
		t.Fatalf("link wrong: %+v", link)
	}

	// linked keys for account = linked key1 + global key2
	linked, err := c.ListPublicKeysLinkedToAccount(ctx, acc.Id, false)
	if err != nil {
		t.Fatalf("ListPublicKeysLinkedToAccount: %v", err)
	}
	if len(linked) != 2 {
		t.Fatalf("expected 2 linked keys (linked + global), got %d: %+v", len(linked), linked)
	}

	// accounts linked to the global key = all accounts
	accsForGlobal, err := c.ListAccountsLinkedToPublicKey(ctx, key2.Id, false)
	if err != nil {
		t.Fatalf("ListAccountsLinkedToPublicKey(global): %v", err)
	}
	if len(accsForGlobal) != 1 {
		t.Fatalf("expected global key to reach 1 account, got %d", len(accsForGlobal))
	}

	// accounts linked to key1 = the one account
	accsForKey1, err := c.ListAccountsLinkedToPublicKey(ctx, key1.Id, false)
	if err != nil {
		t.Fatalf("ListAccountsLinkedToPublicKey(key1): %v", err)
	}
	if len(accsForKey1) != 1 || accsForKey1[0].Id != acc.Id {
		t.Fatalf("expected key1 linked to acc, got %+v", accsForKey1)
	}

	// link listings
	if ls, err := c.ListLinksForAccount(ctx, acc.Id, false); err != nil || len(ls) != 1 {
		t.Fatalf("ListLinksForAccount: len=%d err=%v", len(ls), err)
	}
	if ls, err := c.ListLinksForPublicKey(ctx, key1.Id, false); err != nil || len(ls) != 1 {
		t.Fatalf("ListLinksForPublicKey: len=%d err=%v", len(ls), err)
	}

	// all keys
	if all, err := c.ListPublicKeys(ctx); err != nil || len(all) != 2 {
		t.Fatalf("ListPublicKeys: len=%d err=%v", len(all), err)
	}

	// an expired global key must NOT count as linked/reaching accounts unless expired=true
	pastGlobal := time.Now().Add(-1 * time.Hour)
	expiredGlobal, err := c.CreatePublicKey(ctx, "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5key3 carol", "carol", true, pastGlobal)
	if err != nil {
		t.Fatalf("CreatePublicKey expiredGlobal: %v", err)
	}
	// account now sees: linked key1 + active global key2 (expired global key3 excluded)
	if linked, err := c.ListPublicKeysLinkedToAccount(ctx, acc.Id, false); err != nil || len(linked) != 2 {
		t.Fatalf("expired global key must be excluded (expired=false): len=%d err=%v", len(linked), err)
	}
	// with expired=true the expired global key is included
	if linked, err := c.ListPublicKeysLinkedToAccount(ctx, acc.Id, true); err != nil || len(linked) != 3 {
		t.Fatalf("expired global key must be included (expired=true): len=%d err=%v", len(linked), err)
	}
	// an expired global key reaches no accounts when expired=false, all when expired=true
	if accs, err := c.ListAccountsLinkedToPublicKey(ctx, expiredGlobal.Id, false); err != nil || len(accs) != 0 {
		t.Fatalf("expired global key should reach 0 accounts (expired=false): len=%d err=%v", len(accs), err)
	}
	if accs, err := c.ListAccountsLinkedToPublicKey(ctx, expiredGlobal.Id, true); err != nil || len(accs) != 1 {
		t.Fatalf("expired global key should reach all accounts (expired=true): len=%d err=%v", len(accs), err)
	}
	if err := c.DeletePublicKeys(ctx, expiredGlobal.Id); err != nil {
		t.Fatalf("cleanup expiredGlobal: %v", err)
	}

	// expiry filter: an expired link is excluded unless expired=true
	past := time.Now().Add(-1 * time.Hour)
	if _, err := c.UpdateLink(ctx, acc.Id, key1.Id, past); err != nil {
		t.Fatalf("UpdateLink: %v", err)
	}
	if ls, err := c.ListLinksForAccount(ctx, acc.Id, false); err != nil || len(ls) != 0 {
		t.Fatalf("expired link should be filtered when expired=false: len=%d err=%v", len(ls), err)
	}
	if ls, err := c.ListLinksForAccount(ctx, acc.Id, true); err != nil || len(ls) != 1 {
		t.Fatalf("expired link should be included when expired=true: len=%d err=%v", len(ls), err)
	}

	// delete link
	if err := c.DeleteLink(ctx, acc.Id, key1.Id); err != nil {
		t.Fatalf("DeleteLink: %v", err)
	}
	if _, err := c.GetLink(ctx, acc.Id, key1.Id); err == nil {
		t.Fatalf("expected GetLink to fail after delete")
	}

	// update a public key (comment + flags)
	updated, err := c.UpdatePublicKey(ctx, key1.Id, "alice2", true, time.Time{})
	if err != nil {
		t.Fatalf("UpdatePublicKey: %v", err)
	}
	if updated.Comment != "alice2" || !updated.IsGlobal || !updated.ExpiresAt.IsZero() {
		t.Fatalf("UpdatePublicKey not applied: %+v", updated)
	}
}
