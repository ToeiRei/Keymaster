// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package menu

import (
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/ui/tui/util"
	"github.com/toeirei/keymaster/util/slicest"
)

type Model struct {
	Items       []Item
	ActiveStack []int
	size        util.Size
	focused     bool
}

func New(items ...Item) *Model {
	return &Model{
		Items:       items,
		ActiveStack: []int{0},
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	m.size.Update(msg)

	if m.focused {
		if msg, ok := msg.(tea.KeyMsg); ok {
			switch {
			case key.Matches(msg, DefaultKeyMap.Up):
				m.up()
			case key.Matches(msg, DefaultKeyMap.Down):
				m.down()
			case key.Matches(msg, DefaultKeyMap.Left):
				m.left()
			case key.Matches(msg, DefaultKeyMap.Right):
				return m.right()
			}
		}
	}
	return nil
}

func (m *Model) view() string {
	// Render Menu
	view := renderItems(m.Items, m.ActiveStack)

	// Clip view if too big for viewport
	height := lipgloss.Height(view)
	if height > m.size.Height {
		// TODO comment
		align := float64(slicest.Reduce(m.ActiveStack, func(i int, sum int) int { return sum + i + 1 })) / float64(height)
		lines := strings.Split(view, "\n")
		i := int(float64(height-m.size.Height) * align)
		view = strings.Join(lines[i:i+m.size.Height], "\n")
	}
	return view
}

func (m Model) View() string {
	return lipgloss.
		NewStyle().
		MaxWidth(m.size.Width).
		MaxHeight(m.size.Height).
		Margin(0, 1).
		Render(m.view())
}

func (m *Model) Focus(baseKeyMap help.KeyMap) tea.Cmd {
	m.focused = true
	return util.AnnounceKeyMapCmd(baseKeyMap, DefaultKeyMap)
}
func (m *Model) Blur() {
	m.focused = false
}

// *Model implements util.Model
var _ util.Model = (*Model)(nil)
