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

type Form[T any] struct {
	form form.Form[T]
}

func Open[T any](form form.Form[T]) tea.Cmd {
	return popup.Open(util.ModelPointer(New(form)))
}

// New hosts f in a popup. A hosted form has to dismiss the popup once it
// resolves, so its OnSubmit and OnCancel are wrapped to close first — the
// callbacks themselves only decide what to hand back, and need not close.
// A submit the callback rejects as invalid leaves the popup open.
//
// Buttons carrying their own [form.WithButtonAction] are not covered: they
// report [form.ActionNone], so the form never resolves and they still have to
// close for themselves (see choicepopup).
func New[T any](f form.Form[T]) *Form[T] {
	onSubmit, onCancel := f.OnSubmit, f.OnCancel

	f.OnSubmit = func(result T, err error) (tea.Cmd, bool) {
		var cmd tea.Cmd
		valid := true
		if onSubmit != nil {
			cmd, valid = onSubmit(result, err)
		}
		if !valid {
			return cmd, false
		}
		return tea.Sequence(popup.Close(), cmd), true
	}

	f.OnCancel = func() tea.Cmd {
		var cmd tea.Cmd
		if onCancel != nil {
			cmd = onCancel()
		}
		return tea.Sequence(popup.Close(), cmd)
	}

	return &Form[T]{f}
}

func (m *Form[T]) Init() tea.Cmd { return m.form.Init() }

func (m *Form[T]) Update(msg tea.Msg) tea.Cmd {
	return m.form.Update(msg)
}

func (m Form[T]) View() string { return m.form.ViewLazy() }

func (m *Form[T]) Focus(km help.KeyMap) tea.Cmd { return m.form.Focus(km) }

func (m *Form[T]) Blur() { m.form.Blur() }

// *[Form] implements [util.Model]
var _ util.Model = (*Form[any])(nil)
