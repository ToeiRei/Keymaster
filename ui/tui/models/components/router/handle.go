// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package router

import (
	tea "github.com/charmbracelet/bubbletea"
)

// handle PushMsg
func (r *Router) handlePush(msg PushMsg) tea.Cmd {
	// blur recent model
	(*r.activeModelGet()).Blur()
	// push new model
	r.model_stack = append(r.model_stack, msg.Model)
	// initialize pushed model
	return r.activeModelInit()
}

// handle PopMsg
func (r *Router) handlePop(msg PopMsg) tea.Cmd {
	// pop and blur old models
	for range msg.Count {
		if len(r.model_stack) <= 1 {
			break
		}
		(*r.activeModelPop()).Blur()
	}
	// focus active model
	return r.activeModelFocus()
}

// handle ChangeMsg
func (r *Router) handleChange(msg ChangeMsg) tea.Cmd {
	// destroy recent model
	(*r.activeModelGet()).Blur()
	// set new model
	r.activeModelSet(msg.Model)
	// initialize set model
	return r.activeModelInit()
}
