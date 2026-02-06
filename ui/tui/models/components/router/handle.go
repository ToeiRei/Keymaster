// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package router

import (
	tea "github.com/charmbracelet/bubbletea"
)

// handle PushMsg
func (m *Model) handlePush(msg PushMsg) tea.Cmd {
	// blur recent model
	(*m.activeModelGet()).Blur()
	// push new model
	m.modelStack = append(m.modelStack, msg.Model)
	// initialize pushed model
	return m.activeModelInit()
}

// handle PopMsg
func (m *Model) handlePop(msg PopMsg) tea.Cmd {
	// pop and blur old models
	for range msg.Count {
		if len(m.modelStack) <= 1 {
			break
		}
		(*m.activeModelPop()).Blur()
	}
	// focus active model
	return m.activeModelFocus()
}

// handle ChangeMsg
func (m *Model) handleChange(msg ChangeMsg) tea.Cmd {
	// destroy recent model
	(*m.activeModelGet()).Blur()
	// set new model
	m.activeModelSet(msg.Model)
	// initialize set model
	return m.activeModelInit()
}
