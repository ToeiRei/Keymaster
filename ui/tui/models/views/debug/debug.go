// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package debug

import (
	"fmt"
	"slices"

	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/ui/tui/util"
	"github.com/toeirei/keymaster/util/slicest"
)

type Model struct {
	msgs []tea.Msg
}

func New() *Model {
	return &Model{}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	m.msgs = append(m.msgs, msg)
	return nil
}

func (m Model) View() string {
	lines := slicest.Map(m.msgs, func(msg tea.Msg) string {
		if msg, ok := msg.(tea.KeyMsg); ok {
			return fmt.Sprintf(`- Key Press: "%s"`, msg.String())
		}
		return fmt.Sprintf("- %#v", msg)
	})
	slices.Reverse(lines)
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m *Model) Focus(baseKeyMap help.KeyMap) tea.Cmd {
	m.msgs = append(m.msgs, tea.Msg("i got focussed bitch!"))
	return util.AnnounceKeyMapCmd(baseKeyMap)
}

func (m *Model) Blur() {
	m.msgs = append(m.msgs, tea.Msg("i got blurred bitch!"))
}

// *Model implements util.Model
var _ util.Model = (*Model)(nil)
