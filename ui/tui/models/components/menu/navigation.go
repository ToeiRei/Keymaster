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
	parentLen := m.getParentLen(m.getActiveItemStack())
	// get pointer to current active stack index
	index := &m.ActiveStack[len(m.ActiveStack)-1]
	// increase and rotate if too large
	(*index)++
	if *index > parentLen-1 {
		*index = parentLen - 1
	}
}
func (m *Model) left() {
	if len(m.ActiveStack) > 1 {
		m.ActiveStack = m.ActiveStack[:len(m.ActiveStack)-1]
	}
}
func (m *Model) right() tea.Cmd {
	activeStack := m.getActiveItemStack()
	activeItem := activeStack[len(activeStack)-1]
	if len(activeItem.SubItems) > 0 {
		m.ActiveStack = append(m.ActiveStack, 0)
		return nil
	} else if activeItem.Cmd != nil {
		return activeItem.Cmd
	} else {
		return func() tea.Msg { return ItemSelected{Id: activeItem.Id} }
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

func (m *Model) getParentLen(itemStack []Item) int {
	if len(itemStack) > 1 {
		return len(itemStack[len(itemStack)-2].SubItems)
	} else {
		return len(m.Items)
	}
}
