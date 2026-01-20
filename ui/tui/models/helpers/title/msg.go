package windowtitle

import tea "github.com/charmbracelet/bubbletea"

type titleMsg string

func Set(title string) tea.Cmd {
	return func() tea.Msg { return titleMsg(title) }
}
