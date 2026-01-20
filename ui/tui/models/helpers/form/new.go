// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package form

import (
	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
)

type NewOpt[T any] = func(form *Form[T])

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

func WithOnCancel2(fn func() tea.Cmd) NewOpt[any] {
	return func(form *Form[any]) {
		form.OnCancel = fn
	}
}

func WithResetAfterSubmit[T any]() NewOpt[T] {
	return func(form *Form[T]) {
		form.ResetAfterSubmit = true
	}
}

func WithKeyMap[T any](keyMap help.KeyMap) NewOpt[T] {
	return func(form *Form[T]) {
		form.BaseKeyMap = keyMap
	}
}

func WithInput[T any](id string, input FormInput) NewOpt[T] {
	return func(form *Form[T]) {
		form.items = append(form.items, formItem{
			id:    id,
			input: input,
		})
	}
}

// func With[T any]() NewOpt[T] {
// 	return func(form *Form[T]) {
// 		form. =
// 	}
// }
