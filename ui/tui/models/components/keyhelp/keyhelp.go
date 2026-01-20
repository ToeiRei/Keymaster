package keyhelp

import (
	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/ui/tui/util"
)

type Model struct {
	KeyMap   help.KeyMap
	size     util.Size
	help     help.Model
	Expanded bool
}

func New() *Model {
	return &Model{
		help: help.New(),
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	if m.size.Update(msg) {
		m.help.Width = m.size.Width
		return nil
	}

	if msg, ok := msg.(util.AnnounceKeyMapMsg); ok {
		m.KeyMap = msg.KeyMap
		return nil
	}

	return nil
}

func (m Model) View() string {
	if m.KeyMap != nil {
		if !m.Expanded {
			return ShortHelpView(m.help, m.KeyMap.ShortHelp())
		} else {
			return FullHelpView(m.help, m.KeyMap.FullHelp())
		}
	}
	return ""
}

func (m *Model) Focus() (tea.Cmd, help.KeyMap) {
	return nil, nil
}

func (m *Model) Blur() {}

// *Model implements util.Model
var _ util.Model = (*Model)(nil)

func (m *Model) ToggleExpanded() {
	m.Expanded = !m.Expanded
}
