// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package tui

import (
	"testing"

	"github.com/toeirei/keymaster/internal/core"
	"github.com/toeirei/keymaster/internal/core/db"
	"github.com/toeirei/keymaster/internal/i18n"
	"github.com/toeirei/keymaster/internal/model"
	"github.com/toeirei/keymaster/internal/testutil"
)

func TestContainsIgnoreCase(t *testing.T) {
	if !core.ContainsIgnoreCase("HelloWorld", "hell") {
		t.Fatalf("expected containsIgnoreCase to find substring ignoring case")
	}
	if !core.ContainsIgnoreCase("abc", "") {
		t.Fatalf("empty substr should return true")
	}
	if core.ContainsIgnoreCase("abc", "z") {
		t.Fatalf("expected false for non-matching substring")
	}
}

func TestFilteredKeys(t *testing.T) {
	m := &assignKeysModel{
		keys: []model.PublicKey{
			{ID: 1, Comment: "alice@host", Algorithm: "ssh-ed25519"},
			{ID: 2, Comment: "bob@host", Algorithm: "ssh-rsa"},
			{ID: 3, Comment: "carol@host", Algorithm: "ssh-ed25519"},
		},
	}

	m.keyFilter = "ed25519"
	fk := m.filteredKeys()
	if len(fk) != 2 {
		t.Fatalf("expected 2 filtered keys, got %d", len(fk))
	}

	m.keyFilter = "bob"
	fk = m.filteredKeys()
	if len(fk) != 1 || fk[0].Comment != "bob@host" {
		t.Fatalf("expected single bob key, got %v", fk)
	}
}

func TestRebuildDisplayedAccounts(t *testing.T) {
	// Ensure deterministic server-side search during this test.
	defer db.ClearDefaultAccountSearcher()
	db.SetDefaultAccountSearcher(&testutil.FakeAccountSearcher{})

	m := &accountsModel{
		accounts: []model.Account{
			{ID: 1, Username: "alice", Hostname: "h1", Label: "web", Tags: "role:web"},
			{ID: 2, Username: "bob", Hostname: "h2", Label: "db", Tags: "role:db"},
		},
		cursor: 5,
	}

	// No filter -> displayed equals accounts
	m.filter = ""
	m.rebuildDisplayedAccounts()
	if len(m.displayedAccounts) != 2 {
		t.Fatalf("expected 2 displayed accounts, got %d", len(m.displayedAccounts))
	}
	// Filter by tag
	m.filter = "role:web"
	m.rebuildDisplayedAccounts()
	if len(m.displayedAccounts) != 1 || m.displayedAccounts[0].Username != "alice" {
		t.Fatalf("filtering by tag failed: %v", m.displayedAccounts)
	}
	// Cursor reset when out of bounds
	m.cursor = 10
	m.filter = "no-match"
	m.rebuildDisplayedAccounts()
	if len(m.displayedAccounts) != 0 {
		t.Fatalf("expected no displayed accounts, got %d", len(m.displayedAccounts))
	}
	if m.cursor != 0 {
		t.Fatalf("expected cursor reset to 0, got %d", m.cursor)
	}
}

func TestPadCell_TruncatesAndPads(t *testing.T) {
	s := "abcdef"
	got := padCell(s, 4)
	if got != "abcd" {
		t.Fatalf("expected truncation to 'abcd', got %q", got)
	}
	got = padCell("x", 3)
	if got != "x  " {
		t.Fatalf("expected padding to 'x  ', got %q", got)
	}
}

func TestBoolToYesNo_Localized(t *testing.T) {
	i18n.Init("en")
	if boolToYesNo(true) != "Yes" {
		t.Fatalf("expected 'Yes' for true in en")
	}
	if boolToYesNo(false) != "No" {
		t.Fatalf("expected 'No' for false in en")
	}
	i18n.SetLang("de")
	if boolToYesNo(true) == "Yes" {
		t.Fatalf("expected localized value in de for true")
	}
}
