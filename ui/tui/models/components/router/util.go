// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package router

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/ui/tui/util"
)

func (m *Model) activeModelGet() *util.Model {
	return m.model_stack[len(m.model_stack)-1]
}

func (m *Model) activeModelSet(model *util.Model) {
	m.model_stack[len(m.model_stack)-1] = model
}

func (m *Model) activeModelPop() *util.Model {
	model := m.activeModelGet()
	m.model_stack = m.model_stack[:len(m.model_stack)-1]
	return model
}

func (m *Model) activeModelUpdate(msg tea.Msg) tea.Cmd {
	return (*m.activeModelGet()).Update(msg)
}

func (m *Model) activeModelFocus() tea.Cmd {
	cmd, keyMap := (*m.activeModelGet()).Focus()
	return tea.Batch(cmd, util.AnnounceKeyMapCmd(keyMap))
}

func (m *Model) activeModelInit() tea.Cmd {
	return tea.Sequence(
		(*m.activeModelGet()).Init(),
		m.activeModelUpdate(InitMsg{RouterControll: Controll{rid: m.id}}),
		m.activeModelUpdate(tea.WindowSizeMsg(m.size)),
		m.activeModelFocus(),
	)
}
