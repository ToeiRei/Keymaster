// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package debug

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func footerRuneCount(s string) int {
	return len([]rune(s))
}

func TestDebugScreen_FooterWidth80(t *testing.T) {
	m := newTestModel()
	_, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	v := m.View()
	lines := strings.Split(v, "\n")
	if len(lines) < 1 {
		t.Fatalf("unexpected empty view")
	}
	footer := lines[len(lines)-1]
	if got := footerRuneCount(footer); got != 80 {
		t.Fatalf("footer width mismatch: want=80 got=%d footer=%q", got, footer)
	}
}

func TestDebugScreen_FooterWidth120(t *testing.T) {
	m := newTestModel()
	_, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	v := m.View()
	lines := strings.Split(v, "\n")
	if len(lines) < 1 {
		t.Fatalf("unexpected empty view")
	}
	footer := lines[len(lines)-1]
	if got := footerRuneCount(footer); got != 120 {
		t.Fatalf("footer width mismatch: want=120 got=%d footer=%q", got, footer)
	}
}

