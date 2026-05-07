// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package formpopup

import (
	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/ui/tui/helpers/form"
	"github.com/toeirei/keymaster/ui/tui/helpers/popup"
	"github.com/toeirei/keymaster/ui/tui/util"
)

type Form[T comparable] struct {
	form form.Form[T]
}

func Open[T comparable](form form.Form[T]) tea.Cmd {
	return popup.Open(util.ModelPointer(New(form)))
}

func New[T comparable](form form.Form[T]) *Form[T] {
	return &Form[T]{form}
}

func (m Form[T]) Init() tea.Cmd { return m.form.Init() }

func (m *Form[T]) Update(msg tea.Msg) tea.Cmd {
	return m.form.Update(msg)
}

func (m Form[T]) View() string { return m.form.ViewLazy() }

func (m *Form[T]) Focus(km help.KeyMap) tea.Cmd { return m.form.Focus(km) }

func (m *Form[T]) Blur() { m.form.Blur() }

// *[Form] implements [util.Model]
var _ util.Model = (*Form[any])(nil)
