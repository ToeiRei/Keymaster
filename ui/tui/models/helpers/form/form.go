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

const (
	Left RowAlign = iota
	Right
	Center
	// Strech
)

type RowAlign int


type FormElement interface {
	util.Focusable
	Reset()
	Init() tea.Cmd
	Update(msg tea.Msg) (tea.Cmd, Action)
	Set(any)
	Get() any
	View(width int) string
	Focusable() bool
}

type item struct {
	id      string
	element FormElement
}

type row struct {
	items []int
	align RowAlign
}

type Form[T any] struct {
	OnSubmit         func(result T, err error) tea.Cmd
	OnCancel         func() tea.Cmd
	OnReset          func() tea.Cmd
	ResetAfterSubmit bool

	items       []item
	rows        []row
	activeIndex int
	focused     bool
	baseKeyMap  help.KeyMap
	size        util.Size
}

func (f Form[T]) Init() tea.Cmd {
	return tea.Batch(slicest.Map(f.items, func(item item) tea.Cmd {
		return item.element.Init()
	})...)
}

func (f *Form[T]) Update(msg tea.Msg) tea.Cmd {
	// handle size updates
	if f.size.UpdateFromMsg(msg) {
		return nil
	}

	if f.focused {
		// handle key updates for form
		if kmsg, ok := msg.(tea.KeyMsg); ok {
			switch {
			case key.Matches(kmsg, DefaultKeyMap.Next):
				return f.changeActiveIndex(1)
			case key.Matches(kmsg, DefaultKeyMap.Prev):
				return f.changeActiveIndex(-1)
			}
		}

		// pass msg to active input
		return f.updateActiveInput(msg)
	}

	return nil
}

func (f Form[T]) View() string {
	// TODO refine (this is only a basic implementation)
	return lipgloss.JoinVertical(
		lipgloss.Left,
		slicest.Map(f.rows, func(row row) string {
			return lipgloss.JoinHorizontal(
				lipgloss.Left,
				slicest.Map(row.items, func(item_index int) string {
					return f.items[item_index].element.View(f.size.Width / len(row.items))
				})...,
			)
		})...,
	)
}

// *[Model] implements [util.Focusable]
var _ util.Model = (*Form[any])(nil) // Update with self return

func (f *Form[T]) Focus(baseKeyMap help.KeyMap) tea.Cmd {
	f.focused, f.baseKeyMap = true, baseKeyMap
	return f.items[f.activeIndex].element.Focus(util.MergeKeyMaps(f.baseKeyMap, DefaultKeyMap))
}

func (f *Form[T]) Blur() {
	f.focused, f.baseKeyMap = false, nil
	f.items[f.activeIndex].element.Blur()
}

// *[Model] implements [util.Focusable]
var _ util.Focusable = (*Form[any])(nil)

func (f *Form[T]) Reset() tea.Cmd {
	for _, item := range f.items {
		item.element.Reset()
	}

	var onResetCmd tea.Cmd
	if f.OnReset != nil {
		onResetCmd = f.OnReset()
	}

	return tea.Sequence(
		onResetCmd,
		// maintain index_delta >= 0 when changing active index, so the next and not the previos focusable element will get acitivated!
		f.changeActiveIndex(len(f.items)-f.activeIndex),
	)
}

func (f *Form[T]) Submit() tea.Cmd {
	var onSubmitCmd tea.Cmd
	if f.OnSubmit != nil {
		data, err := f.Get()
		onSubmitCmd = f.OnSubmit(data, err)
	}

	var resetCmd tea.Cmd
	if f.ResetAfterSubmit {
		resetCmd = f.Reset()
	}

	return tea.Sequence(onSubmitCmd, resetCmd)
}

func (f *Form[T]) Cancel() tea.Cmd {
	if f.OnCancel != nil {
		return f.OnCancel()
	}
	return nil
}

func (f *Form[T]) updateActiveInput(msg tea.Msg) tea.Cmd {
	var (
		updateCmd tea.Cmd
		actionCmd tea.Cmd
		action    Action
	)

	updateCmd, action = f.items[f.activeIndex].element.Update(msg)

	switch action {
	case ActionNone:
	case ActionNext:
		actionCmd = f.changeActiveIndex(1)
	case ActionPrev:
		actionCmd = f.changeActiveIndex(-1)
	case ActionSubmit:
		actionCmd = f.Submit()
	case ActionCancel:
		actionCmd = f.Cancel()
	case ActionReset:
		actionCmd = f.Reset()
	}

	return tea.Batch(updateCmd, actionCmd)
}

func (f *Form[T]) changeActiveIndex(index_delta int) tea.Cmd {
	// default to current item
	newActiveIndex := f.activeIndex

	// direction aware search for the next focusable item
	for i := 0; i < len(f.items); i++ {
		var index int
		if index_delta >= 0 {
			index = (f.activeIndex + index_delta + i) % len(f.items)
		} else {
			index = (f.activeIndex + index_delta - i) % len(f.items)
		}
		if index < 0 {
			index += len(f.items)
		}
		if f.items[index].element.Focusable() {
			newActiveIndex = index
			break
		}
	}

	if f.activeIndex == newActiveIndex {
		return nil
	}

	f.items[f.activeIndex].element.Blur()
	f.activeIndex = newActiveIndex

	// if index_delta != 0 {
	// 	oldActiveIndex := f.activeIndex
	// 	f.activeIndex += index_delta

	// 	if f.activeIndex > len(f.items)-1 {
	// 		f.activeIndex = 0
	// 	}
	// 	if f.activeIndex < 0 {
	// 		f.activeIndex = len(f.items) - 1
	// 	}

	// 	f.items[oldActiveIndex].input.Blur()
	// }

	return f.items[f.activeIndex].element.Focus(util.MergeKeyMaps(f.baseKeyMap, DefaultKeyMap))
}

func (f *Form[T]) Get() (T, error) {
	var data T
	values := make(map[string]any, len(f.items))

	for _, item := range f.items {
		values[item.id] = item.element.Get()
	}

	err := decode(values, &data)
	return data, err
}

func (f *Form[T]) Set(data T) error {
	values := make(map[string]any, len(f.items))
	if err := decode(data, &values); err != nil {
		return err
	}

	for i := range f.items {
		if value, ok := values[f.items[i].id]; ok {
			f.items[i].element.Set(value)
		}
	}

	return nil
}

func decode(input any, output any) error {
	config := &mapstructure.DecoderConfig{
		Metadata: nil,
		Result:   output,
		TagName:  "mapstructure,form", // TODO deprecate "mapstructure"
	}

	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return err
	}

	return decoder.Decode(input)
}
