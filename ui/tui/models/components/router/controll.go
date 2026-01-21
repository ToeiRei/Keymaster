// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package router

// TODO rewrite with util.Model in mind

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/ui/tui/util"
)

type RouterControll struct {
	rid int
}

func (rc *RouterControll) Push(model *util.Model) tea.Cmd {
	return func() tea.Msg { return PushMsg{rid: rc.rid, Model: model} }
}
func (rc *RouterControll) Pop(count int) tea.Cmd {
	return func() tea.Msg { return PopMsg{rid: rc.rid, Count: count} }
}
func (rc *RouterControll) Change(model *util.Model) tea.Cmd {
	return func() tea.Msg { return ChangeMsg{rid: rc.rid, Model: model} }
}

// func (rc *RouterControll) IsMsgOwner(msg tea.Msg) bool {
// 	rmsg, ok := msg.(RouterMsg)
// 	return ok && rmsg.routerId() == rc.rid
// }
