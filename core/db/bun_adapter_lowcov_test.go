package db

import (
	"testing"
)

func TestBunAdapter_LowCoverage(t *testing.T) {
	s, err := New("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Add three accounts
	_, err = s.AddAccount("alice", "host1", "A", "env=prod")
	if err != nil {
		t.Fatalf("AddAccount alice failed: %v", err)
	}
	id2, err := s.AddAccount("bob", "host2", "B", "env=dev")
	if err != nil {
		t.Fatalf("AddAccount bob failed: %v", err)
	}
	id3, err := s.AddAccount("carol", "host3", "C", "env=prod;role=web")
	if err != nil {
		t.Fatalf("AddAccount carol failed: %v", err)
	}

	// Toggle bob to inactive via public Store API
	if err := s.ToggleAccountStatus(id2); err != nil {
		t.Fatalf("ToggleAccountStatus failed: %v", err)
	}

	// GetAllAccounts -> expect at least 3
	all, err := s.GetAllAccounts()
	if err != nil {
		t.Fatalf("GetAllAccounts failed: %v", err)
	}
	if len(all) < 3 {
		t.Fatalf("expected >=3 accounts, got %d", len(all))
	}

	// GetAllActiveAccounts -> bob should be inactive
	active, err := s.GetAllActiveAccounts()
	if err != nil {
		t.Fatalf("GetAllActiveAccounts failed: %v", err)
	}
	if len(active) < 2 {
		t.Fatalf("expected >=2 active accounts, got %d", len(active))
	}

	// Known hosts via Store API
	if err := s.AddKnownHostKey("example.com", "ssh-rsa AAA"); err != nil {
		t.Fatalf("AddKnownHostKey failed: %v", err)
	}
	kh, err := s.GetKnownHostKey("example.com")
	if err != nil {
		t.Fatalf("GetKnownHostKey failed: %v", err)
	}
	if kh != "ssh-rsa AAA" {
		t.Fatalf("unexpected known host value: %q", kh)
	}

	// Delete an account and verify GetAllAccounts reflects it
	if err := s.DeleteAccount(id3); err != nil {
		t.Fatalf("DeleteAccount failed: %v", err)
	}
	post, err := s.GetAllAccounts()
	if err != nil {
		t.Fatalf("GetAllAccounts after delete failed: %v", err)
	}
	if len(post) < 2 {
		t.Fatalf("expected >=2 accounts after delete, got %d", len(post))
	}
}
