// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package popupviews

import (
	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/form"
	"github.com/toeirei/keymaster/ui/tui/util"
)

type FormModel[T any] struct {
	form      form.Form[T]
	innerSize util.Size
	size      util.Size
}

func NewForm[T any](form form.Form[T], width, height int) *FormModel[T] {
	return &FormModel[T]{
		form: form,
		innerSize: util.Size{
			Width:  width,
			Height: height,
		},
	}
}

func (m FormModel[T]) Init() tea.Cmd {
	return m.form.Init()
}

func (m *FormModel[T]) Update(msg tea.Msg) tea.Cmd {
	if m.size.UpdateFromMsg(msg) {
		size := util.Size{
			Width:  min(m.innerSize.Width, m.size.Width),
			Height: min(m.innerSize.Height, m.size.Height),
		}
		return m.form.Update(size.ToMsg())
	}
	return m.form.Update(msg)
}

func (m FormModel[T]) View() string {
	// TODO only for testing... size of form needs to be made non greedy
	return lipgloss.NewStyle().MaxWidth(40).Render(m.form.View())
	// return m.form.View()
}

func (m *FormModel[T]) Focus(baseKeyMap help.KeyMap) tea.Cmd {
	return m.form.Focus(baseKeyMap)
}
func (m *FormModel[T]) Blur() {
	m.form.Blur()
}

// *FormModel implements util.Model
var _ util.Model = (*FormModel[any])(nil)
