package db

import (
	"context"
	"testing"
	"time"
)

// TestSearchPublicKeysBun_OnlyReturnMatchingKeys verifies search doesn't return unrelated keys
func TestSearchPublicKeysBun_OnlyReturnMatchingKeys(t *testing.T) {
	bStore, err := New("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	bdb := bStore.BunDB()

	// Create keys with distinct patterns
	keys := []struct {
		algorithm string
		keyData   string
		comment   string
	}{
		{"ssh-ed25519", "AAAAC3test1", "production-key"},
		{"ssh-ed25519", "AAAAC3test2", "staging-key"},
		{"ssh-rsa", "AAAAB3test3", "backup-key"},
		{"ssh-ed25519", "AAAAC3test4", "development-key"},
	}

	for _, k := range keys {
		err = AddPublicKeyBun(bdb, k.algorithm, k.keyData, k.comment, false, time.Time{})
		if err != nil {
			t.Fatalf("AddPublicKeyBun %s failed: %v", k.comment, err)
		}
	}

	tests := []struct {
		name        string
		query       string
		expectCount int
		expectNames []string
	}{
		{"Search for 'prod'", "prod", 1, []string{"production-key"}},
		{"Search for 'key'", "key", 4, []string{"production-key", "staging-key", "backup-key", "development-key"}},
		{"Search for 'rsa'", "rsa", 1, []string{"backup-key"}},
		{"Search for 'staging'", "staging", 1, []string{"staging-key"}},
		{"Search for 'ed25519'", "ed25519", 3, []string{"production-key", "staging-key", "development-key"}},
		{"Search for nonexistent", "xyzabc", 0, []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := SearchPublicKeysBun(bdb, tt.query)
			if err != nil {
				t.Fatalf("SearchPublicKeysBun failed: %v", err)
			}
			if len(results) != tt.expectCount {
				t.Errorf("expected %d results, got %d. Results: %+v", tt.expectCount, len(results), results)
			}

			// Verify all results match expected names
			foundNames := make(map[string]bool)
			for _, r := range results {
				foundNames[r.Comment] = true
			}

			for _, expectedName := range tt.expectNames {
				if !foundNames[expectedName] {
					t.Errorf("expected result '%s' not found. Got: %v", expectedName, foundNames)
				}
			}

			// Verify no unexpected results
			for name := range foundNames {
				found := false
				for _, expectedName := range tt.expectNames {
					if name == expectedName {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("unexpected result '%s' found", name)
				}
			}
		})
	}
}

// TestSearchAccountsBun_OnlyReturnMatchingAccounts verifies search doesn't return unrelated accounts
func TestSearchAccountsBun_OnlyReturnMatchingAccounts(t *testing.T) {
	bStore, err := New("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	bdb := bStore.BunDB()

	// Create accounts with distinct patterns
	accounts := []struct {
		username string
		hostname string
		label    string
	}{
		{"admin", "prod-server.example.com", "Production Admin"},
		{"root", "staging-server.example.com", "Staging Root"},
		{"deploy", "ci-runner.example.com", "CI/CD Deploy"},
		{"backup", "backup-storage.example.com", "Backup Service"},
	}

	for _, acc := range accounts {
		_, err = AddAccountBun(bdb, acc.username, acc.hostname, acc.label, "")
		if err != nil {
			t.Fatalf("AddAccountBun %s@%s failed: %v", acc.username, acc.hostname, err)
		}
	}

	tests := []struct {
		name        string
		query       string
		expectCount int
		expectHosts []string
	}{
		{"Search for 'prod'", "prod", 1, []string{"prod-server.example.com"}},
		{"Search for 'admin'", "admin", 1, []string{"prod-server.example.com"}},
		{"Search for 'server'", "server", 2, []string{"prod-server.example.com", "staging-server.example.com"}},
		{"Search for 'example.com'", "example.com", 4, []string{"prod-server.example.com", "staging-server.example.com", "ci-runner.example.com", "backup-storage.example.com"}},
		{"Search for 'root'", "root", 1, []string{"staging-server.example.com"}},
		{"Search for 'staging'", "staging", 1, []string{"staging-server.example.com"}},
		{"Search for nonexistent", "xyzabc", 0, []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := SearchAccountsBun(bdb, tt.query)
			if err != nil {
				t.Fatalf("SearchAccountsBun failed: %v", err)
			}
			if len(results) != tt.expectCount {
				t.Errorf("expected %d results, got %d", tt.expectCount, len(results))
				for i, r := range results {
					t.Logf("  Result %d: %s@%s (label=%s)", i, r.Username, r.Hostname, r.Label)
				}
			}

			// Verify all results match expected hostnames
			foundHosts := make(map[string]bool)
			for _, r := range results {
				foundHosts[r.Hostname] = true
			}

			for _, expectedHost := range tt.expectHosts {
				if !foundHosts[expectedHost] {
					t.Errorf("expected result with hostname '%s' not found", expectedHost)
				}
			}

			// Verify no unexpected results
			for host := range foundHosts {
				found := false
				for _, expectedHost := range tt.expectHosts {
					if host == expectedHost {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("unexpected result with hostname '%s' found", host)
				}
			}
		})
	}
}

// TestGetAccountsForKeyBun_OnlyReturnAccountsWithKey verifies accounts with key assignment
func TestGetAccountsForKeyBun_OnlyReturnAccountsWithKey(t *testing.T) {
	bStore, err := New("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	bdb := bStore.BunDB()
	ctx := context.Background()

	// Create accounts
	_, err = ExecRaw(ctx, bdb, "INSERT INTO accounts(username, hostname, label) VALUES(?, ?, ?)", "alice", "server1.example.com", "Alice")
	if err != nil {
		t.Fatalf("insert alice failed: %v", err)
	}
	_, err = ExecRaw(ctx, bdb, "INSERT INTO accounts(username, hostname, label) VALUES(?, ?, ?)", "bob", "server2.example.com", "Bob")
	if err != nil {
		t.Fatalf("insert bob failed: %v", err)
	}
	_, err = ExecRaw(ctx, bdb, "INSERT INTO accounts(username, hostname, label) VALUES(?, ?, ?)", "charlie", "server3.example.com", "Charlie")
	if err != nil {
		t.Fatalf("insert charlie failed: %v", err)
	}

	aliceAcc, _ := GetAccountByIDBun(bdb, 1)
	bobAcc, _ := GetAccountByIDBun(bdb, 2)

	// Create keys
	err = AddPublicKeyBun(bdb, "ssh-ed25519", "AAAAC3key1", "key1", false, time.Time{})
	if err != nil {
		t.Fatalf("AddPublicKeyBun key1 failed: %v", err)
	}
	key1, _ := GetPublicKeyByCommentBun(bdb, "key1")

	err = AddPublicKeyBun(bdb, "ssh-ed25519", "AAAAC3key2", "key2", false, time.Time{})
	if err != nil {
		t.Fatalf("AddPublicKeyBun key2 failed: %v", err)
	}
	key2, _ := GetPublicKeyByCommentBun(bdb, "key2")

	// Assign key1 to Alice and Bob, key2 only to Alice
	AssignKeyToAccountBun(bdb, key1.ID, aliceAcc.ID)
	AssignKeyToAccountBun(bdb, key1.ID, bobAcc.ID)
	AssignKeyToAccountBun(bdb, key2.ID, aliceAcc.ID)

	// Test: GetAccountsForKey for key1 should return Alice and Bob only
	accountsForKey1, err := GetAccountsForKeyBun(bdb, key1.ID)
	if err != nil {
		t.Fatalf("GetAccountsForKeyBun(key1) failed: %v", err)
	}
	if len(accountsForKey1) != 2 {
		t.Errorf("expected 2 accounts for key1, got %d", len(accountsForKey1))
	}

	key1AccountNames := make(map[string]bool)
	for _, acc := range accountsForKey1 {
		key1AccountNames[acc.Username] = true
	}

	if !key1AccountNames["alice"] || !key1AccountNames["bob"] {
		t.Errorf("expected Alice and Bob for key1, got: %v", key1AccountNames)
	}
	if key1AccountNames["charlie"] {
		t.Errorf("Charlie should not have key1")
	}

	// Test: GetAccountsForKey for key2 should return only Alice
	accountsForKey2, err := GetAccountsForKeyBun(bdb, key2.ID)
	if err != nil {
		t.Fatalf("GetAccountsForKeyBun(key2) failed: %v", err)
	}
	if len(accountsForKey2) != 1 {
		t.Errorf("expected 1 account for key2, got %d", len(accountsForKey2))
	}
	if accountsForKey2[0].Username != "alice" {
		t.Errorf("expected Alice for key2, got %s", accountsForKey2[0].Username)
	}
}
