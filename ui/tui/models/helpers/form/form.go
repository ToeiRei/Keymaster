// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package form

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/go-viper/mapstructure/v2"
	"github.com/toeirei/keymaster/ui/tui/util"
	"github.com/toeirei/keymaster/util/slicest"
)

type FormInput interface {
	util.Focusable
	Reset()
	Init() tea.Cmd
	Update(msg tea.Msg) (tea.Cmd, Action)
	Set(any)
	Get() any
	View(width int) string
}

type formItem struct {
	id    string
	input FormInput
}

type formRow struct {
	items []int
}

type Form[T any] struct {
	OnSubmit         func(result T, err error) tea.Cmd
	OnCancel         func() tea.Cmd
	ResetAfterSubmit bool

	items       []formItem
	rows        []formRow
	activeIndex int
	focused     bool
	baseKeyMap  help.KeyMap
	size        util.Size
}

func (f Form[T]) Init() tea.Cmd {
	return tea.Batch(slicest.Map(f.items, func(item formItem) tea.Cmd {
		return item.input.Init()
	})...)
}

func (f Form[T]) Update(msg tea.Msg) (Form[T], tea.Cmd) {
	// handle size updates
	if f.size.Update(msg) {
		return f, nil
	}

	if f.focused {
		var cmd tea.Cmd

		// handle key updates for form
		if kmsg, ok := msg.(tea.KeyMsg); ok {
			switch {
			case key.Matches(kmsg, DefaultKeyMap.Next):
				cmd = f.changeActiveIndex(1)
				return f, cmd
			case key.Matches(kmsg, DefaultKeyMap.Prev):
				cmd = f.changeActiveIndex(-1)
				return f, cmd
			}
		}

		// pass msg to active input
		cmd = f.updateActiveInput(msg)
		return f, cmd
	}

	return f, nil
}

func (f Form[T]) View() string {
	// TODO refine (this is only a basic implementation)
	return lipgloss.JoinVertical(
		lipgloss.Left,
		// slicest.Map(f.items, func(item formItem) string {
		// 	return item.input.View(f.size.Width)
		// })...,
		slicest.Map(f.rows, func(row formRow) string {
			return lipgloss.JoinHorizontal(
				lipgloss.Center,
				slicest.Map(row.items, func(item_index int) string {
					return f.items[item_index].input.View(f.size.Width / len(row.items))
				})...,
			)
		})...,
	)
}

// *Model implements util.Focusable
// var _ util.Model = (*Form[any])(nil) // Update with self return

func (f *Form[T]) Focus(baseKeyMap help.KeyMap) tea.Cmd {
	f.focused, f.baseKeyMap = true, baseKeyMap
	return f.items[f.activeIndex].input.Focus(util.MergeKeyMaps(f.baseKeyMap, DefaultKeyMap))
}

func (f *Form[T]) Blur() {
	f.focused, f.baseKeyMap = false, nil
	f.items[f.activeIndex].input.Blur()
}

// *Model implements util.Focusable
var _ util.Focusable = (*Form[any])(nil)

func (f *Form[T]) Reset() tea.Cmd {
	for _, item := range f.items {
		item.input.Reset()
	}

	return f.changeActiveIndex(-f.activeIndex)
}

func (f *Form[T]) Submit() tea.Cmd {
	var resetCmd tea.Cmd
	data, err := f.Get()
	if f.ResetAfterSubmit {
		resetCmd = f.Reset()
	}
	return tea.Batch(
		resetCmd,
		f.OnSubmit(data, err),
	)
}

func (f *Form[T]) updateActiveInput(msg tea.Msg) tea.Cmd {
	var (
		updateCmd tea.Cmd
		actionCmd tea.Cmd
		action    Action
	)

	updateCmd, action = f.items[f.activeIndex].input.Update(msg)

	switch action {
	case ActionNone:
	case ActionNext:
		actionCmd = f.changeActiveIndex(1)
	case ActionPrev:
		actionCmd = f.changeActiveIndex(-1)
	case ActionSubmit:
		actionCmd = f.Submit()
	case ActionCancel:
		actionCmd = f.OnCancel()
	}

	return tea.Batch(updateCmd, actionCmd)
}

func (f *Form[T]) changeActiveIndex(index int) tea.Cmd {
	index = index % len(f.items)

	if index != 0 && f.focused {
		oldActiveIndex := f.activeIndex
		f.activeIndex += index

		if f.activeIndex > len(f.items)-1 {
			f.activeIndex = 0
		}
		if f.activeIndex < 0 {
			f.activeIndex = len(f.items) - 1
		}

		f.items[oldActiveIndex].input.Blur()
	}

	return f.items[f.activeIndex].input.Focus(util.MergeKeyMaps(f.baseKeyMap, DefaultKeyMap))
}

func (f *Form[T]) Get() (T, error) {
	var data T
	values := make(map[string]any, len(f.items))

	for _, item := range f.items {
		values[item.id] = item.input.Get()
	}

	err := mapstructure.Decode(values, &data)
	return data, err
}

func (f *Form[T]) Set(data T) error {
	values := make(map[string]any, len(f.items))
	if err := mapstructure.Decode(data, &values); err != nil {
		return err
	}

	for i := range f.items {
		if value, ok := values[f.items[i].id]; ok {
			f.items[i].input.Set(value)
		}
	}

	return nil
}
