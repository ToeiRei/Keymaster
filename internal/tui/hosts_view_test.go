// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/internal/model"
)

func TestHostsRebuildLines_ExpandedAndFilter(t *testing.T) {
	m := hostsViewModel{}
	m.accountsByHost = map[string][]model.Account{
		"host1.example.com": {{Username: "u1", Hostname: "host1.example.com", Label: "L1"}},
		"host2.example.com": {{Username: "u2", Hostname: "host2.example.com", Label: "L2"}, {Username: "u3", Hostname: "host2.example.com"}},
	}
	m.sortedHosts = []string{"host1.example.com", "host2.example.com"}
	m.expanded = map[string]bool{"host1.example.com": true}
	m.filter = ""
	m.cursor = 0

	m.rebuildLines()

	// Expect lines: "host1.example.com", account, "host2.example.com"
	if len(m.lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(m.lines))
	}
	if host, ok := m.lines[0].(string); !ok || host != "host1.example.com" {
		t.Fatalf("expected first line to be host 'host1.example.com', got %#v", m.lines[0])
	}
	if _, ok := m.lines[1].(model.Account); !ok {
		t.Fatalf("expected second line to be Account, got %#v", m.lines[1])
	}
	if host, ok := m.lines[2].(string); !ok || host != "host2.example.com" {
		t.Fatalf("expected third line to be host 'host2.example.com', got %#v", m.lines[2])
	}
}

func TestHostsUpdate_FilteringMode_InputAndBackspace(t *testing.T) {
	m := hostsViewModel{}
	m.sortedHosts = []string{"host1.example.com", "host2.example.com", "gateway.local"}
	m.accountsByHost = map[string][]model.Account{}
	m.isFiltering = true
	m.filter = ""

	// Type 'g'
	modelIface, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	mv := modelIface.(hostsViewModel)
	if mv.filter != "g" {
		t.Fatalf("expected filter to be 'g', got %q", mv.filter)
	}
	// Type 'a'
	modelIface, _ = mv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	mv = modelIface.(hostsViewModel)
	if mv.filter != "ga" {
		t.Fatalf("expected filter to be 'ga', got %q", mv.filter)
	}
	// Backspace
	modelIface, _ = mv.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	mv = modelIface.(hostsViewModel)
	if mv.filter != "g" {
		t.Fatalf("expected filter to be 'g' after backspace, got %q", mv.filter)
	}
	// Enter to finish filtering
	modelIface, _ = mv.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mv = modelIface.(hostsViewModel)
	if mv.isFiltering {
		t.Fatalf("expected isFiltering to be false after Enter")
	}
}

func TestHostsUpdate_ToggleExpandOnEnter(t *testing.T) {
	m := hostsViewModel{}
	m.sortedHosts = []string{"host1.example.com", "host2.example.com"}
	m.accountsByHost = map[string][]model.Account{
		"host1.example.com": {{Username: "u1", Hostname: "host1.example.com"}},
	}
	m.expanded = make(map[string]bool)
	m.rebuildLines()
	// ensure cursor at first host
	m.cursor = 0
	// Press enter key
	modelIface, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mv := modelIface.(hostsViewModel)
	if !mv.expanded["host1.example.com"] {
		t.Fatalf("expected 'host1.example.com' to be expanded after Enter")
	}
}
