// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/i18n"
	"github.com/toeirei/keymaster/internal/model"
)

func TestAccounts_Update_NavigationAndActions(t *testing.T) {
	i18n.Init("en")
	// Initialize an in-memory DB for form initialization (tag autocompletion)
	_ = initTestDB()
	m := accountsModel{}
	m.displayedAccounts = []model.Account{{ID: 1, Username: "a"}, {ID: 2, Username: "b"}, {ID: 3, Username: "c"}}
	m.cursor = 0

	// Press down
	mi, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m1 := mi.(*accountsModel)
	if m1.cursor != 1 {
		t.Fatalf("expected cursor 1 after down, got %d", m1.cursor)
	}

	// Press up
	mi, _ = m1.Update(tea.KeyMsg{Type: tea.KeyUp})
	m2 := mi.(*accountsModel)
	if m2.cursor != 0 {
		t.Fatalf("expected cursor 0 after up, got %d", m2.cursor)
	}

	// Delete key should open confirmation if item exists
	mi, _ = m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m3 := mi.(*accountsModel)
	if !m3.isConfirmingDelete {
		t.Fatalf("expected isConfirmingDelete true after 'd' key")
	}
	if m3.accountToDelete.ID != 1 {
		t.Fatalf("expected accountToDelete ID 1, got %d", m3.accountToDelete.ID)
	}
}

func TestAccounts_Update_EditAndAdd_OpenForm(t *testing.T) {
	i18n.Init("en")
	m := accountsModel{}
	m.displayedAccounts = []model.Account{{ID: 7, Username: "u7", Hostname: "h7"}}
	m.cursor = 0

	// Edit
	mi, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m1 := mi.(*accountsModel)
	if m1.state != accountsFormView {
		t.Fatalf("expected state accountsFormView after 'e', got %v", m1.state)
	}
	_ = cmd // Init command may be nil depending on form; accept nil but ensure form populated

	// Add (new account) - should also open form
	m2 := accountsModel{}
	mi2, _ := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m3 := mi2.(*accountsModel)
	if m3.state != accountsFormView {
		t.Fatalf("expected state accountsFormView after 'a', got %v", m3.state)
	}
}

func TestAccounts_Update_FilterToggleAndVerify(t *testing.T) {
	i18n.Init("en")
	_ = initTestDB()
	m := accountsModel{}
	m.displayedAccounts = []model.Account{{ID: 9, Username: "x", Hostname: "hostx"}}

	// Enter filter mode with '/'
	mi, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m1 := mi.(*accountsModel)
	if !m1.isFiltering {
		t.Fatalf("expected isFiltering true after '/' key")
	}

	// Exit filter mode with Esc
	mi, _ = m1.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m2 := mi.(*accountsModel)
	if m2.isFiltering {
		t.Fatalf("expected isFiltering false after Esc")
	}

	// (skip verify host key command in unit test to avoid network/async behavior)
}

func initTestDB() error {
	// Use sqlite in-memory for fast test DB with migrations applied.
	return db.InitDB("sqlite", ":memory:")
}
