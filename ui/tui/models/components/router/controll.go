// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package router

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/ui/tui/util"
)

type Controll struct {
	rid int
}

func (c *Controll) Push(model *util.Model) tea.Cmd {
	return func() tea.Msg { return PushMsg{rid: c.rid, Model: model} }
}
func (c *Controll) Pop(count int) tea.Cmd {
	return func() tea.Msg { return PopMsg{rid: c.rid, Count: count} }
}
func (c *Controll) Change(model *util.Model) tea.Cmd {
	return func() tea.Msg { return ChangeMsg{rid: c.rid, Model: model} }
}
