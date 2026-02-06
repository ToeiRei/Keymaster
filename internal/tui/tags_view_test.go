// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/internal/core/model"
)

func TestRebuildLines_ExpandedAndFilter(t *testing.T) {
	m := tagsViewModel{}
	m.accountsByTag = map[string][]model.Account{
		"alpha": {{Username: "u1", Hostname: "h1", Label: "L1"}},
		"beta":  {{Username: "u2", Hostname: "h2", Label: "L2"}, {Username: "u3", Hostname: "h3"}},
	}
	m.sortedTags = []string{"alpha", "beta"}
	m.expanded = map[string]bool{"alpha": true}
	m.filter = ""
	m.cursor = 0

	m.rebuildLines()

	// Expect lines: "alpha", account, "beta"
	if len(m.lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(m.lines))
	}
	if tag, ok := m.lines[0].(string); !ok || tag != "alpha" {
		t.Fatalf("expected first line to be tag 'alpha', got %#v", m.lines[0])
	}
	if _, ok := m.lines[1].(model.Account); !ok {
		t.Fatalf("expected second line to be Account, got %#v", m.lines[1])
	}
	if tag, ok := m.lines[2].(string); !ok || tag != "beta" {
		t.Fatalf("expected third line to be tag 'beta', got %#v", m.lines[2])
	}
}

func TestUpdate_FilteringMode_InputAndBackspace(t *testing.T) {
	m := tagsViewModel{}
	m.sortedTags = []string{"alpha", "beta", "gamma"}
	m.accountsByTag = map[string][]model.Account{}
	m.isFiltering = true
	m.filter = ""

	// Type 'g'
	modelIface, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	mv := modelIface.(tagsViewModel)
	if mv.filter != "g" {
		t.Fatalf("expected filter to be 'g', got %q", mv.filter)
	}
	// Type 'a'
	modelIface, _ = mv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	mv = modelIface.(tagsViewModel)
	if mv.filter != "ga" {
		t.Fatalf("expected filter to be 'ga', got %q", mv.filter)
	}
	// Backspace
	modelIface, _ = mv.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	mv = modelIface.(tagsViewModel)
	if mv.filter != "g" {
		t.Fatalf("expected filter to be 'g' after backspace, got %q", mv.filter)
	}
	// Enter to finish filtering
	modelIface, _ = mv.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mv = modelIface.(tagsViewModel)
	if mv.isFiltering {
		t.Fatalf("expected isFiltering to be false after Enter")
	}
}

func TestUpdate_ToggleExpandOnEnter(t *testing.T) {
	m := tagsViewModel{}
	m.sortedTags = []string{"alpha", "beta"}
	m.accountsByTag = map[string][]model.Account{
		"alpha": {{Username: "u1", Hostname: "h1"}},
	}
	m.expanded = make(map[string]bool)
	m.rebuildLines()
	// ensure cursor at first tag
	m.cursor = 0
	// Press enter key
	modelIface, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mv := modelIface.(tagsViewModel)
	if !mv.expanded["alpha"] {
		t.Fatalf("expected 'alpha' to be expanded after Enter")
	}
}
