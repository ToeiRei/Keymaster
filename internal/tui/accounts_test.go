// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package tui

import (
	"strings"
	"testing"

	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/model"
	"github.com/toeirei/keymaster/internal/testutil"
)

func TestRebuildDisplayedAccounts_FilteringAndListContent(t *testing.T) {
	// Inject fake searcher so server-side search behavior is deterministic.
	defer db.ClearDefaultAccountSearcher()
	db.SetDefaultAccountSearcher(&testutil.FakeAccountSearcher{})

	m := accountsModel{}
	m.accounts = []model.Account{
		{ID: 1, Username: "alice", Hostname: "host1", Label: "web", Tags: "ops", IsActive: true},
		{ID: 2, Username: "bob", Hostname: "db1", Label: "db", Tags: "infra", IsActive: false},
		{ID: 3, Username: "carol", Hostname: "host2", Label: "web", Tags: "ops,db", IsActive: true},
	}

	// No filter -> all accounts shown
	m.filter = ""
	m.rebuildDisplayedAccounts()
	if len(m.displayedAccounts) != 3 {
		t.Fatalf("expected 3 displayed accounts, got %d", len(m.displayedAccounts))
	}

	// Verify listContentView contains inactive account info when unfiltered
	contentAll := m.listContentView()
	if !strings.Contains(contentAll, "bob") && !strings.Contains(contentAll, "db1") {
		t.Fatalf("expected inactive account info to appear in full list content")
	}

	// Filter by label 'web'
	m.filter = "web"
	m.rebuildDisplayedAccounts()
	if len(m.displayedAccounts) != 2 {
		t.Fatalf("expected 2 displayed accounts when filtering 'web', got %d", len(m.displayedAccounts))
	}

	// Verify listContentView contains cursor marker when cursor set (filtered view)
	m.cursor = 1
	content := m.listContentView()
	if !strings.Contains(content, "▸") {
		t.Fatalf("expected cursor marker in filtered list content, got: %q", content)
	}
	if !strings.Contains(content, "carol") {
		t.Fatalf("expected filtered content to include 'carol', got: %q", content)
	}
}
