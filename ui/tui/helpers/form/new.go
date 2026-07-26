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
	form := Form[T]{
		ResetToInitialData: true,
		DefaultRowAlign:    Strech,
	}
	for _, opt := range opts {
		opt(&form)
	}
	form.changeActiveIndex(0)
	return form
}

func WithOnSubmit[T any](fn func(result T, err error) (tea.Cmd, bool)) FormOpt[T] {
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

func WithInitialData[T any](data T) FormOpt[T] {
	return func(form *Form[T]) {
		form.SetInitialData(data)
	}
}

// If the guard returns the provided confirmCmd, the loss of data will be CONFIRMED.
// Return nil to PREVENT data loss.
func WithOnDiscardGuard[T any](guard func(confirmCmd tea.Cmd) tea.Cmd) FormOpt[T] {
	return func(form *Form[T]) {
		form.DiscardGuard = guard
	}
}

func WithResetToInitialData[T any](b bool) FormOpt[T] {
	return func(form *Form[T]) {
		form.ResetToInitialData = b
	}
}

func WithFocusI[T any](i int) FormOpt[T] {
	return func(form *Form[T]) {
		form.activeIndex = i
	}
}

func WithDefaultRowAlign[T any](align RowAlign) FormOpt[T] {
	if align == Default {
		panic("Default is an invalid value for form.DefaultRowAlign")
	}
	return func(form *Form[T]) {
		form.DefaultRowAlign = align
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
		form.items = append(form.items, Item{id, element, nil})
	}
}
