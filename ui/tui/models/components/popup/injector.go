// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package popup

import (
	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/ui/tui/util"
)

type popup struct {
	model   *util.Model
	onClose func(*util.Model) tea.Cmd
}

type Injector struct {
	child  *util.Model
	popups []popup
	size   util.Size
}

func NewInjector(child *util.Model) *Injector {
	return &Injector{
		child: child,
	}
}

func (m Injector) Init() tea.Cmd {
	return (*m.child).Init()
}

func (m *Injector) Update(msg tea.Msg) tea.Cmd {
	m.size.Update(msg)

	switch msg := msg.(type) {
	case openMsg:
		return m.open(popup{
			model:   msg.Model,
			onClose: msg.OnClose,
		})
	case closeMsg:
		return m.close()
	}

	return (*m.activeModel()).Update(msg)
}

func (m Injector) View() string {
	// maybe use when rendering is delegated to the popups model
	return (*m.activeModel()).View()

	// if len(m.popups) > 0 {
	// 	return lipgloss.
	// 		NewStyle().
	// 		Render(lipgloss.Place(
	// 			m.size.Width, m.size.Height,
	// 			lipgloss.Center, lipgloss.Center,
	// 			(*m.popups[len(m.popups)-1].model).View(),
	// 			lipgloss.WithWhitespaceChars("."),
	// 		))
	// }
	// return (*m.child).View()
}

func (m *Injector) Focus() (tea.Cmd, help.KeyMap) {
	return (*m.activeModel()).Focus()
}
func (m *Injector) Blur() {
	(*m.activeModel()).Blur()
}

// *Model implements util.Model
var _ util.Model = (*Injector)(nil)

func (m *Injector) open(popup popup) tea.Cmd {
	// blur active view
	m.blurActiveModel()
	// append new popup
	m.popups = append(m.popups, popup)
	// init and focus new popup
	return tea.Batch(
		(*popup.model).Init(),
		m.focusActiveModel(),
		(*m.activeModel()).Update(m.size.ToMsg()),
	)
}

func (m *Injector) close() tea.Cmd {
	// popup left to close?
	if len(m.popups) > 0 {
		// blur old popup
		m.blurActiveModel()
		// pop old popup
		var onCloseCmd tea.Cmd
		if popup := m.popups[len(m.popups)-1]; popup.onClose != nil {
			onCloseCmd = popup.onClose(popup.model)
		}
		m.popups = m.popups[:len(m.popups)-1]
		// focus underlying view
		return tea.Batch(
			m.focusActiveModel(),
			onCloseCmd,
		)
	}
	return nil
}

func (m *Injector) activeModel() *util.Model {
	if len(m.popups) > 0 {
		return m.popups[len(m.popups)-1].model
	} else {
		return m.child
	}
}
func (m *Injector) focusActiveModel() tea.Cmd {
	cmd, keyMap := m.Focus()
	return tea.Batch(cmd, util.AnnounceKeyMapCmd(keyMap))
}
func (m *Injector) blurActiveModel() {
	m.Blur()
}
