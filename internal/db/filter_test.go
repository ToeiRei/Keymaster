// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package db

import (
	"testing"

	"github.com/toeirei/keymaster/internal/model"
)

func TestFilterAccountsByTokens_Basic(t *testing.T) {
	accounts := []model.Account{
		{ID: 1, Username: "deploy", Hostname: "prod-01.example.com", Label: "Prod Web"},
		{ID: 2, Username: "admin", Hostname: "db-01", Label: "DB Server"},
		{ID: 3, Username: "user", Hostname: "staging-01", Label: "Staging"},
	}

	// Nil/empty tokens -> return original slice
	out := FilterAccountsByTokens(accounts, nil)
	if len(out) != len(accounts) {
		t.Fatalf("expected original slice returned for nil tokens")
	}

	out = FilterAccountsByTokens(accounts, []string{})
	if len(out) != len(accounts) {
		t.Fatalf("expected original slice returned for empty tokens")
	}

	// Match by username
	got := FilterAccountsByTokens(accounts, []string{"deploy"})
	if len(got) != 1 || got[0].ID != 1 {
		t.Fatalf("expected only deploy account, got: %v", got)
	}

	// Match by hostname substring
	got = FilterAccountsByTokens(accounts, []string{"prod-01"})
	if len(got) != 1 || got[0].ID != 1 {
		t.Fatalf("expected only prod-01 account, got: %v", got)
	}

	// Match by label (case-insensitive)
	got = FilterAccountsByTokens(accounts, []string{"prod web"})
	if len(got) != 1 || got[0].ID != 1 {
		t.Fatalf("expected only Prod Web account, got: %v", got)
	}

	// Case-insensitive token
	got = FilterAccountsByTokens(accounts, []string{"ADMIN"})
	if len(got) != 1 || got[0].ID != 2 {
		t.Fatalf("expected admin account for uppercase token, got: %v", got)
	}

	// Multiple tokens (AND semantics): username must match 'admin' and host contain 'db'
	got = FilterAccountsByTokens(accounts, []string{"admin", "db"})
	if len(got) != 1 || got[0].ID != 2 {
		t.Fatalf("expected only admin db account for combined tokens, got: %v", got)
	}

	// Multiple tokens where one does not match -> no results
	got = FilterAccountsByTokens(accounts, []string{"admin", "prod"})
	if len(got) != 0 {
		t.Fatalf("expected no accounts for conflicting tokens, got: %v", got)
	}

	// Tokens with spaces and empty tokens should be ignored
	got = FilterAccountsByTokens(accounts, []string{" ", "user"})
	if len(got) != 1 || got[0].ID != 3 {
		t.Fatalf("expected user account when tokens contain whitespace and 'user', got: %v", got)
	}
}
