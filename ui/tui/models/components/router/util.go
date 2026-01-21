// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package router

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/ui/tui/util"
)

func (r *Router) activeModelGet() *util.Model {
	return r.model_stack[len(r.model_stack)-1]
}

func (r *Router) activeModelSet(model *util.Model) {
	r.model_stack[len(r.model_stack)-1] = model
}

func (r *Router) activeModelPop() *util.Model {
	model := r.activeModelGet()
	r.model_stack = r.model_stack[:len(r.model_stack)-1]
	return model
}

func (r *Router) activeModelUpdate(msg tea.Msg) tea.Cmd {
	return (*r.activeModelGet()).Update(msg)
}

func (r *Router) activeModelFocus() tea.Cmd {
	cmd, keyMap := (*r.activeModelGet()).Focus()
	return tea.Batch(cmd, util.AnnounceKeyMapCmd(keyMap))
}

func (r *Router) activeModelInit() tea.Cmd {
	return tea.Sequence(
		(*r.activeModelGet()).Init(),
		r.activeModelUpdate(InitMsg{RouterControll: RouterControll{rid: r.id}}),
		r.activeModelUpdate(tea.WindowSizeMsg(r.size)),
		r.activeModelFocus(),
	)
}
