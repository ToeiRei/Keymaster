// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package router

import (
	"sync/atomic"

	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/ui/tui/util"
)

type routerId uint32

var routerIdCounter atomic.Uint32

type Model struct {
	id           routerId
	size         util.Size
	modelStack   []*util.Model
	parentKeyMap help.KeyMap
}

func New(initialModel *util.Model) (*Model, Controll) {
	rid := routerId(routerIdCounter.Add(1))
	return &Model{
		id:         rid,
		modelStack: []*util.Model{initialModel},
	}, Controll{rid}
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

	if m.size.UpdateFromMsg(msg) {
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

func (m *Model) Focus(parentKeyMap help.KeyMap) tea.Cmd {
	m.parentKeyMap = parentKeyMap
	return (*m.activeModelGet()).Focus(parentKeyMap)
}

func (m *Model) Blur() {
	(*m.activeModelGet()).Blur()
}

// *[Model] implements [util.Model]
var _ util.Model = (*Model)(nil)

func (m *Model) GetStack() []*util.Model {
	stack := make([]*util.Model, len(m.modelStack))
	copy(stack, m.modelStack)
	return stack
}

func (m *Model) isMsgOwner(msg tea.Msg) bool {
	rmsg, ok := msg.(RouterMsg)
	return ok && rmsg.routerID() == m.id
}

func IsRouterMsg(msg tea.Msg) bool {
	_, ok := msg.(RouterMsg)
	return ok
}
