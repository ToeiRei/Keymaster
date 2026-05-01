// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package popupviews

import (
	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/form"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/popup"
	"github.com/toeirei/keymaster/ui/tui/util"
)

type FormModel[T comparable] struct {
	form form.Form[T]
}

func OpenForm[T comparable](form form.Form[T]) tea.Cmd {
	return popup.Open(util.ModelPointer(newForm(form)))
}

func newForm[T comparable](form form.Form[T]) *FormModel[T] {
	return &FormModel[T]{form}
}

func (m FormModel[T]) Init() tea.Cmd { return m.form.Init() }

func (m *FormModel[T]) Update(msg tea.Msg) tea.Cmd {
	return m.form.Update(msg)
}

func (m FormModel[T]) View() string { return m.form.ViewLazy() }

func (m *FormModel[T]) Focus(km help.KeyMap) tea.Cmd { return m.form.Focus(km) }

func (m *FormModel[T]) Blur() { m.form.Blur() }

// *[FormModel] implements [util.Model]
var _ util.Model = (*FormModel[any])(nil)
