// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package form

import (
	"github.com/bobg/go-generics/v4/slices"
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

func WithFocusI[T any](i int) FormOpt[T] {
	return func(form *Form[T]) {
		form.activeIndex = i
	}
}

func WithFocus[T any](id string) FormOpt[T] {
	return func(form *Form[T]) {
		i := slices.IndexFunc(form.items, func(item Item) bool { return item.Id == id })
		if i >= 0 {
			form.activeIndex = i
		}
	}
}

func WithRowItem[T any](id string, element FormElement, opts ...RowOpt[T]) FormOpt[T] {
	_opts := make([]RowOpt[T], 1, len(opts)+1)
	_opts[0] = WithItem[T](id, element)
	_opts = append(_opts, opts...)

	return WithRow(_opts...)
}

// func WithSingleElementRows[T any](opts ...RowOpt[T]) FormOpt[T] {
// 	return func(form *Form[T]) {
// 		for _, item := range items {
// 			WithRow([]Item{item}, opts...)(form)
// 		}
// 	}
// }

func WithRow[T any](opts ...RowOpt[T]) FormOpt[T] {
	return func(form *Form[T]) {
		row := row{}
		for _, opt := range opts {
			opt(form, &row)
		}
		form.rows = append(form.rows, row)

	}
}

func WithAlign[T any](align RowAlign) RowOpt[T] {
	return func(form *Form[T], row *row) { row.align = align }
}

func WithItem[T any](id string, element FormElement) RowOpt[T] {
	return func(form *Form[T], row *row) {
		row.items = append(row.items, len(form.items))
		form.items = append(form.items, Item{id, element})
	}
}
