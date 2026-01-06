// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package debug

import (
	"fmt"
	"os"

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
	// Use the Pane primitive to compose header, body and footer so this
	// debug screen exercises the framework API used by real views.
	p := frame.NewPane()
	p.SetHeader("FRAME TEST SCREEN")
	p.SetFooterTokens("h:help m:menu", "q:quit")
	p.SetViewport(&m.vp)
	p.SetSize(m.width, m.height)
	return p.View()
}
