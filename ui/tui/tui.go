// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/ui/tui/models/views/root"
)

func Run() error {
	_, err := tea.NewProgram(
		root.New(),
		tea.WithAltScreen(),
		// tea.WithMouseCellMotion(),
	).Run()
	return err
}
