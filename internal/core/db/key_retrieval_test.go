// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package db

import (
	"context"
	"testing"
	"time"
)

// TestGetGlobalPublicKeysBun_OnlyReturnsGlobalKeys verifies that GetGlobalPublicKeysBun
// returns ONLY keys with is_global=1 and does not return non-global keys.
func TestGetGlobalPublicKeysBun_OnlyReturnsGlobalKeys(t *testing.T) {
	bStore, err := New("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	bdb := bStore.BunDB()

	// Create mix of global and non-global keys
	// Global keys
	err = AddPublicKeyBun(bdb, "ssh-ed25519", "AAAAC3global1", "global-key-1", true, time.Time{})
	if err != nil {
		t.Fatalf("AddPublicKeyBun global1 failed: %v", err)
	}
	err = AddPublicKeyBun(bdb, "ssh-ed25519", "AAAAC3global2", "global-key-2", true, time.Time{})
	if err != nil {
		t.Fatalf("AddPublicKeyBun global2 failed: %v", err)
	}
	err = AddPublicKeyBun(bdb, "ssh-rsa", "AAAAB3global3", "global-key-3", true, time.Time{})
	if err != nil {
		t.Fatalf("AddPublicKeyBun global3 failed: %v", err)
	}

	// Non-global keys
	err = AddPublicKeyBun(bdb, "ssh-ed25519", "AAAAC3regular1", "regular-key-1", false, time.Time{})
	if err != nil {
		t.Fatalf("AddPublicKeyBun regular1 failed: %v", err)
	}
	err = AddPublicKeyBun(bdb, "ssh-ed25519", "AAAAC3regular2", "regular-key-2", false, time.Time{})
	if err != nil {
		t.Fatalf("AddPublicKeyBun regular2 failed: %v", err)
	}

	// Get global keys
	globalKeys, err := GetGlobalPublicKeysBun(bdb)
	if err != nil {
		t.Fatalf("GetGlobalPublicKeysBun failed: %v", err)
	}

	// Verify we got exactly 3 global keys
	if len(globalKeys) != 3 {
		t.Fatalf("expected 3 global keys, got %d", len(globalKeys))
	}

	// Verify all returned keys are actually global
	for _, key := range globalKeys {
		if !key.IsGlobal {
			t.Errorf("GetGlobalPublicKeysBun returned non-global key: ID=%d, Comment=%s, IsGlobal=%v",
				key.ID, key.Comment, key.IsGlobal)
		}
	}

	// Verify the returned keys are the correct ones
	globalComments := make(map[string]bool)
	for _, key := range globalKeys {
		globalComments[key.Comment] = true
	}

	expectedGlobalComments := []string{"global-key-1", "global-key-2", "global-key-3"}
	for _, expected := range expectedGlobalComments {
		if !globalComments[expected] {
			t.Errorf("expected global key '%s' not found in results", expected)
		}
	}

	// Verify no regular keys are in the results
	unexpectedComments := []string{"regular-key-1", "regular-key-2"}
	for _, unexpected := range unexpectedComments {
		if globalComments[unexpected] {
			t.Errorf("non-global key '%s' incorrectly returned by GetGlobalPublicKeysBun", unexpected)
		}
	}
}

// TestGetKeysForAccountBun_OnlyReturnsAccountSpecificKeys verifies that GetKeysForAccountBun
// returns ONLY keys assigned to that specific account via account_keys table.
// This test would have caught the BUN ORM bug where it returned all keys.
func TestGetKeysForAccountBun_OnlyReturnsAccountSpecificKeys(t *testing.T) {
	bStore, err := New("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	bdb := bStore.BunDB()
	ctx := context.Background()

	// Create three accounts
	_, err = ExecRaw(ctx, bdb, "INSERT INTO accounts(username, hostname, label) VALUES(?, ?, ?)", "alice", "server1.example.com", "Alice")
	if err != nil {
		t.Fatalf("insert alice account failed: %v", err)
	}
	_, err = ExecRaw(ctx, bdb, "INSERT INTO accounts(username, hostname, label) VALUES(?, ?, ?)", "bob", "server2.example.com", "Bob")
	if err != nil {
		t.Fatalf("insert bob account failed: %v", err)
	}
	_, err = ExecRaw(ctx, bdb, "INSERT INTO accounts(username, hostname, label) VALUES(?, ?, ?)", "charlie", "server3.example.com", "Charlie")
	if err != nil {
		t.Fatalf("insert charlie account failed: %v", err)
	}

	aliceAccount, _ := GetAccountByIDBun(bdb, 1)
	bobAccount, _ := GetAccountByIDBun(bdb, 2)
	charlieAccount, _ := GetAccountByIDBun(bdb, 3)

	// Create mix of global and non-global keys
	// Global keys (should NOT be returned by GetKeysForAccountBun)
	err = AddPublicKeyBun(bdb, "ssh-ed25519", "AAAAC3global1", "global-key-1", true, time.Time{})
	if err != nil {
		t.Fatalf("AddPublicKeyBun global1 failed: %v", err)
	}
	globalKey1, _ := GetPublicKeyByCommentBun(bdb, "global-key-1")

	err = AddPublicKeyBun(bdb, "ssh-ed25519", "AAAAC3global2", "global-key-2", true, time.Time{})
	if err != nil {
		t.Fatalf("AddPublicKeyBun global2 failed: %v", err)
	}

	// Non-global keys
	err = AddPublicKeyBun(bdb, "ssh-ed25519", "AAAAC3alice1", "alice-key-1", false, time.Time{})
	if err != nil {
		t.Fatalf("AddPublicKeyBun alice1 failed: %v", err)
	}
	aliceKey1, _ := GetPublicKeyByCommentBun(bdb, "alice-key-1")

	err = AddPublicKeyBun(bdb, "ssh-ed25519", "AAAAC3alice2", "alice-key-2", false, time.Time{})
	if err != nil {
		t.Fatalf("AddPublicKeyBun alice2 failed: %v", err)
	}
	aliceKey2, _ := GetPublicKeyByCommentBun(bdb, "alice-key-2")

	err = AddPublicKeyBun(bdb, "ssh-rsa", "AAAAB3bob1", "bob-key-1", false, time.Time{})
	if err != nil {
		t.Fatalf("AddPublicKeyBun bob1 failed: %v", err)
	}
	bobKey1, _ := GetPublicKeyByCommentBun(bdb, "bob-key-1")

	err = AddPublicKeyBun(bdb, "ssh-ed25519", "AAAAC3unassigned", "unassigned-key", false, time.Time{})
	if err != nil {
		t.Fatalf("AddPublicKeyBun unassigned failed: %v", err)
	}

	// Assign keys to accounts via account_keys
	// Alice gets 2 keys
	err = AssignKeyToAccountBun(bdb, aliceKey1.ID, aliceAccount.ID)
	if err != nil {
		t.Fatalf("AssignKeyToAccountBun alice1 failed: %v", err)
	}
	err = AssignKeyToAccountBun(bdb, aliceKey2.ID, aliceAccount.ID)
	if err != nil {
		t.Fatalf("AssignKeyToAccountBun alice2 failed: %v", err)
	}

	// Bob gets 1 key
	err = AssignKeyToAccountBun(bdb, bobKey1.ID, bobAccount.ID)
	if err != nil {
		t.Fatalf("AssignKeyToAccountBun bob1 failed: %v", err)
	}

	// Charlie gets 0 keys (no assignments)

	// TEST 1: Verify Alice gets exactly her 2 assigned keys
	aliceKeys, err := GetKeysForAccountBun(bdb, aliceAccount.ID)
	if err != nil {
		t.Fatalf("GetKeysForAccountBun alice failed: %v", err)
	}
	if len(aliceKeys) != 2 {
		t.Fatalf("expected 2 keys for Alice, got %d", len(aliceKeys))
	}

	aliceKeyIDs := make(map[int]bool)
	for _, key := range aliceKeys {
		aliceKeyIDs[key.ID] = true
		// Verify returned keys are NOT global
		if key.IsGlobal {
			t.Errorf("GetKeysForAccountBun for Alice returned global key: ID=%d, Comment=%s",
				key.ID, key.Comment)
		}
	}

	if !aliceKeyIDs[aliceKey1.ID] || !aliceKeyIDs[aliceKey2.ID] {
		t.Errorf("Alice's assigned keys not found in results. Expected IDs: %d, %d. Got: %v",
			aliceKey1.ID, aliceKey2.ID, aliceKeyIDs)
	}

	// Verify Alice does NOT get Bob's keys
	if aliceKeyIDs[bobKey1.ID] {
		t.Errorf("Alice incorrectly received Bob's key (ID=%d)", bobKey1.ID)
	}

	// Verify Alice does NOT get global keys
	if aliceKeyIDs[globalKey1.ID] {
		t.Errorf("Alice incorrectly received global key (ID=%d)", globalKey1.ID)
	}

	// TEST 2: Verify Bob gets exactly his 1 assigned key
	bobKeys, err := GetKeysForAccountBun(bdb, bobAccount.ID)
	if err != nil {
		t.Fatalf("GetKeysForAccountBun bob failed: %v", err)
	}
	if len(bobKeys) != 1 {
		t.Fatalf("expected 1 key for Bob, got %d", len(bobKeys))
	}
	if bobKeys[0].ID != bobKey1.ID {
		t.Errorf("expected Bob's key ID=%d, got ID=%d", bobKey1.ID, bobKeys[0].ID)
	}
	if bobKeys[0].IsGlobal {
		t.Errorf("Bob's key incorrectly marked as global")
	}

	// TEST 3: Verify Charlie gets 0 keys (no assignments)
	charlieKeys, err := GetKeysForAccountBun(bdb, charlieAccount.ID)
	if err != nil {
		t.Fatalf("GetKeysForAccountBun charlie failed: %v", err)
	}
	if len(charlieKeys) != 0 {
		t.Fatalf("expected 0 keys for Charlie (no assignments), got %d", len(charlieKeys))
	}

	// TEST 4: Verify unassigned key doesn't appear in any account's results
	unassignedKey, _ := GetPublicKeyByCommentBun(bdb, "unassigned-key")

	// Check Alice's keys
	for _, key := range aliceKeys {
		if key.ID == unassignedKey.ID {
			t.Errorf("unassigned key (ID=%d) incorrectly appeared in Alice's results", unassignedKey.ID)
		}
	}

	// Check Bob's keys
	for _, key := range bobKeys {
		if key.ID == unassignedKey.ID {
			t.Errorf("unassigned key (ID=%d) incorrectly appeared in Bob's results", unassignedKey.ID)
		}
	}

	// Check Charlie's keys
	for _, key := range charlieKeys {
		if key.ID == unassignedKey.ID {
			t.Errorf("unassigned key (ID=%d) incorrectly appeared in Charlie's results", unassignedKey.ID)
		}
	}
}

// TestGetKeysForAccountBun_EmptyDatabase verifies behavior with no keys
func TestGetKeysForAccountBun_EmptyDatabase(t *testing.T) {
	bStore, err := New("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	bdb := bStore.BunDB()
	ctx := context.Background()

	// Create account
	_, err = ExecRaw(ctx, bdb, "INSERT INTO accounts(username, hostname, label) VALUES(?, ?, ?)", "user", "host.example.com", "Test")
	if err != nil {
		t.Fatalf("insert account failed: %v", err)
	}
	acc, _ := GetAccountByIDBun(bdb, 1)

	// Get keys for account (should be empty)
	keys, err := GetKeysForAccountBun(bdb, acc.ID)
	if err != nil {
		t.Fatalf("GetKeysForAccountBun failed: %v", err)
	}
	if len(keys) != 0 {
		t.Fatalf("expected 0 keys for empty database, got %d", len(keys))
	}

	// Get global keys (should be empty)
	globalKeys, err := GetGlobalPublicKeysBun(bdb)
	if err != nil {
		t.Fatalf("GetGlobalPublicKeysBun failed: %v", err)
	}
	if len(globalKeys) != 0 {
		t.Fatalf("expected 0 global keys for empty database, got %d", len(globalKeys))
	}
}

// TestKeyRetrieval_RealWorldScenario simulates the exact bug scenario:
// - Multiple global keys exist
// - Multiple accounts exist
// - Each account has specific keys assigned
// - Verify deployment would get correct keys (global + account-specific)
func TestKeyRetrieval_RealWorldScenario(t *testing.T) {
	bStore, err := New("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	bdb := bStore.BunDB()
	ctx := context.Background()

	// Create accounts similar to the bug report scenario
	_, err = ExecRaw(ctx, bdb, "INSERT INTO accounts(username, hostname, label) VALUES(?, ?, ?)", "root", "pve.host", "PVE Root")
	if err != nil {
		t.Fatalf("insert pve account failed: %v", err)
	}
	_, err = ExecRaw(ctx, bdb, "INSERT INTO accounts(username, hostname, label) VALUES(?, ?, ?)", "admin", "web.host", "Web Admin")
	if err != nil {
		t.Fatalf("insert web account failed: %v", err)
	}

	pveAccount, _ := GetAccountByIDBun(bdb, 1)
	webAccount, _ := GetAccountByIDBun(bdb, 2)

	// Create 4 global keys (like JuiceSSH, root@ansible, vbauer@framework, vbauer@stargazer.at)
	globalKeyComments := []string{"JuiceSSH", "root@ansible", "vbauer@framework", "vbauer@stargazer.at"}
	for i, comment := range globalKeyComments {
		err = AddPublicKeyBun(bdb, "ssh-ed25519", "AAAAC3global"+string(rune(i+1)), comment, true, time.Time{})
		if err != nil {
			t.Fatalf("AddPublicKeyBun %s failed: %v", comment, err)
		}
	}

	// Create account-specific keys
	err = AddPublicKeyBun(bdb, "ssh-rsa", "AAAAB3pve", "root@pve", false, time.Time{})
	if err != nil {
		t.Fatalf("AddPublicKeyBun root@pve failed: %v", err)
	}
	pveKey, _ := GetPublicKeyByCommentBun(bdb, "root@pve")

	err = AddPublicKeyBun(bdb, "ssh-ed25519", "AAAAC3web", "admin@web", false, time.Time{})
	if err != nil {
		t.Fatalf("AddPublicKeyBun admin@web failed: %v", err)
	}
	webKey, _ := GetPublicKeyByCommentBun(bdb, "admin@web")

	// Assign account-specific keys
	err = AssignKeyToAccountBun(bdb, pveKey.ID, pveAccount.ID)
	if err != nil {
		t.Fatalf("AssignKeyToAccountBun pve failed: %v", err)
	}
	err = AssignKeyToAccountBun(bdb, webKey.ID, webAccount.ID)
	if err != nil {
		t.Fatalf("AssignKeyToAccountBun web failed: %v", err)
	}

	// Get all keys for PVE account
	globalKeys, err := GetGlobalPublicKeysBun(bdb)
	if err != nil {
		t.Fatalf("GetGlobalPublicKeysBun failed: %v", err)
	}
	pveAccountKeys, err := GetKeysForAccountBun(bdb, pveAccount.ID)
	if err != nil {
		t.Fatalf("GetKeysForAccountBun pve failed: %v", err)
	}

	// VERIFY: Global keys should be 4
	if len(globalKeys) != 4 {
		t.Errorf("expected 4 global keys, got %d", len(globalKeys))
	}

	// VERIFY: PVE account-specific keys should be 1
	if len(pveAccountKeys) != 1 {
		t.Errorf("expected 1 account-specific key for PVE, got %d", len(pveAccountKeys))
	}
	if len(pveAccountKeys) > 0 && pveAccountKeys[0].Comment != "root@pve" {
		t.Errorf("expected PVE account key to be 'root@pve', got '%s'", pveAccountKeys[0].Comment)
	}

	// VERIFY: Total deployment for PVE should be 5 keys (4 global + 1 account-specific)
	totalPVEKeys := len(globalKeys) + len(pveAccountKeys)
	if totalPVEKeys != 5 {
		t.Errorf("expected 5 total keys for PVE deployment (4 global + 1 account), got %d", totalPVEKeys)
	}

	// VERIFY: Web account-specific keys should be 1
	webAccountKeys, err := GetKeysForAccountBun(bdb, webAccount.ID)
	if err != nil {
		t.Fatalf("GetKeysForAccountBun web failed: %v", err)
	}
	if len(webAccountKeys) != 1 {
		t.Errorf("expected 1 account-specific key for Web, got %d", len(webAccountKeys))
	}
	if len(webAccountKeys) > 0 && webAccountKeys[0].Comment != "admin@web" {
		t.Errorf("expected Web account key to be 'admin@web', got '%s'", webAccountKeys[0].Comment)
	}

	// VERIFY: PVE account should NOT get web's key
	for _, key := range pveAccountKeys {
		if key.ID == webKey.ID {
			t.Errorf("PVE account incorrectly received web account's key")
		}
	}

	// VERIFY: Web account should NOT get PVE's key
	for _, key := range webAccountKeys {
		if key.ID == pveKey.ID {
			t.Errorf("Web account incorrectly received PVE account's key")
		}
	}

	// VERIFY: Account-specific keys are NOT marked as global
	allAccountKeys := append(pveAccountKeys, webAccountKeys...)
	for _, key := range allAccountKeys {
		if key.IsGlobal {
			t.Errorf("account-specific key '%s' (ID=%d) incorrectly marked as global",
				key.Comment, key.ID)
		}
	}

	// VERIFY: Global keys are marked as global
	for _, key := range globalKeys {
		if !key.IsGlobal {
			t.Errorf("global key '%s' (ID=%d) not marked as global", key.Comment, key.ID)
		}
	}
}
