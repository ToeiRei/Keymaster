// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package form

import (
	"github.com/bobg/go-generics/v4/slices"
	tea "github.com/charmbracelet/bubbletea"
)

type FormOpt[T comparable] = func(form *Form[T])

type RowOpt[T comparable] = func(form *Form[T], row *row)

func New[T comparable](opts ...FormOpt[T]) Form[T] {
	form := Form[T]{resetToInitialData: true}
	for _, opt := range opts {
		opt(&form)
	}
	form.changeActiveIndex(0)
	return form
}

func WithOnSubmit[T comparable](fn func(result T, err error) tea.Cmd) FormOpt[T] {
	return func(form *Form[T]) {
		form.OnSubmit = fn
	}
}

func WithOnCancel[T comparable](fn func() tea.Cmd) FormOpt[T] {
	return func(form *Form[T]) {
		form.OnCancel = fn
	}
}

func WithOnReset[T comparable](fn func() tea.Cmd) FormOpt[T] {
	return func(form *Form[T]) {
		form.OnReset = fn
	}
}

func WithResetAfterSubmit[T comparable]() FormOpt[T] {
	return func(form *Form[T]) {
		form.ResetAfterSubmit = true
	}
}

func WithInitialData[T comparable](data T) FormOpt[T] {
	return func(form *Form[T]) {
		form.SetInitialData(data)
	}
}

// If the guard returns a non nil value, the loss of data will be prevented.
// Action will be [ActionReset] or [ActionCancel].
// Send [ConfirmCancelMsg] or [ConfirmResetMsg] msg to confirm the intend.
func WithDiscardGuard[T comparable](guard func(action Action) tea.Cmd) FormOpt[T] {
	return func(form *Form[T]) {
		form.DiscardGuard = guard
	}
}

func WithFocusI[T comparable](i int) FormOpt[T] {
	return func(form *Form[T]) {
		form.activeIndex = i
	}
}

func WithFocus[T comparable](id string) FormOpt[T] {
	return func(form *Form[T]) {
		i := slices.IndexFunc(form.items, func(item Item) bool { return item.Id == id })
		if i >= 0 {
			form.activeIndex = i
		}
	}
}

func WithRowItem[T comparable](id string, element FormElement, opts ...RowOpt[T]) FormOpt[T] {
	_opts := make([]RowOpt[T], 1, len(opts)+1)
	_opts[0] = WithItem[T](id, element)
	_opts = append(_opts, opts...)

	return WithRow(_opts...)
}

func WithRow[T comparable](opts ...RowOpt[T]) FormOpt[T] {
	return func(form *Form[T]) {
		row := row{align: Strech}
		for _, opt := range opts {
			opt(form, &row)
		}
		form.rows = append(form.rows, row)

	}
}

func WithAlign[T comparable](align RowAlign) RowOpt[T] {
	return func(form *Form[T], row *row) { row.align = align }
}

func WithItem[T comparable](id string, element FormElement) RowOpt[T] {
	return func(form *Form[T], row *row) {
		row.items = append(row.items, len(form.items))
		form.items = append(form.items, Item{id, element, nil})
	}
}
