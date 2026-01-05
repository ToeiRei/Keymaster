// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package tui

import (
	"strings"
	"testing"

	"github.com/toeirei/keymaster/internal/i18n"
	"github.com/toeirei/keymaster/internal/model"
)

func TestAuditActionStyleAndRebuild(t *testing.T) {
	// Check styles render something non-empty
	s := auditActionStyle("DELETE_ACCOUNT_1")
	if s.Render("x") == "" {
		t.Fatalf("expected non-empty render from high-risk style")
	}
	s2 := auditActionStyle("ADD_ACCOUNT")
	if s2.Render("x") == "" {
		t.Fatalf("expected non-empty render from low-risk style")
	}

	// Test rebuildTableRows with entries
	m := &auditLogModel{
		allEntries: []model.AuditLogEntry{
			{Timestamp: "2025-01-01T00:00:00Z", Username: "alice", Action: "ADD_ACCOUNT", Details: "ok"},
			{Timestamp: "2025-01-02T00:00:00Z", Username: "bob", Action: "DELETE_ACCOUNT", Details: "removed"},
		},
	}
	m.filter = ""
	m.filterCol = 0
	m.rebuildTableRows()
	rows := m.table.Rows()
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows after rebuild, got %d", len(rows))
	}

	// Try filter by username
	m.filter = "bob"
	m.filterCol = 2
	m.rebuildTableRows()
	rows = m.table.Rows()
	if len(rows) != 1 {
		t.Fatalf("expected 1 row when filtering by bob, got %d", len(rows))
	}
}

func TestPublicKeysViewUsageReport(t *testing.T) {
	i18n.Init("en")
	m := &publicKeysModel{}
	// empty usage
	m.usageReportKey = model.PublicKey{Comment: "k1"}
	m.usageReportAccts = nil
	out := m.viewUsageReport()
	if !strings.Contains(out, "No audit log entries") && !strings.Contains(out, "No accounts") && len(out) == 0 {
		// not a strict assertion, just ensure function runs
		t.Fatalf("unexpected empty usage report output: %q", out)
	}

	// populate usage
	m.usageReportAccts = []model.Account{{Username: "alice", Hostname: "h1", Label: "web"}}
	out = m.viewUsageReport()
	if !strings.Contains(out, "- ") || !strings.Contains(out, "alice") {
		t.Fatalf("expected usage report to list accounts, got: %q", out)
	}
}

func TestManyViews_RenderNonEmpty(t *testing.T) {
	i18n.Init("en")

	// accountsModel.View
	am := &accountsModel{
		accounts: []model.Account{{ID: 1, Username: "alice", Hostname: "h1", Label: "web"}},
		width:    120,
		height:   30,
	}
	if v := am.View(); v == "" {
		t.Fatalf("accountsModel.View returned empty string")
	}

	// publicKeysModel.View
	pk := &publicKeysModel{
		keys:   []model.PublicKey{{ID: 1, Comment: "k1", Algorithm: "ssh-ed25519"}},
		width:  80,
		height: 24,
	}
	if v := pk.View(); v == "" {
		t.Fatalf("publicKeysModel.View returned empty string")
	}

	// assignKeysModel.View
	ak := &assignKeysModel{
		accounts: []model.Account{{ID: 1, Username: "a", Hostname: "h"}},
		keys:     []model.PublicKey{{ID: 2, Comment: "k2", Algorithm: "ssh-rsa"}},
		width:    100,
		height:   30,
	}
	if v := ak.View(); v == "" {
		t.Fatalf("assignKeysModel.View returned empty string")
	}

	// accountFormModel.View (zero value should not panic)
	var af accountFormModel
	_ = af.View() // acceptable but ensure no panic
}

