// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package debug

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
		case "k", "up":
			m.vp.LineUp(1)
		case "J":
			m.menu.MoveDown()
		case "K":
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
	if m.width < 10 || m.height < 5 {
		return "Terminal too small"
	}

	// Calculate layout measurements following the canonical contract.
	navWidth := 24
	if m.width < 60 {
		navWidth = m.width / 3
	}
	sepWidth := 3 // " │ "
	bodyWidth := m.width - navWidth - sepWidth

	// Calculate available height.
	headerHeight := 2
	footerHeight := 1
	mainHeight := m.height - headerHeight - footerHeight
	if mainHeight < 3 {
		mainHeight = 3
	}

	// Step 1: Render header block (title + subtitle).
	headerBlock := m.renderHeader()

	// Step 2: Render nav pane.
	navPane := m.renderNav(navWidth, mainHeight)

	// Step 3: Render separator pane.
	sepPane := m.renderSeparator(mainHeight)

	// Step 4: Render body pane.
	bodyPane := m.renderBody(bodyWidth, mainHeight)

	// Step 5: Compose main area (horizontal join of nav, sep, body).
	main := lipgloss.JoinHorizontal(lipgloss.Top, navPane, sepPane, bodyPane)

	// Step 6: Render footer.
	footer := m.renderFooter()

	// Step 7: Compose final layout (vertical join of header, main, footer).
	final := lipgloss.JoinVertical(lipgloss.Left, headerBlock, main, footer)

	return final
}

// renderHeader produces the 2-row header block.
func (m testModel) renderHeader() string {
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("12")).
		Bold(true)

	subtitleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8"))

	title := titleStyle.Render("🔑 Keymaster — Layout Test")
	subtitle := subtitleStyle.Render("An agentless SSH key manager that just does the job.")

	return lipgloss.JoinVertical(lipgloss.Left, title, subtitle)
}

// renderNav produces the left navigation pane.
func (m testModel) renderNav(width, height int) string {
	style := lipgloss.NewStyle().
		Width(width).
		Height(height)

	m.menu.SetSize(width, height)
	rendered := m.menu.Render()

	return style.Render(rendered)
}

// renderSeparator produces the vertical separator pane.
func (m testModel) renderSeparator(height int) string {
	style := lipgloss.NewStyle().
		Width(3).
		Height(height)

	// Pad separator to match height.
	lines := make([]string, height)
	lines[0] = " │ "
	for i := 1; i < height; i++ {
		lines[i] = " │ "
	}

	return style.Render(lipgloss.JoinVertical(lipgloss.Center, lines...))
}

// renderBody produces the right body pane with viewport.
func (m testModel) renderBody(width, height int) string {
	// Update viewport dimensions.
	m.vp.Width = width
	m.vp.Height = height

	style := lipgloss.NewStyle().
		Width(width).
		Height(height)

	// For now, render viewport directly.
	return style.Render(m.vp.View())
}

// renderFooter produces the bottom footer.
func (m testModel) renderFooter() string {
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color("4")).
		Width(m.width)

	text := " j/k body scroll  J/K menu  q quit"
	return footerStyle.Render(text)
}
