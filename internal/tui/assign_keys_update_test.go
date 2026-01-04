package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/internal/i18n"
	"github.com/toeirei/keymaster/internal/model"
)

func TestAssignKeys_AccountFilteringMode(t *testing.T) {
	i18n.Init("en")
	m := &assignKeysModel{}
	m.accounts = []model.Account{{ID: 1, Username: "alice"}, {ID: 2, Username: "bob"}}

	// Enter filtering mode
	m.updateAccountSelection(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !m.isFilteringAcct {
		t.Fatalf("expected isFilteringAcct true after '/'")
	}

	// Type 'b' into filter
	m.updateAccountSelection(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	if m.accountFilter != "b" {
		t.Fatalf("expected accountFilter 'b', got %q", m.accountFilter)
	}

	// Backspace should clear
	m.updateAccountSelection(tea.KeyMsg{Type: tea.KeyBackspace})
	if m.accountFilter != "" {
		t.Fatalf("expected accountFilter '', got %q", m.accountFilter)
	}

	// Enter should exit filtering mode
	m.updateAccountSelection(tea.KeyMsg{Type: tea.KeyEnter})
	if m.isFilteringAcct {
		t.Fatalf("expected isFilteringAcct false after Enter")
	}
}

func TestAssignKeys_SwitchToKeysAndBack(t *testing.T) {
	i18n.Init("en")
	// Prepare model with one account and a couple keys
	a := model.Account{ID: 10, Username: "ua"}
	k1 := model.PublicKey{ID: 21, Comment: "k1"}
	k2 := model.PublicKey{ID: 22, Comment: "k2"}

	m := &assignKeysModel{}
	m.accounts = []model.Account{a}
	m.keys = []model.PublicKey{k1, k2}

	// Select account by simulating Enter (accountCursor defaults to 0)
	_, _ = m.updateAccountSelection(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'\n'}})

	// Force select via Enter string path
	_, _ = m.updateAccountSelection(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'\r'}})

	// Use the explicit Enter effect by calling with KeyEnter
	_, _ = m.updateAccountSelection(tea.KeyMsg{Type: tea.KeyEnter})
	// model may have transitioned state; ensure it can handle key selection updates
	// Now press 'q' to go back to account selection
	m3, _ := m.updateKeySelection(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	_ = m3
	if m.state != assignStateSelectAccount {
		t.Fatalf("expected state assignStateSelectAccount after 'q', got %v", m.state)
	}
}

func TestAssignKeys_KeyNavigationAndFiltering(t *testing.T) {
	i18n.Init("en")
	k1 := model.PublicKey{ID: 31, Comment: "one"}
	k2 := model.PublicKey{ID: 32, Comment: "two"}
	k3 := model.PublicKey{ID: 33, Comment: "three"}
	m := &assignKeysModel{}
	m.keys = []model.PublicKey{k1, k2, k3}
	m.state = assignStateSelectKeys

	// Move down (simulate 'j') twice
	m.updateKeySelection(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m.updateKeySelection(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.keyCursor != 2 {
		t.Fatalf("expected keyCursor 2 after two downs, got %d", m.keyCursor)
	}

	// Move up (simulate 'k') once
	m.updateKeySelection(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if m.keyCursor != 1 {
		t.Fatalf("expected keyCursor 1 after up, got %d", m.keyCursor)
	}

	// Enter filtering mode with '/'
	m.updateKeySelection(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !m.isFilteringKey {
		t.Fatalf("expected isFilteringKey true after '/'")
	}
	// Type 't' into filter
	m.updateKeySelection(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	if m.keyFilter != "t" {
		t.Fatalf("expected keyFilter 't', got %q", m.keyFilter)
	}
	// Exit filter with Esc
	m.updateKeySelection(tea.KeyMsg{Type: tea.KeyEsc})
	if m.isFilteringKey {
		t.Fatalf("expected isFilteringKey false after Esc")
	}
}
