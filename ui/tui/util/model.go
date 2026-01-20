// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package util

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

type Model interface {
	Init() tea.Cmd
	Update(tea.Msg) tea.Cmd
	View() string
	Focusable
}

// polyfill: won't be needed as of go 1.26
func new[T any](v T) *T { return &v }

func ModelPointer[T any, PT interface {
	*T
	Model
}](v PT) *Model {
	return new(Model(v))
}

func BorrowModel[T any, PT interface {
	*T
	Model
}](m *Model) (PT, func()) {
	t := (*m).(PT)
	return t, func() { *m = Model(t) }
}

func BorrowModelFunc[T any, PT interface {
	*T
	Model
}](m *Model, fn func(PT)) {
	t := (*m).(PT)
	fn(t)
	*m = Model(t)
}

func BorrowModelSafe[T Model](m *Model) (*T, func(), error) {
	if t, ok := (*m).(T); ok {
		t := new(t)
		return t, func() { *m = Model(*t) }, nil
	} else {
		return nil, nil, fmt.Errorf("type mismatch inferring model: %T != %T", m, t)
	}
}

func BorrowModelFuncSafe[T Model](m *Model, fn func(*T)) error {
	if t, ok := (*m).(T); ok {
		t := new(t)
		fn(t)
		*m = Model(*t)
		return nil
	} else {
		return fmt.Errorf("type mismatch inferring model: %T != %T", m, t)
	}
}
