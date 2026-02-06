// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package db

import (
	"context"
	"testing"
)

// TestGetAccountsForKeyBun verifies that GetAccountsForKeyBun correctly returns
// only the accounts with explicit assignments to a key, not all accounts.
func TestGetAccountsForKeyBun_SelectiveAssignment(t *testing.T) {
	WithTestStore(t, func(s *BunStore) {
		ctx := context.Background()
		bdb := s.BunDB()

		// Create 4 accounts
		var aids []int
		for i := 1; i <= 4; i++ {
			aid, err := AddAccountBun(bdb, "user"+string(rune('0'+i)), "host"+string(rune('0'+i)), "label"+string(rune('0'+i)), "")
			if err != nil {
				t.Fatalf("AddAccountBun failed: %v", err)
			}
			aids = append(aids, aid)
		}

		// Create 1 key
		res, err := ExecRaw(ctx, bdb, "INSERT INTO public_keys (algorithm, key_data, comment, is_global) VALUES (?, ?, ?, ?)",
			"ssh-ed25519", "AAAAB3...", "test-key", false)
		if err != nil {
			t.Fatalf("insert key failed: %v", err)
		}
		keyID64, _ := res.LastInsertId()
		keyID := int(keyID64)

		// Assign the key to only 2 of the 4 accounts
		if _, err := ExecRaw(ctx, bdb, "INSERT INTO account_keys(key_id, account_id) VALUES(?, ?)", keyID, aids[0]); err != nil {
			t.Fatalf("insert account_key 1 failed: %v", err)
		}
		if _, err := ExecRaw(ctx, bdb, "INSERT INTO account_keys(key_id, account_id) VALUES(?, ?)", keyID, aids[2]); err != nil {
			t.Fatalf("insert account_key 2 failed: %v", err)
		}

		// Query: should return only 2 accounts (the ones assigned)
		accounts, err := GetAccountsForKeyBun(bdb, keyID)
		if err != nil {
			t.Fatalf("GetAccountsForKeyBun failed: %v", err)
		}

		// Verify: should have exactly 2 accounts
		if len(accounts) != 2 {
			t.Errorf("expected 2 accounts assigned to key, got %d", len(accounts))
			t.Logf("accounts returned: %+v", accounts)
		}

		// Verify: should be the correct accounts
		if len(accounts) >= 1 && accounts[0].ID != aids[0] {
			t.Errorf("first account should be %d, got %d", aids[0], accounts[0].ID)
		}
		if len(accounts) >= 2 && accounts[1].ID != aids[2] {
			t.Errorf("second account should be %d, got %d", aids[2], accounts[1].ID)
		}

		// Verify: should NOT include the unassigned accounts
		for _, acc := range accounts {
			if acc.ID == aids[1] || acc.ID == aids[3] {
				t.Errorf("unassigned account %d should not be in results", acc.ID)
			}
		}
	})
}

// TestGetAccountsForKeyBun_NoAssignments verifies that a key with no assignments
// returns an empty slice.
func TestGetAccountsForKeyBun_NoAssignments(t *testing.T) {
	WithTestStore(t, func(s *BunStore) {
		ctx := context.Background()
		bdb := s.BunDB()

		// Create account(s) - should not matter for this test
		AddAccountBun(bdb, "user1", "host1", "label1", "")

		// Create a key with NO assignments
		res, err := ExecRaw(ctx, bdb, "INSERT INTO public_keys (algorithm, key_data, comment, is_global) VALUES (?, ?, ?, ?)",
			"ssh-rsa", "AAAAB3...", "lonely-key", false)
		if err != nil {
			t.Fatalf("insert key failed: %v", err)
		}
		keyID64, _ := res.LastInsertId()
		keyID := int(keyID64)

		// Query: should return empty slice
		accounts, err := GetAccountsForKeyBun(bdb, keyID)
		if err != nil {
			t.Fatalf("GetAccountsForKeyBun failed: %v", err)
		}

		// Verify: should have 0 accounts
		if len(accounts) != 0 {
			t.Errorf("expected 0 accounts for unassigned key, got %d: %+v", len(accounts), accounts)
		}
	})
}

// TestGetAccountsForKeyBun_DistinctDeduplication verifies that DISTINCT works
// correctly and accounts aren't duplicated even if there are multiple joins.
func TestGetAccountsForKeyBun_NoDuplicates(t *testing.T) {
	WithTestStore(t, func(s *BunStore) {
		ctx := context.Background()
		bdb := s.BunDB()

		// Create accounts
		aid1, _ := AddAccountBun(bdb, "user1", "host1", "label1", "")
		aid2, _ := AddAccountBun(bdb, "user2", "host2", "label2", "")

		// Create key
		res, _ := ExecRaw(ctx, bdb, "INSERT INTO public_keys (algorithm, key_data, comment, is_global) VALUES (?, ?, ?, ?)",
			"ssh-ed25519", "AAAAB3...", "test-key", false)
		keyID64, _ := res.LastInsertId()
		keyID := int(keyID64)

		// Assign to both accounts
		ExecRaw(ctx, bdb, "INSERT INTO account_keys(key_id, account_id) VALUES(?, ?)", keyID, aid1)
		ExecRaw(ctx, bdb, "INSERT INTO account_keys(key_id, account_id) VALUES(?, ?)", keyID, aid2)

		// Query
		accounts, err := GetAccountsForKeyBun(bdb, keyID)
		if err != nil {
			t.Fatalf("GetAccountsForKeyBun failed: %v", err)
		}

		// Should have exactly 2 accounts, no duplicates
		if len(accounts) != 2 {
			t.Errorf("expected 2 accounts, got %d: %+v", len(accounts), accounts)
		}

		// Check for duplicates by ID
		idMap := make(map[int]int)
		for _, acc := range accounts {
			idMap[acc.ID]++
		}
		for id, count := range idMap {
			if count > 1 {
				t.Errorf("account %d appears %d times (duplicated)", id, count)
			}
		}
	})
}
