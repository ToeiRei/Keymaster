// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package db

import (
	"testing"
	"time"
)

// TestKeyManager_GetAllPublicKeys_ReturnsAllKeys verifies that GetAllPublicKeys
// returns all keys regardless of global status or expiry.
func TestKeyManager_GetAllPublicKeys_ReturnsAllKeys(t *testing.T) {
	bStore, err := New("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	bdb := bStore.BunDB()
	km := DefaultKeyManager()

	// Add mix of keys: global, non-global, expired, not expired
	futureDate := time.Now().Add(30 * 24 * time.Hour)
	pastDate := time.Now().Add(-30 * 24 * time.Hour)

	err = AddPublicKeyBun(bdb, "ssh-ed25519", "AAAAC3key1", "global-active", true, time.Time{})
	if err != nil {
		t.Fatalf("AddPublicKeyBun key1 failed: %v", err)
	}

	err = AddPublicKeyBun(bdb, "ssh-ed25519", "AAAAC3key2", "global-future-expiry", true, futureDate)
	if err != nil {
		t.Fatalf("AddPublicKeyBun key2 failed: %v", err)
	}

	err = AddPublicKeyBun(bdb, "ssh-ed25519", "AAAAC3key3", "global-expired", true, pastDate)
	if err != nil {
		t.Fatalf("AddPublicKeyBun key3 failed: %v", err)
	}

	err = AddPublicKeyBun(bdb, "ssh-rsa", "AAAAB3key4", "local-active", false, time.Time{})
	if err != nil {
		t.Fatalf("AddPublicKeyBun key4 failed: %v", err)
	}

	err = AddPublicKeyBun(bdb, "ssh-rsa", "AAAAB3key5", "local-expired", false, pastDate)
	if err != nil {
		t.Fatalf("AddPublicKeyBun key5 failed: %v", err)
	}

	// Get all keys
	allKeys, err := km.GetAllPublicKeys()
	if err != nil {
		t.Fatalf("GetAllPublicKeys failed: %v", err)
	}

	// Should have all 5 keys
	if len(allKeys) != 5 {
		t.Errorf("expected 5 keys, got %d", len(allKeys))
	}

	// Verify we have the expected mix
	var globalCount, localCount, expiredCount, activeCount int
	now := time.Now()
	for _, key := range allKeys {
		if key.IsGlobal {
			globalCount++
		} else {
			localCount++
		}
		if !key.ExpiresAt.IsZero() && key.ExpiresAt.Before(now) {
			expiredCount++
		} else {
			activeCount++
		}
	}

	if globalCount != 3 {
		t.Errorf("expected 3 global keys, got %d", globalCount)
	}
	if localCount != 2 {
		t.Errorf("expected 2 local keys, got %d", localCount)
	}
	if expiredCount != 2 {
		t.Errorf("expected 2 expired keys, got %d", expiredCount)
	}
	if activeCount != 3 {
		t.Errorf("expected 3 active keys, got %d", activeCount)
	}
}

// TestKeyManager_TogglePublicKeyGlobal_ChangesStatus verifies that toggling
// a key's global status actually changes it correctly.
func TestKeyManager_TogglePublicKeyGlobal_ChangesStatus(t *testing.T) {
	bStore, err := New("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	bdb := bStore.BunDB()
	km := DefaultKeyManager()

	// Add a non-global key
	addedKey, err := AddPublicKeyAndGetModelBun(bdb, "ssh-ed25519", "AAAAC3testkey", "test@example.com", false, time.Time{})
	if err != nil {
		t.Fatalf("AddPublicKeyAndGetModelBun failed: %v", err)
	}

	if addedKey.IsGlobal {
		t.Fatalf("key should start as non-global, but IsGlobal=%v", addedKey.IsGlobal)
	}

	// Toggle to global
	err = km.TogglePublicKeyGlobal(addedKey.ID)
	if err != nil {
		t.Fatalf("TogglePublicKeyGlobal (to global) failed: %v", err)
	}

	// Verify it's now global
	keys, err := km.GetAllPublicKeys()
	if err != nil {
		t.Fatalf("GetAllPublicKeys failed: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}
	if !keys[0].IsGlobal {
		t.Errorf("after toggle, key should be global, but IsGlobal=%v", keys[0].IsGlobal)
	}

	// Toggle back to non-global
	err = km.TogglePublicKeyGlobal(addedKey.ID)
	if err != nil {
		t.Fatalf("TogglePublicKeyGlobal (to non-global) failed: %v", err)
	}

	// Verify it's now non-global again
	keys, err = km.GetAllPublicKeys()
	if err != nil {
		t.Fatalf("GetAllPublicKeys failed: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}
	if keys[0].IsGlobal {
		t.Errorf("after second toggle, key should be non-global, but IsGlobal=%v", keys[0].IsGlobal)
	}
}

// TestKeyManager_SetPublicKeyExpiry_UpdatesExpiry verifies that setting
// expiry dates works correctly, including clearing expiry.
func TestKeyManager_SetPublicKeyExpiry_UpdatesExpiry(t *testing.T) {
	bStore, err := New("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	bdb := bStore.BunDB()
	km := DefaultKeyManager()

	// Add a key with no expiry
	addedKey, err := AddPublicKeyAndGetModelBun(bdb, "ssh-ed25519", "AAAAC3testkey", "test@example.com", false, time.Time{})
	if err != nil {
		t.Fatalf("AddPublicKeyAndGetModelBun failed: %v", err)
	}

	if !addedKey.ExpiresAt.IsZero() {
		t.Fatalf("key should start with no expiry, but ExpiresAt=%v", addedKey.ExpiresAt)
	}

	// Set expiry to future date
	futureDate := time.Date(2027, 12, 31, 23, 59, 59, 0, time.UTC)
	err = km.SetPublicKeyExpiry(addedKey.ID, futureDate)
	if err != nil {
		t.Fatalf("SetPublicKeyExpiry failed: %v", err)
	}

	// Verify expiry is set
	keys, err := km.GetAllPublicKeys()
	if err != nil {
		t.Fatalf("GetAllPublicKeys failed: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}
	if keys[0].ExpiresAt.IsZero() {
		t.Errorf("after setting expiry, ExpiresAt should not be zero")
	}
	// Allow for some timezone/precision differences
	if keys[0].ExpiresAt.Year() != 2027 || keys[0].ExpiresAt.Month() != 12 || keys[0].ExpiresAt.Day() != 31 {
		t.Errorf("expected expiry date 2027-12-31, got %v", keys[0].ExpiresAt)
	}

	// Clear expiry (set to zero time)
	err = km.SetPublicKeyExpiry(addedKey.ID, time.Time{})
	if err != nil {
		t.Fatalf("SetPublicKeyExpiry (clear) failed: %v", err)
	}

	// Verify expiry is cleared
	keys, err = km.GetAllPublicKeys()
	if err != nil {
		t.Fatalf("GetAllPublicKeys failed: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}
	if !keys[0].ExpiresAt.IsZero() {
		t.Errorf("after clearing expiry, ExpiresAt should be zero, but got %v", keys[0].ExpiresAt)
	}
}

// TestKeyManager_GetGlobalPublicKeys_OnlyReturnsGlobal verifies that GetGlobalPublicKeys
// returns only keys with IsGlobal=true and filters out non-global keys.
func TestKeyManager_GetGlobalPublicKeys_OnlyReturnsGlobal(t *testing.T) {
	bStore, err := New("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	bdb := bStore.BunDB()
	km := DefaultKeyManager()

	// Add mix of global and non-global keys
	err = AddPublicKeyBun(bdb, "ssh-ed25519", "AAAAC3global1", "global-1", true, time.Time{})
	if err != nil {
		t.Fatalf("AddPublicKeyBun global1 failed: %v", err)
	}

	err = AddPublicKeyBun(bdb, "ssh-ed25519", "AAAAC3global2", "global-2", true, time.Time{})
	if err != nil {
		t.Fatalf("AddPublicKeyBun global2 failed: %v", err)
	}

	err = AddPublicKeyBun(bdb, "ssh-ed25519", "AAAAC3local1", "local-1", false, time.Time{})
	if err != nil {
		t.Fatalf("AddPublicKeyBun local1 failed: %v", err)
	}

	err = AddPublicKeyBun(bdb, "ssh-rsa", "AAAAB3local2", "local-2", false, time.Time{})
	if err != nil {
		t.Fatalf("AddPublicKeyBun local2 failed: %v", err)
	}

	// Get only global keys
	globalKeys, err := km.GetGlobalPublicKeys()
	if err != nil {
		t.Fatalf("GetGlobalPublicKeys failed: %v", err)
	}

	// Should have exactly 2 global keys
	if len(globalKeys) != 2 {
		t.Errorf("expected 2 global keys, got %d", len(globalKeys))
	}

	// Verify all returned keys are global
	for _, key := range globalKeys {
		if !key.IsGlobal {
			t.Errorf("GetGlobalPublicKeys returned non-global key: ID=%d, Comment=%s", key.ID, key.Comment)
		}
	}

	// Verify the comments are correct
	comments := make(map[string]bool)
	for _, key := range globalKeys {
		comments[key.Comment] = true
	}
	if !comments["global-1"] {
		t.Errorf("expected global-1 in results")
	}
	if !comments["global-2"] {
		t.Errorf("expected global-2 in results")
	}
	if comments["local-1"] || comments["local-2"] {
		t.Errorf("non-global keys should not be in GetGlobalPublicKeys results")
	}
}

// TestKeyManager_AssignAndUnassign_CorrectAccounts verifies that key assignment
// and unassignment work correctly and GetKeysForAccount returns the right keys.
func TestKeyManager_AssignAndUnassign_CorrectAccounts(t *testing.T) {
	bStore, err := New("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	bdb := bStore.BunDB()
	km := DefaultKeyManager()
	am := DefaultAccountManager()

	// Create two accounts
	acc1ID, err := am.AddAccount("user1", "host1.example.com", "Account 1", "tag1")
	if err != nil {
		t.Fatalf("AddAccount acc1 failed: %v", err)
	}
	acc2ID, err := am.AddAccount("user2", "host2.example.com", "Account 2", "tag2")
	if err != nil {
		t.Fatalf("AddAccount acc2 failed: %v", err)
	}

	// Create three keys (all non-global for this test)
	key1, err := AddPublicKeyAndGetModelBun(bdb, "ssh-ed25519", "AAAAC3key1", "key-1", false, time.Time{})
	if err != nil {
		t.Fatalf("AddPublicKeyAndGetModelBun key1 failed: %v", err)
	}
	key2, err := AddPublicKeyAndGetModelBun(bdb, "ssh-ed25519", "AAAAC3key2", "key-2", false, time.Time{})
	if err != nil {
		t.Fatalf("AddPublicKeyAndGetModelBun key2 failed: %v", err)
	}
	key3, err := AddPublicKeyAndGetModelBun(bdb, "ssh-rsa", "AAAAB3key3", "key-3", false, time.Time{})
	if err != nil {
		t.Fatalf("AddPublicKeyAndGetModelBun key3 failed: %v", err)
	}

	// Assign key1 and key2 to acc1
	err = km.AssignKeyToAccount(key1.ID, acc1ID)
	if err != nil {
		t.Fatalf("AssignKeyToAccount key1->acc1 failed: %v", err)
	}
	err = km.AssignKeyToAccount(key2.ID, acc1ID)
	if err != nil {
		t.Fatalf("AssignKeyToAccount key2->acc1 failed: %v", err)
	}

	// Assign key2 and key3 to acc2
	err = km.AssignKeyToAccount(key2.ID, acc2ID)
	if err != nil {
		t.Fatalf("AssignKeyToAccount key2->acc2 failed: %v", err)
	}
	err = km.AssignKeyToAccount(key3.ID, acc2ID)
	if err != nil {
		t.Fatalf("AssignKeyToAccount key3->acc2 failed: %v", err)
	}

	// Get keys for acc1 - should have key1 and key2
	keys1, err := km.GetKeysForAccount(acc1ID)
	if err != nil {
		t.Fatalf("GetKeysForAccount acc1 failed: %v", err)
	}
	if len(keys1) != 2 {
		t.Errorf("expected 2 keys for acc1, got %d", len(keys1))
	}
	comments1 := make(map[string]bool)
	for _, key := range keys1 {
		comments1[key.Comment] = true
	}
	if !comments1["key-1"] || !comments1["key-2"] {
		t.Errorf("acc1 should have key-1 and key-2, got: %v", comments1)
	}
	if comments1["key-3"] {
		t.Errorf("acc1 should not have key-3")
	}

	// Get keys for acc2 - should have key2 and key3
	keys2, err := km.GetKeysForAccount(acc2ID)
	if err != nil {
		t.Fatalf("GetKeysForAccount acc2 failed: %v", err)
	}
	if len(keys2) != 2 {
		t.Errorf("expected 2 keys for acc2, got %d", len(keys2))
	}
	comments2 := make(map[string]bool)
	for _, key := range keys2 {
		comments2[key.Comment] = true
	}
	if !comments2["key-2"] || !comments2["key-3"] {
		t.Errorf("acc2 should have key-2 and key-3, got: %v", comments2)
	}
	if comments2["key-1"] {
		t.Errorf("acc2 should not have key-1")
	}

	// Unassign key2 from acc1
	err = km.UnassignKeyFromAccount(key2.ID, acc1ID)
	if err != nil {
		t.Fatalf("UnassignKeyFromAccount key2->acc1 failed: %v", err)
	}

	// Get keys for acc1 again - should now only have key1
	keys1, err = km.GetKeysForAccount(acc1ID)
	if err != nil {
		t.Fatalf("GetKeysForAccount acc1 (after unassign) failed: %v", err)
	}
	if len(keys1) != 1 {
		t.Errorf("expected 1 key for acc1 after unassign, got %d", len(keys1))
	}
	if keys1[0].Comment != "key-1" {
		t.Errorf("acc1 should only have key-1, got: %s", keys1[0].Comment)
	}

	// Get keys for acc2 - should still have key2 and key3
	keys2, err = km.GetKeysForAccount(acc2ID)
	if err != nil {
		t.Fatalf("GetKeysForAccount acc2 (after unassign) failed: %v", err)
	}
	if len(keys2) != 2 {
		t.Errorf("expected 2 keys for acc2 after unassign from acc1, got %d", len(keys2))
	}
}

// TestKeyManager_GetAccountsForKey_ReturnsCorrectAccounts verifies that
// GetAccountsForKey returns only accounts that have the key assigned.
func TestKeyManager_GetAccountsForKey_ReturnsCorrectAccounts(t *testing.T) {
	bStore, err := New("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	bdb := bStore.BunDB()
	km := DefaultKeyManager()
	am := DefaultAccountManager()

	// Create three accounts
	acc1ID, err := am.AddAccount("user1", "host1.example.com", "Account 1", "")
	if err != nil {
		t.Fatalf("AddAccount acc1 failed: %v", err)
	}
	_, err = am.AddAccount("user2", "host2.example.com", "Account 2", "")
	if err != nil {
		t.Fatalf("AddAccount acc2 failed: %v", err)
	}
	acc3ID, err := am.AddAccount("user3", "host3.example.com", "Account 3", "")
	if err != nil {
		t.Fatalf("AddAccount acc3 failed: %v", err)
	}

	// Create a key
	key1, err := AddPublicKeyAndGetModelBun(bdb, "ssh-ed25519", "AAAAC3testkey", "shared-key", false, time.Time{})
	if err != nil {
		t.Fatalf("AddPublicKeyAndGetModelBun failed: %v", err)
	}

	// Assign key to acc1 and acc3 (not acc2)
	err = km.AssignKeyToAccount(key1.ID, acc1ID)
	if err != nil {
		t.Fatalf("AssignKeyToAccount key->acc1 failed: %v", err)
	}
	err = km.AssignKeyToAccount(key1.ID, acc3ID)
	if err != nil {
		t.Fatalf("AssignKeyToAccount key->acc3 failed: %v", err)
	}

	// Get accounts for key1 - should have acc1 and acc3
	accounts, err := km.GetAccountsForKey(key1.ID)
	if err != nil {
		t.Fatalf("GetAccountsForKey failed: %v", err)
	}
	if len(accounts) != 2 {
		t.Errorf("expected 2 accounts for key, got %d", len(accounts))
	}

	// Verify we got the right accounts
	usernames := make(map[string]bool)
	for _, acc := range accounts {
		usernames[acc.Username] = true
	}
	if !usernames["user1"] || !usernames["user3"] {
		t.Errorf("expected accounts user1 and user3, got: %v", usernames)
	}
	if usernames["user2"] {
		t.Errorf("user2 should not be returned for this key")
	}
}

// TestKeyManager_DeletePublicKey_RemovesKeyAndAssignments verifies that
// deleting a key removes it from all accounts.
func TestKeyManager_DeletePublicKey_RemovesKeyAndAssignments(t *testing.T) {
	bStore, err := New("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	bdb := bStore.BunDB()
	km := DefaultKeyManager()
	am := DefaultAccountManager()

	// Create an account
	acc1ID, err := am.AddAccount("user1", "host1.example.com", "Account 1", "")
	if err != nil {
		t.Fatalf("AddAccount failed: %v", err)
	}

	// Create two keys
	key1, err := AddPublicKeyAndGetModelBun(bdb, "ssh-ed25519", "AAAAC3key1", "key-1", false, time.Time{})
	if err != nil {
		t.Fatalf("AddPublicKeyAndGetModelBun key1 failed: %v", err)
	}
	key2, err := AddPublicKeyAndGetModelBun(bdb, "ssh-ed25519", "AAAAC3key2", "key-2", false, time.Time{})
	if err != nil {
		t.Fatalf("AddPublicKeyAndGetModelBun key2 failed: %v", err)
	}

	// Assign both keys to account
	err = km.AssignKeyToAccount(key1.ID, acc1ID)
	if err != nil {
		t.Fatalf("AssignKeyToAccount key1 failed: %v", err)
	}
	err = km.AssignKeyToAccount(key2.ID, acc1ID)
	if err != nil {
		t.Fatalf("AssignKeyToAccount key2 failed: %v", err)
	}

	// Verify account has both keys
	keys, err := km.GetKeysForAccount(acc1ID)
	if err != nil {
		t.Fatalf("GetKeysForAccount failed: %v", err)
	}
	if len(keys) != 2 {
		t.Errorf("expected 2 keys before delete, got %d", len(keys))
	}

	// Delete key1
	err = km.DeletePublicKey(key1.ID)
	if err != nil {
		t.Fatalf("DeletePublicKey failed: %v", err)
	}

	// Verify key1 no longer exists
	allKeys, err := km.GetAllPublicKeys()
	if err != nil {
		t.Fatalf("GetAllPublicKeys failed: %v", err)
	}
	if len(allKeys) != 1 {
		t.Errorf("expected 1 key after delete, got %d", len(allKeys))
	}
	if allKeys[0].Comment != "key-2" {
		t.Errorf("expected key-2 to remain, got: %s", allKeys[0].Comment)
	}

	// Verify account now only has key2
	keys, err = km.GetKeysForAccount(acc1ID)
	if err != nil {
		t.Fatalf("GetKeysForAccount after delete failed: %v", err)
	}
	if len(keys) != 1 {
		t.Errorf("expected 1 key for account after delete, got %d", len(keys))
	}
	if keys[0].Comment != "key-2" {
		t.Errorf("account should only have key-2, got: %s", keys[0].Comment)
	}
}
