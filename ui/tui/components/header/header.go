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
const logoTitle string = "🔑 Keymaster"
const logoTagline string = "An agentless SSH key manager that just does the job."
const logo string = logoTitle + "\n" + logoTagline

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
	m.size.UpdateFromMsg(msg)
	return nil
}

func (m Model) View() string {
	brand := lipgloss.NewStyle().
		Bold(true).
		Render(logoTitle) + "\n" +
		lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			Render(logoTagline)

	return lipgloss.
		NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		Render(lipgloss.PlaceHorizontal(
			m.size.Width,
			lipgloss.Left,
			brand,
		))
}

func (m *Model) Focus(parentKeyMap help.KeyMap) tea.Cmd {
	return util.AnnounceKeyMapCmd(parentKeyMap)
}

func (m *Model) Blur() {}

// *[Model] implements [util.Model]
var _ util.Model = (*Model)(nil)
