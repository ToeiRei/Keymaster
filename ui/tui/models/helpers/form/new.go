// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package form

import (
	tea "github.com/charmbracelet/bubbletea"
)

type FormOpt[T any] = func(form *Form[T])

type RowOpt[T any] = func(form *Form[T], row *row)

func New[T any](opts ...FormOpt[T]) Form[T] {
	form := Form[T]{}
	for _, opt := range opts {
		opt(&form)
	}
	form.changeActiveIndex(0)
	return form
}

func WithOnSubmit[T any](fn func(result T, err error) tea.Cmd) FormOpt[T] {
	return func(form *Form[T]) {
		form.OnSubmit = fn
	}
}

func WithOnCancel[T any](fn func() tea.Cmd) FormOpt[T] {
	return func(form *Form[T]) {
		form.OnCancel = fn
	}
}

func WithOnReset[T any](fn func() tea.Cmd) FormOpt[T] {
	return func(form *Form[T]) {
		form.OnReset = fn
	}
}

func WithResetAfterSubmit[T any]() FormOpt[T] {
	return func(form *Form[T]) {
		form.ResetAfterSubmit = true
	}
}

func WithFocus[T any](i int) FormOpt[T] {
	return func(form *Form[T]) {
		form.activeIndex = i
	}
}

func WithSingleElementRow[T any](id string, element FormElement) FormOpt[T] {
	return WithRow(WithElement[T](id, element))
}

func WithRow[T any](opts ...RowOpt[T]) FormOpt[T] {
	return func(form *Form[T]) {
		row := row{}
		for _, opt := range opts {
			opt(form, &row)
		}
		form.rows = append(form.rows, row)

	}
}

func WithFocusNext[T any]() RowOpt[T] {
	return func(form *Form[T], row *row) {
		form.activeIndex = len(form.items)
	}
}

func WithAlign[T any](align RowAlign) RowOpt[T] {
	return func(form *Form[T], row *row) {
		row.align = align
	}
}

func WithElement[T any](id string, element FormElement) RowOpt[T] {
	return func(form *Form[T], row *row) {
		row.items = append(row.items, len(form.items))
		form.items = append(form.items, item{id: id, element: element})
	}
}
