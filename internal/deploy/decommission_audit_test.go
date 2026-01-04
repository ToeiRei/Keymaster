package deploy

import (
	"testing"

	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/model"
)

// TestDecommissionAccount_LogsAuditActions verifies that DecommissionAccount
// writes audit log entries via the package-level AuditWriter when one is
// injected. This ensures audit events like DECOMMISSION_START and
// DECOMMISSION_SUCCESS are emitted.
func TestDecommissionAccount_LogsAuditActions(t *testing.T) {
	if err := db.InitDB("sqlite", ":memory:"); err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}

	// Create a simple account to operate on.
	id, err := db.AddAccount("decom", "example.com", "label", "")
	if err != nil {
		t.Fatalf("AddAccount failed: %v", err)
	}

	fake := &db.FakeAuditWriter{}
	db.SetDefaultAuditWriter(fake)
	defer db.ClearDefaultAuditWriter()

	acc := model.Account{ID: id, Username: "decom", Hostname: "example.com", Label: "label", IsActive: true}

	res := DecommissionAccount(acc, "dummy-system-key", DecommissionOptions{SkipRemoteCleanup: true})

	if res.DatabaseDeleteError != nil {
		t.Fatalf("expected no database delete error, got: %v", res.DatabaseDeleteError)
	}
	if !res.DatabaseDeleteDone {
		t.Fatalf("expected database delete to be marked done")
	}

	if len(fake.Calls) < 2 {
		t.Fatalf("expected at least 2 audit calls (start + final), got %d", len(fake.Calls))
	}

	// First call should be DECOMMISSION_START (or DRYRUN if dryrun path used).
	first := fake.Calls[0][0]
	if first != "DECOMMISSION_START" && first != "DECOMMISSION_DRYRUN" {
		t.Fatalf("unexpected first audit action: %s", first)
	}

	// Last call should be either success or failed depending on flow.
	last := fake.Calls[len(fake.Calls)-1][0]
	if last != "DECOMMISSION_SUCCESS" && last != "DECOMMISSION_FAILED" {
		t.Fatalf("unexpected final audit action: %s", last)
	}
}
