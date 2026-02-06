// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/internal/core/model"
)

func TestKeysDeploymentRebuildLines_ExpandedAndFilter(t *testing.T) {
	m := keysDeploymentViewModel{}
	m.accountsByKey = map[string][]model.Account{
		"key-one": {{Username: "u1", Hostname: "h1.example.com", Label: "Web1"}},
		"key-two": {{Username: "u2", Hostname: "h2.example.com", Label: "DB1"}, {Username: "u3", Hostname: "h3.example.com"}},
	}
	m.sortedKeys = []string{"key-one", "key-two"}
	m.expanded = map[string]bool{"key-one": true}
	m.keysByComment = map[string]*model.PublicKey{
		"key-one": {ID: 1, Comment: "key-one", Algorithm: "ssh-ed25519"},
		"key-two": {ID: 2, Comment: "key-two", Algorithm: "ssh-rsa"},
	}
	m.filter = ""
	m.cursor = 0

	m.rebuildLines()

	// Expect lines: "key-one", account, "key-two"
	if len(m.lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(m.lines))
	}
	if keyComment, ok := m.lines[0].(string); !ok || keyComment != "key-one" {
		t.Fatalf("expected first line to be key 'key-one', got %#v", m.lines[0])
	}
	if _, ok := m.lines[1].(model.Account); !ok {
		t.Fatalf("expected second line to be Account, got %#v", m.lines[1])
	}
	if keyComment, ok := m.lines[2].(string); !ok || keyComment != "key-two" {
		t.Fatalf("expected third line to be key 'key-two', got %#v", m.lines[2])
	}
}

func TestKeysDeploymentUpdate_FilteringMode_InputAndBackspace(t *testing.T) {
	m := keysDeploymentViewModel{}
	m.sortedKeys = []string{"dev-key", "prod-key", "staging-key"}
	m.accountsByKey = map[string][]model.Account{}
	m.keysByComment = map[string]*model.PublicKey{}
	m.isFiltering = true
	m.filter = ""

	// Type 'p'
	modelIface, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	mv := modelIface.(keysDeploymentViewModel)
	if mv.filter != "p" {
		t.Fatalf("expected filter to be 'p', got %q", mv.filter)
	}
	// Type 'r'
	modelIface, _ = mv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	mv = modelIface.(keysDeploymentViewModel)
	if mv.filter != "pr" {
		t.Fatalf("expected filter to be 'pr', got %q", mv.filter)
	}
	// Backspace
	modelIface, _ = mv.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	mv = modelIface.(keysDeploymentViewModel)
	if mv.filter != "p" {
		t.Fatalf("expected filter to be 'p' after backspace, got %q", mv.filter)
	}
	// Enter to finish filtering
	modelIface, _ = mv.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mv = modelIface.(keysDeploymentViewModel)
	if mv.isFiltering {
		t.Fatalf("expected isFiltering to be false after Enter")
	}
}

func TestKeysDeploymentUpdate_ToggleExpandOnEnter(t *testing.T) {
	m := keysDeploymentViewModel{}
	m.sortedKeys = []string{"key-alpha", "key-beta"}
	m.accountsByKey = map[string][]model.Account{
		"key-alpha": {{Username: "u1", Hostname: "h1.example.com"}},
	}
	m.keysByComment = map[string]*model.PublicKey{
		"key-alpha": {ID: 1, Comment: "key-alpha", Algorithm: "ssh-ed25519"},
	}
	m.expanded = make(map[string]bool)
	m.rebuildLines()
	// ensure cursor at first key
	m.cursor = 0
	// Press enter key
	modelIface, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mv := modelIface.(keysDeploymentViewModel)
	if !mv.expanded["key-alpha"] {
		t.Fatalf("expected 'key-alpha' to be expanded after Enter")
	}
}
