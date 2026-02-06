// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package db

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestAssignKeyToAccount_GlobalKey_ReturnsError(t *testing.T) {
	bStore, err := New("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	bdb := bStore.BunDB()
	ctx := context.Background()

	// Create account
	_, err = ExecRaw(ctx, bdb, "INSERT INTO accounts(username, hostname, label) VALUES(?, ?, ?)", "user1", "host1.example.com", "Test1")
	if err != nil {
		t.Fatalf("insert account failed: %v", err)
	}
	acc, _ := GetAccountByIDBun(bdb, 1)
	if acc == nil {
		t.Fatal("account not found after insert")
	}

	// Create a GLOBAL key
	err = AddPublicKeyBun(bdb, "ssh-ed25519", "AAAAC3test", "global-key", true, time.Time{})
	if err != nil {
		t.Fatalf("AddPublicKeyBun failed: %v", err)
	}
	pk, _ := GetPublicKeyByCommentBun(bdb, "global-key")
	if pk == nil {
		t.Fatal("key not found after insert")
	}

	// Try to assign the global key to an account - should fail
	err = AssignKeyToAccountBun(bdb, pk.ID, acc.ID)
	if err == nil {
		t.Fatal("expected error when assigning global key, got nil")
	}
	if !strings.Contains(err.Error(), "cannot assign global key") {
		t.Fatalf("expected error about global key, got: %v", err)
	}

	// Verify the key was NOT added to account_keys
	keys, err := GetKeysForAccountBun(bdb, acc.ID)
	if err != nil {
		t.Fatalf("GetKeysForAccountBun failed: %v", err)
	}
	if len(keys) != 0 {
		t.Fatalf("expected 0 keys in account_keys for account %d, got %d", acc.ID, len(keys))
	}
}

func TestAssignKeyToAccount_NonGlobalKey_Succeeds(t *testing.T) {
	bStore, err := New("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	bdb := bStore.BunDB()
	ctx := context.Background()

	// Create account
	_, err = ExecRaw(ctx, bdb, "INSERT INTO accounts(username, hostname, label) VALUES(?, ?, ?)", "user2", "host2.example.com", "Test2")
	if err != nil {
		t.Fatalf("insert account failed: %v", err)
	}
	acc, _ := GetAccountByIDBun(bdb, 1)
	if acc == nil {
		t.Fatal("account not found after insert")
	}

	// Create a NON-GLOBAL key
	err = AddPublicKeyBun(bdb, "ssh-ed25519", "AAAAC3test2", "regular-key", false, time.Time{})
	if err != nil {
		t.Fatalf("AddPublicKeyBun failed: %v", err)
	}
	pk, _ := GetPublicKeyByCommentBun(bdb, "regular-key")
	if pk == nil {
		t.Fatal("key not found after insert")
	}

	// Assign the non-global key to an account - should succeed
	err = AssignKeyToAccountBun(bdb, pk.ID, acc.ID)
	if err != nil {
		t.Fatalf("expected no error when assigning non-global key, got: %v", err)
	}

	// Verify the key WAS added to account_keys
	keys, err := GetKeysForAccountBun(bdb, acc.ID)
	if err != nil {
		t.Fatalf("GetKeysForAccountBun failed: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected 1 key in account_keys for account %d, got %d", acc.ID, len(keys))
	}
	if keys[0].ID != pk.ID {
		t.Fatalf("expected key ID %d, got %d", pk.ID, keys[0].ID)
	}
}
