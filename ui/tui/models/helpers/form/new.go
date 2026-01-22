// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package form

import (
	tea "github.com/charmbracelet/bubbletea"
)

type NewOpt[T any] = func(form *Form[T])

// type RowOpt[T any] = func(form *Form[T])

func New[T any](opts ...NewOpt[T]) Form[T] {
	form := Form[T]{}
	for _, opt := range opts {
		opt(&form)
	}
	return form
}

func WithOnSubmit[T any](fn func(result T, err error) tea.Cmd) NewOpt[T] {
	return func(form *Form[T]) {
		form.OnSubmit = fn
	}
}

func WithOnCancel[T any](fn func() tea.Cmd) NewOpt[T] {
	return func(form *Form[T]) {
		form.OnCancel = fn
	}
}

func WithResetAfterSubmit[T any]() NewOpt[T] {
	return func(form *Form[T]) {
		form.ResetAfterSubmit = true
	}
}

func WithInput[T any](id string, input FormInput) NewOpt[T] {
	return func(form *Form[T]) {
		form.rows = append(form.rows, formRow{items: []int{len(form.items)}})
		form.items = append(form.items, formItem{id: id, input: input})
	}
}

func WithInputInline[T any](id string, input FormInput) NewOpt[T] {
	return func(form *Form[T]) {
		if len(form.rows) > 0 {
			form.rows[len(form.rows)-1].items = append(form.rows[len(form.rows)-1].items, len(form.items))
		} else {
			form.rows = append(form.rows, formRow{items: []int{len(form.items)}})
		}
		form.items = append(form.items, formItem{id: id, input: input})
	}
}
