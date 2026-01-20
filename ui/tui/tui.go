package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/ui/tui/models/views/root"
)

func Run() error {
	_, err := tea.NewProgram(
		root.New(),
		tea.WithAltScreen(),
	).Run()
	return err
}
