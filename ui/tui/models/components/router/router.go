// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package router

// TODO rewrite with util.Model in mind

import (
	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/ui/tui/util"
)

var routerId = 1

type Router struct {
	id          int
	size        util.Size
	model_stack []*util.Model
}

func New(initial_model *util.Model) (Router, RouterControll) {
	routerId++
	return Router{
			id:          routerId - 1,
			model_stack: []*util.Model{initial_model},
		}, RouterControll{
			rid: routerId - 1,
		}
}

func (r Router) Init() tea.Cmd {
	return tea.Batch(
		(*r.activeModelGet()).Init(),
		r.activeModelUpdate(InitMsg{RouterControll: RouterControll{rid: r.id}}),
		r.activeModelUpdate(tea.WindowSizeMsg(r.size)),
	)
}

func (r *Router) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd

	if r.size.Update(msg) {
		// pass window size messages
		cmd = r.activeModelUpdate(msg)
	} else if r.isMsgOwner(msg) {
		// handle controll messages meant for this router
		switch msg := msg.(type) {
		case PushMsg:
			cmd = r.handlePush(msg)
		case PopMsg:
			cmd = r.handlePop(msg)
		case ChangeMsg:
			cmd = r.handleChange(msg)
		}
	} else if IsRouterMsg(msg) {
		// do not pass init messages, to prevent childs from obtaining parent routers RouterControll
		if _, ok := msg.(InitMsg); !ok {
			// pass other controll messages for child routers
			cmd = r.activeModelUpdate(msg)
		}
	} else {
		// pass other messages
		cmd = r.activeModelUpdate(msg)
	}

	return cmd
}

func (r Router) View() string {
	return (*r.activeModelGet()).View()
}

func (r *Router) Focus() (tea.Cmd, help.KeyMap) {
	return (*r.activeModelGet()).Focus()
}

func (r *Router) Blur() {
	(*r.activeModelGet()).Blur()
}

// *Model implements util.Model
var _ util.Model = (*Router)(nil)

func (r *Router) isMsgOwner(msg tea.Msg) bool {
	rmsg, ok := msg.(RouterMsg)
	return ok && rmsg.routerId() == r.id
}

func IsRouterMsg(msg tea.Msg) bool {
	_, ok := msg.(RouterMsg)
	return ok
}
