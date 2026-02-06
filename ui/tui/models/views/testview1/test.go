// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package testview1

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/ui/tui/models/components/router"
	"github.com/toeirei/keymaster/ui/tui/util"
)

type Model struct {
	rc router.Controll
}

func New(rc router.Controll) *Model {
	return &Model{rc}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	// handle keys messages
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(msg, DefaultKeyMap.Quit):
			return m.rc.Pop(1)
		}
	}
	return nil
}

func (m Model) View() string {
	return "basically nothing to see here"
}

func (m *Model) Focus(baseKeyMap help.KeyMap) tea.Cmd {
	return util.AnnounceKeyMapCmd(baseKeyMap, DefaultKeyMap)
}

func (m *Model) Blur() {}

// *Model implements util.Model
var _ util.Model = (*Model)(nil)
