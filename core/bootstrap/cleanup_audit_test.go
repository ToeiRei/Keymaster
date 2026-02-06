// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package bootstrap

import (
	"testing"
	"time"

	"github.com/toeirei/keymaster/core/db"
	"github.com/toeirei/keymaster/testutil"
)

// TestCleanupOrphanedSession_LogsAudit verifies that cleaning up an orphaned
// bootstrap session emits a BOOTSTRAP_FAILED audit entry.
func TestCleanupOrphanedSession_LogsAudit(t *testing.T) {
	if _, err := db.New("sqlite", ":memory:"); err != nil {
		t.Fatalf("db.New failed: %v", err)
	}

	// Save a bootstrap session record to exercise deletion path
	id := "test-session-1"
	expires := time.Now().Add(1 * time.Hour)
	if err := db.SaveBootstrapSession(id, "bob", "host.example", "", "", "pubkey", expires, "active"); err != nil {
		t.Fatalf("SaveBootstrapSession failed: %v", err)
	}

	fake := &testutil.FakeAuditWriter{}
	db.SetDefaultAuditWriter(fake)
	defer db.ClearDefaultAuditWriter()

	// Retrieve the session model and call the cleanup helper.
	s, err := db.GetBootstrapSession(id)
	if err != nil {
		t.Fatalf("GetBootstrapSession failed: %v", err)
	}
	if s == nil {
		t.Fatalf("expected saved session, got nil")
	}

	if err := cleanupOrphanedSessionModel(s); err != nil {
		t.Fatalf("cleanupOrphanedSessionModel failed: %v", err)
	}

	if len(fake.Calls) == 0 {
		t.Fatalf("expected audit calls, got none")
	}
	if fake.Calls[0][0] != "BOOTSTRAP_FAILED" {
		t.Fatalf("unexpected audit action: %s", fake.Calls[0][0])
	}

	// Ensure session was removed
	s2, _ := db.GetBootstrapSession(id)
	if s2 != nil {
		t.Fatalf("expected session deleted, still present")
	}
}
