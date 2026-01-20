// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package hostedit

import (
	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/ui/tui/util"
)

type Model struct{}

func New() *Model {
	return &Model{}
}

// Init implements util.Model.
func (m *Model) Init() tea.Cmd {
	panic("unimplemented")
}

// Update implements util.Model.
func (m *Model) Update(tea.Msg) tea.Cmd {
	panic("unimplemented")
}

// View implements util.Model.
func (m *Model) View() string {
	panic("unimplemented")
}

// Focus implements util.Model.
func (m *Model) Focus() (tea.Cmd, help.KeyMap) {
	panic("unimplemented")
}

// Blur implements util.Model.
func (m *Model) Blur() {
	panic("unimplemented")
}

// *Model implements util.Model
var _ util.Model = (*Model)(nil)
