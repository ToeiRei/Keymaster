package db

import (
	"testing"
	"time"

	"github.com/toeirei/keymaster/internal/core/model"
)

func TestIntegrationStore_BasicReadWrite(t *testing.T) {
	s, err := New("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Add two accounts
	id1, err := s.AddAccount("alice", "host1.example", "alice", "tag1,tag2")
	if err != nil {
		t.Fatalf("AddAccount alice failed: %v", err)
	}
	id2, err := s.AddAccount("bob", "host2.example", "bob", "tag2")
	if err != nil {
		t.Fatalf("AddAccount bob failed: %v", err)
	}
	if id1 == id2 {
		t.Fatalf("expected distinct account ids, got %d and %d", id1, id2)
	}

	// GetAllAccounts should include both
	all, err := s.GetAllAccounts()
	if err != nil {
		t.Fatalf("GetAllAccounts failed: %v", err)
	}
	if len(all) < 2 {
		t.Fatalf("expected >=2 accounts, got %d", len(all))
	}

	// Search for alice
	res, err := s.SearchAccounts("alice")
	if err != nil {
		t.Fatalf("SearchAccounts failed: %v", err)
	}
	found := false
	for _, a := range res {
		if a.Username == "alice" && a.Hostname == "host1.example" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected to find alice in search results")
	}

	// System key operations
	pub := "ssh-ed25519 AAAA... alice@example"
	priv := "PRIVATEKEYDATA"
	serial, err := s.CreateSystemKey(pub, priv)
	if err != nil {
		t.Fatalf("CreateSystemKey failed: %v", err)
	}
	if serial <= 0 {
		t.Fatalf("invalid serial returned: %d", serial)
	}
	sk, err := s.GetActiveSystemKey()
	if err != nil {
		t.Fatalf("GetActiveSystemKey failed: %v", err)
	}
	if sk == nil || sk.PublicKey != pub {
		t.Fatalf("active system key mismatch: got %+v", sk)
	}

	// Audit logging
	if err := s.LogAction("TEST_ACTION", "integration test"); err != nil {
		t.Fatalf("LogAction failed: %v", err)
	}
	entries, err := s.GetAllAuditLogEntries()
	if err != nil {
		t.Fatalf("GetAllAuditLogEntries failed: %v", err)
	}
	has := false
	for _, e := range entries {
		if e.Action == "TEST_ACTION" && e.Details == "integration test" {
			has = true
			break
		}
	}
	if !has {
		t.Fatalf("expected audit entry TEST_ACTION not found; entries: %+v", entries)
	}

	// Bootstrap session round-trip
	id := "sess-1"
	expires := time.Now().Add(1 * time.Hour)
	if err := s.SaveBootstrapSession(id, "alice", "host1.example", "label", "tag1", "tempPub", expires, "pending"); err != nil {
		t.Fatalf("SaveBootstrapSession failed: %v", err)
	}
	bsess, err := s.GetBootstrapSession(id)
	if err != nil {
		t.Fatalf("GetBootstrapSession failed: %v", err)
	}
	if bsess == nil || bsess.Username != "alice" {
		t.Fatalf("bootstrap session mismatch: %+v", bsess)
	}

	// Cleanup: delete accounts we created
	if err := s.DeleteAccount(id1); err != nil {
		t.Fatalf("DeleteAccount id1 failed: %v", err)
	}
	if err := s.DeleteAccount(id2); err != nil {
		t.Fatalf("DeleteAccount id2 failed: %v", err)
	}

	// Ensure GetAllAccounts still works and returns zero or more
	remaining, err := s.GetAllAccounts()
	if err != nil {
		t.Fatalf("GetAllAccounts after delete failed: %v", err)
	}
	_ = remaining // no strict assertion beyond no error

	// Validate types exported in model for sanity
	var a model.Account
	_ = a
}
