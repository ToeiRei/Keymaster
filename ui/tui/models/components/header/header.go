// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package header

import (
	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/ui/tui/util"
)

const logo2 string = "" +
	"╦╔═┌─┐┬ ┬┌┬┐┌─┐┌─┐┌┬┐┌─┐┬─┐\n" +
	"╠╩╗├┤ └┬┘│││├─┤└─┐ │ ├┤ ├┬┘\n" +
	"╩ ╩└─┘ ┴ ┴ ┴┴ ┴└─┘ ┴ └─┘┴└─"
const logo string = "🗝️ Master 🔑"

// Keep `logo2` available for the TUI rewrite. Reference it so linters
// don't flag it as unused while the new UI consumes it.
var _ = logo2

type Model struct {
	size util.Size
}

func New() *Model {
	return &Model{}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	m.size.Update(msg)
	return nil
}

func (m Model) View() string {
	return lipgloss.
		NewStyle().
		Border(lipgloss.NormalBorder(), false).
		BorderBottom(true).
		Render(lipgloss.PlaceHorizontal(
			m.size.Width,    //m.size.Height-1,
			lipgloss.Center, //lipgloss.Center,
			logo,
		))
}

func (m *Model) Focus() (tea.Cmd, help.KeyMap) {
	return nil, nil
}

func (m *Model) Blur() {}

// *Model implements util.Model
var _ util.Model = (*Model)(nil)
