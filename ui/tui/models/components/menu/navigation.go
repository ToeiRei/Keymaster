// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package menu

import tea "github.com/charmbracelet/bubbletea"

func (m *Model) up() {
	// get pointer to current active stack index
	index := &m.ActiveStack[len(m.ActiveStack)-1]
	// decrease and rotate if too small
	(*index)--
	if *index < 0 {
		*index = 0
	}
}
func (m *Model) down() {
	// get parents sub items len
	parent_len := m.getParentLen(m.getActiveItemStack())
	// get pointer to current active stack index
	index := &m.ActiveStack[len(m.ActiveStack)-1]
	// increase and rotate if too large
	(*index)++
	if *index > parent_len-1 {
		*index = parent_len - 1
	}
}
func (m *Model) left() {
	if len(m.ActiveStack) > 1 {
		m.ActiveStack = m.ActiveStack[:len(m.ActiveStack)-1]
	}
}
func (m *Model) right() tea.Cmd {
	active_stack := m.getActiveItemStack()
	active_item := active_stack[len(active_stack)-1]
	if len(active_item.SubItems) > 0 {
		m.ActiveStack = append(m.ActiveStack, 0)
		return nil
	} else if active_item.Cmd != nil {
		return active_item.Cmd
	} else {
		return func() tea.Msg { return ItemSelected{Id: active_item.Id} }
	}
}

func (m *Model) getActiveItemStack() []Item {
	var stack []Item
	var cursor []Item = m.Items

	for _, i := range m.ActiveStack {
		item := cursor[i]
		cursor = item.SubItems
		stack = append(stack, item)
	}

	return stack
}

func (m *Model) getParentLen(item_stack []Item) int {
	if len(item_stack) > 1 {
		return len(item_stack[len(item_stack)-2].SubItems)
	} else {
		return len(m.Items)
	}
}
