// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package util

import tea "github.com/charmbracelet/bubbletea"

type Size struct {
	Width  int
	Height int
}

func (s *Size) Update(msg tea.Msg) bool {
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		s.Width, s.Height = msg.Width, msg.Height
		return true
	}
	return false
}

func (s *Size) ToMsg() tea.WindowSizeMsg {
	return tea.WindowSizeMsg{
		Width:  s.Width,
		Height: s.Height,
	}
}
