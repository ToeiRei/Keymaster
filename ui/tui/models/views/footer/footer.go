// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package footer

import (
	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/ui/tui/models/components/keyhelp"
	"github.com/toeirei/keymaster/ui/tui/util"
)

type Model struct {
	baseKeyMap help.KeyMap
	size       util.Size
	help       *keyhelp.Model
}

func New(baseKeyMap help.KeyMap) *Model {
	return &Model{
		baseKeyMap: baseKeyMap,
		help:       keyhelp.New(),
		// TODO implement and add status component
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	// catch AnnounceFocusMsg and inject baseKeyMap
	if msg, ok := msg.(util.AnnounceKeyMapMsg); ok {
		return (*m.help).Update(util.AnnounceKeyMapMsg{
			KeyMap: util.MergeKeyMaps(msg.KeyMap, m.baseKeyMap),
		})
	}

	m.size.Update(msg)
	return (*m.help).Update(msg)
}

func (m Model) view() string {
	return m.help.View()
}

func (m Model) View() string {
	h_pos := lipgloss.Left
	if m.help.Expanded {
		h_pos = lipgloss.Center
	}

	return lipgloss.
		NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(true).
		Render(lipgloss.Place(
			m.size.Width, m.size.Height,
			h_pos, lipgloss.Top,
			m.view(),
		))
}

func (m *Model) Focus(baseKeyMap help.KeyMap) tea.Cmd {
	return m.help.Focus(baseKeyMap)
}

func (m *Model) Blur() {
	m.help.Blur()
}

// *Model implements util.Model
var _ util.Model = (*Model)(nil)

func (m *Model) ToggleExpanded() {
	m.help.ToggleExpanded()
}
