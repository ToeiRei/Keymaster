// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package debug

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/internal/tui/frame"
)

// Launch runs the development-only test screen. It returns immediately if the
// environment variable KEYMASTER_TUI_TEST is not set to "1".
func Launch() {
	if os.Getenv("KEYMASTER_TUI_TEST") != "1" {
		return
	}
	if _, err := tea.NewProgram(newTestModel()).Run(); err != nil {
		// On failure, print to stderr and exit non-zero to aid debugging.
		os.Stderr.WriteString("test screen error: " + err.Error() + "\n")
		os.Exit(1)
	}
}

type testModel struct {
	vp     viewport.Model
	width  int
	height int
	menu   *frame.ListView
}

func newTestModel() testModel {
	m := testModel{
		vp: viewport.New(20, 5),
	}
	// Populate the viewport with sample multi-line content including unicode.
	sample := "This is a TUI framework test screen.\n"
	sample += "It shows header, scrollable body and footer.\n"
	sample += "Unicode test: ✓ ✅ ✨ — 漢字 — テスト\n"
	for i := 0; i < 40; i++ {
		sample += fmt.Sprintf("Line %c — The quick brown fox jumps over the lazy dog.\n", 'A'+(i%26))
	}
	m.vp.SetContent(sample)
	// Create a simple left-hand menu list for the dashboard mock
	menuItems := []string{"Overview", "Accounts", "Public Keys", "Deploy", "Audit", "Tags"}
	ml := frame.NewList(menuItems)
	m.menu = ml
	return m
}

func (m testModel) Init() tea.Cmd { return nil }

func (m testModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "j", "down":
			m.vp.LineDown(1)
			m.menu.MoveDown()
		case "k", "up":
			m.vp.LineUp(1)
			m.menu.MoveUp()
		}
	case tea.WindowSizeMsg:
		// Reserve 3 lines for header/footer and padding
		m.width = msg.Width
		m.height = msg.Height
		bodyHeight := msg.Height - 3
		if bodyHeight < 3 {
			bodyHeight = 3
		}
		m.vp.Height = bodyHeight
		m.vp.Width = msg.Width
	}
	return m, nil
}

func (m testModel) View() string {
	// Global header (emoji + app + view)
	title := "🔑 Keymaster — TUI Framework Test Screen"
	subtitle := "An agentless SSH key manager that just does the job."

	// center the title and subtitle within the total width
	center := func(s string) string {
		w := m.width
		if w <= 0 {
			return s + "\n"
		}
		rs := []rune(s)
		pad := (w - len(rs)) / 2
		if pad < 0 {
			pad = 0
		}
		return strings.Repeat(" ", pad) + s + "\n"
	}

	// Build the column body (left + separator + right)
	leftWidth := 24
	if m.width < 60 {
		leftWidth = m.width / 3
	}
	sep := " │ "
	// Reserve 2 chars for the outer box borders so combined columns fit the
	// inner width (m.width - 2).
	rightWidth := m.width - leftWidth - len(sep) - 2

	// Left column: menu
	m.menu.SetSize(leftWidth, m.height-2)
	left := m.menu.Render()

	// Right column: pane with viewport content using structured Pane layout
	right := frame.NewPane()
	right.SetHeader("🔑 Dashboard Overview")
	// compact navigation row for the pane (small contextual actions)
	right.SetNav("[r] refresh | [s] search | [o] options")
	right.SetBodyMargin(2)
	right.SetFooterTokens("h Help     q Quit", "")
	right.SetViewport(&m.vp)
	right.SetSize(rightWidth, m.height)

	// Combine columns with a separator into b
	var b strings.Builder
	leftLines := strings.Split(strings.TrimRight(left, "\n"), "\n")
	rightLines := strings.Split(strings.TrimRight(right.View(), "\n"), "\n")
	maxLines := len(leftLines)
	if len(rightLines) > maxLines {
		maxLines = len(rightLines)
	}
	for i := 0; i < maxLines; i++ {
		l := ""
		if i < len(leftLines) {
			l = leftLines[i]
		}
		r := ""
		if i < len(rightLines) {
			r = rightLines[i]
		}
		// pad left to fixed width
		pad := leftWidth - len([]rune(l))
		if pad < 0 {
			pad = 0
		}
		// vertical separator between columns
		b.WriteString(l + strings.Repeat(" ", pad) + sep + r + "\n")
	}
	// body content ready
	body := b.String()

	// If width is too small, fall back to plain output
	// If width is too small, fall back to plain output
	if m.width < 10 {
		return center(title) + center(subtitle) + "\n" + body
	}

	// Wrap the combined columns inside a top-level Pane so header/nav/footer
	// and separators are produced by the Pane primitive. This keeps layout
	// consistent and avoids mixing lipgloss-style joins.
	outer := frame.NewPane()
	// Use a left-aligned single-line header: icon + app + view
	outer.SetHeader("🔑 Keymaster – TUI Framework")
	outer.SetNav("[r] refresh | [s] search | [o] options")
	outer.SetBodyMargin(2)
	// Create a viewport for the outer pane containing the combined body
	vp := viewport.New(10, 3)
	vp.SetContent(body)
	outer.SetViewport(&vp)
	outer.SetSize(m.width, m.height)
	// Pane no longer renders a footer; render a background-colored status
	// bar below the pane for help/quit information.
	return outer.View() + "\n" + frame.StatusBar("h Help", "q Quit", m.width)

}
