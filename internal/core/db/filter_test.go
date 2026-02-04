package db

import (
	"testing"

	"github.com/toeirei/keymaster/internal/model"
)

func TestFilterPublicKeysByTokens_Basics(t *testing.T) {
	keys := []model.PublicKey{
		{ID: 1, Algorithm: "ssh-ed25519", KeyData: "AAAAB3NzaC1lZDI1NTE5", Comment: "alice@host"},
		{ID: 2, Algorithm: "ssh-rsa", KeyData: "AAAAB3NzaC1yc2E", Comment: "bob@example"},
		{ID: 3, Algorithm: "ssh-ed25519", KeyData: "XYZDATA", Comment: "service-key"},
	}

	// empty tokens returns original slice
	out := FilterPublicKeysByTokens(keys, nil)
	if len(out) != len(keys) {
		t.Fatalf("expected original slice when tokens nil")
	}

	out = FilterPublicKeysByTokens(keys, []string{})
	if len(out) != len(keys) {
		t.Fatalf("expected original slice when tokens empty")
	}

	// match by comment (case-insensitive)
	res := FilterPublicKeysByTokens(keys, []string{"ALICE"})
	if len(res) != 1 || res[0].ID != 1 {
		t.Fatalf("expected single match for ALICE, got %+v", res)
	}

	// match by algorithm
	res = FilterPublicKeysByTokens(keys, []string{"rsa"})
	if len(res) != 1 || res[0].ID != 2 {
		t.Fatalf("expected single match for rsa, got %+v", res)
	}

	// match by key data substring
	res = FilterPublicKeysByTokens(keys, []string{"XYZ"})
	if len(res) != 1 || res[0].ID != 3 {
		t.Fatalf("expected single match for XYZ, got %+v", res)
	}

	// multiple tokens must all match (AND semantics)
	res = FilterPublicKeysByTokens(keys, []string{"ssh-ed25519", "alice"})
	if len(res) != 1 || res[0].ID != 1 {
		t.Fatalf("expected single match for ed25519+alice, got %+v", res)
	}

	// token that doesn't match yields empty
	res = FilterPublicKeysByTokens(keys, []string{"no-match"})
	if len(res) != 0 {
		t.Fatalf("expected no matches for no-match, got %+v", res)
	}

	// tokens with whitespace are trimmed
	res = FilterPublicKeysByTokens(keys, []string{"  bob@exAMPle  "})
	if len(res) != 1 || res[0].ID != 2 {
		t.Fatalf("expected match for trimmed token, got %+v", res)
	}
}
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
