package db

import (
	"testing"
)

// Tests that exercise Bun-backed searcher wrappers and helpers using an
// in-memory sqlite DB initialized via InitDB.
func TestBunAccountKeyAndAuditSearchers(t *testing.T) {
	WithTestStore(t, func(s *SqliteStore) {
		bdb := s.bun

		// Add accounts
		if _, err := AddAccountBun(bdb, "alice", "host1", "lbl", ""); err != nil {
			t.Fatalf("AddAccountBun failed: %v", err)
		}
		if _, err := AddAccountBun(bdb, "bob", "host2", "lbl2", ""); err != nil {
			t.Fatalf("AddAccountBun failed: %v", err)
		}

		// Account searcher via Bun implementation
		as := NewBunAccountSearcher(bdb)
		res, err := as.SearchAccounts("ali")
		if err != nil {
			t.Fatalf("SearchAccounts failed: %v", err)
		}
		if len(res) != 1 || res[0].Username != "alice" {
			t.Fatalf("unexpected SearchAccounts result: %+v", res)
		}

		// Add public keys
		if err := AddPublicKeyBun(bdb, "ssh-rsa", "AAAAB3NzaRSAdata", "alice@key", false); err != nil {
			t.Fatalf("AddPublicKeyBun failed: %v", err)
		}
		if err := AddPublicKeyBun(bdb, "ssh-ed25519", "ED25519data", "service-key", true); err != nil {
			t.Fatalf("AddPublicKeyBun failed: %v", err)
		}

		ks := NewBunKeySearcher(bdb)
		kres, err := ks.SearchPublicKeys("service")
		if err != nil {
			t.Fatalf("SearchPublicKeys failed: %v", err)
		}
		if len(kres) != 1 || kres[0].Comment != "service-key" {
			t.Fatalf("unexpected SearchPublicKeys result: %+v", kres)
		}

		// Audit entries
		if err := LogActionBun(bdb, "TEST_ACTION", "details"); err != nil {
			t.Fatalf("LogActionBun failed: %v", err)
		}
		aus := NewBunAuditSearcher(bdb)
		ares, err := aus.GetAllAuditLogEntries()
		if err != nil {
			t.Fatalf("GetAllAuditLogEntries failed: %v", err)
		}
		if len(ares) == 0 {
			t.Fatalf("expected at least one audit entry, got none")
		}
	})
}
