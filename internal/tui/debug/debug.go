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

	// Use full terminal dimensions (colored bars provide visual structure).
	frameW := m.width
	frameH := m.height

	// Calculate layout measurements following the canonical contract.
	navWidth := 24
	if frameW < 60 {
		navWidth = frameW / 3
	}
	sepWidth := 3 // " │ "
	bodyWidth := frameW - navWidth - sepWidth

	// Calculate available height.
	// Account for: header (2) + footer (1) + 2 horizontal separators
	headerHeight := 2
	footerHeight := 1
	separatorLines := 2 // top and bottom box lines
	mainHeight := frameH - headerHeight - footerHeight - separatorLines
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

	// Step 5b: Create horizontal separators for box effect.
	hSep := m.renderHorizontalSeparator()

	// Step 6: Render footer.
	footer := m.renderFooter()

	// Step 7: Compose final layout (vertical join of header, main, footer with separators).
	final := lipgloss.JoinVertical(lipgloss.Left, headerBlock, hSep, main, hSep, footer)

	return final
}

// renderHeader produces the 2-row header block with background.
func (m testModel) renderHeader() string {
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color("60")).
		Bold(true).
		Width(m.width)

	subtitleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color("60")).
		Width(m.width)

	title := titleStyle.Render("🔑 Keymaster — Layout Test")
	subtitle := subtitleStyle.Render("An agentless SSH key manager that just does the job.")

	return lipgloss.JoinVertical(lipgloss.Left, title, subtitle)
}

// renderNav produces the left navigation pane.
func (m testModel) renderNav(width, height int) string {
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("12")).
		Bold(true)

	header := headerStyle.Render("🗂️  Navigation")

	style := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Padding(1, 0, 0, 0)

	m.menu.SetSize(width, height-3) // account for header and margins
	rendered := m.menu.Render()

	// Compose: header + empty line + menu
	emptyLine := lipgloss.NewStyle().Width(width).Render("")
	nav := lipgloss.JoinVertical(lipgloss.Left, header, emptyLine, rendered)

	return style.Render(nav)
}

// renderSeparator produces the vertical separator pane.
func (m testModel) renderSeparator(height int) string {
	style := lipgloss.NewStyle().
		Width(3).
		Height(height)

	// Pad separator to match height, but skip first line to align with nav header.
	lines := make([]string, height)
	lines[0] = "" // empty line at top
	for i := 1; i < height; i++ {
		lines[i] = " │ "
	}

	return style.Render(lipgloss.JoinVertical(lipgloss.Center, lines...))
}

// renderBody produces the right body pane with viewport.
func (m testModel) renderBody(width, height int) string {
	// Create header for breathing room and context.
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("12")).
		Bold(true)

	header := headerStyle.Render("📋 Overview")

	// Render viewport with adjusted height for header and margins.
	headerHeight := 1
	topMargin := 1
	bottomMargin := 1
	contentHeight := height - headerHeight - topMargin - bottomMargin
	if contentHeight < 3 {
		contentHeight = 3
	}

	m.vp.Width = width
	m.vp.Height = contentHeight

	// Compose: empty line + header + empty line + content
	emptyLine := lipgloss.NewStyle().Width(width).Render("")
	body := lipgloss.JoinVertical(lipgloss.Left, emptyLine, header, emptyLine, m.vp.View())

	style := lipgloss.NewStyle().
		Width(width).
		Height(height)

	return style.Render(body)
}

// renderFooter produces the bottom footer.
func (m testModel) renderFooter() string {
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color("60")).
		Width(m.width)

	text := " j/k body scroll  J/K menu  q quit"
	return footerStyle.Render(text)
}

// renderHorizontalSeparator produces a horizontal line for box framing.
func (m testModel) renderHorizontalSeparator() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Width(m.width)

	line := ""
	for i := 0; i < m.width; i++ {
		line += "─"
	}
	return style.Render(line)
}
