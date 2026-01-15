// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package deploy_test

import (
	"testing"

	"github.com/toeirei/keymaster/internal/core"
	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/model"
	"github.com/toeirei/keymaster/internal/security"
	"github.com/toeirei/keymaster/internal/testutil"
)

// TestDecommissionAccount_LogsAuditActions verifies that DecommissionAccount
// writes audit log entries via the DB AuditWriter when one is injected.
func TestDecommissionAccount_LogsAuditActions(t *testing.T) {
	if err := db.InitDB("sqlite", ":memory:"); err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}

	// Create a simple account to operate on.
	mgr := db.DefaultAccountManager()
	if mgr == nil {
		t.Fatalf("no account manager available")
	}
	id, err := mgr.AddAccount("decom", "example.com", "label", "")
	if err != nil {
		t.Fatalf("AddAccount failed: %v", err)
	}

	fake := &testutil.FakeAuditWriter{}
	db.SetDefaultAuditWriter(fake)
	defer db.ClearDefaultAuditWriter()

	acc := model.Account{ID: id, Username: "decom", Hostname: "example.com", Label: "label", IsActive: true}

	// Use core DecommissionAccount facade which orchestrates and delegates to deploy.
	res := core.DecommissionAccount(acc, security.FromString("dummy-system-key"), core.DecommissionOptions{SkipRemoteCleanup: true})

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
