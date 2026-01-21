// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package router

import (
	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/ui/tui/util"
)

var routerId = 1 // TODO change to atomic int... just to be sure

type Model struct {
	id          int
	size        util.Size
	model_stack []*util.Model
}

func New(initial_model *util.Model) (*Model, Controll) {
	routerId++
	return &Model{
			id:          routerId - 1,
			model_stack: []*util.Model{initial_model},
		}, Controll{
			rid: routerId - 1,
		}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		(*m.activeModelGet()).Init(),
		m.activeModelUpdate(InitMsg{RouterControll: Controll{rid: m.id}}),
		m.activeModelUpdate(tea.WindowSizeMsg(m.size)),
	)
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd

	if m.size.Update(msg) {
		// pass window size messages
		cmd = m.activeModelUpdate(msg)
	} else if m.isMsgOwner(msg) {
		// handle controll messages meant for this router
		switch msg := msg.(type) {
		case PushMsg:
			cmd = m.handlePush(msg)
		case PopMsg:
			cmd = m.handlePop(msg)
		case ChangeMsg:
			cmd = m.handleChange(msg)
		}
	} else if IsRouterMsg(msg) {
		// do not pass init messages, to prevent childs from obtaining parent routers RouterControll
		if _, ok := msg.(InitMsg); !ok {
			// pass other controll messages for child routers
			cmd = m.activeModelUpdate(msg)
		}
	} else {
		// pass other messages
		cmd = m.activeModelUpdate(msg)
	}

	return cmd
}

func (m Model) View() string {
	return (*m.activeModelGet()).View()
}

func (m *Model) Focus() (tea.Cmd, help.KeyMap) {
	return (*m.activeModelGet()).Focus()
}

func (m *Model) Blur() {
	(*m.activeModelGet()).Blur()
}

// *Model implements util.Model
var _ util.Model = (*Model)(nil)

func (m *Model) isMsgOwner(msg tea.Msg) bool {
	rmsg, ok := msg.(RouterMsg)
	return ok && rmsg.routerId() == m.id
}

func IsRouterMsg(msg tea.Msg) bool {
	_, ok := msg.(RouterMsg)
	return ok
}
