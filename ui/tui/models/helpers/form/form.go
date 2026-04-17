// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package form

import (
	"strings"

	"github.com/bobg/go-generics/v4/slices"
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
	Strech
	SpaceBetween
)

type RowAlign int

var rowAlignments = map[RowAlign]lipgloss.Position{
	Left:   lipgloss.Left,
	Right:  lipgloss.Right,
	Center: lipgloss.Center,
}

type FormElement interface {
	util.Focusable
	Reset()
	Init() (tea.Cmd, GlobalKeyMap)
	Update(msg tea.Msg) (tea.Cmd, Action)
	Set(any)
	Get() any
	View(width int, eager bool) string
	Focusable() bool
}

type Item struct {
	Id           string
	Element      FormElement
	globalKeyMap GlobalKeyMap
}

type row struct {
	items []int
	align RowAlign
}

type Form[T comparable] struct {
	OnSubmit           func(result T, err error) tea.Cmd
	OnCancel           func() tea.Cmd
	OnReset            func() tea.Cmd
	DiscardGuard       func(confirmCmd tea.Cmd) tea.Cmd
	InitialData        T
	ResetAfterSubmit   bool
	ResetToInitialData bool

	items        []Item
	rows         []row
	activeIndex  int
	parentKeyMap help.KeyMap
	focused      bool
	size         util.Size
}

func (f Form[T]) Init() tea.Cmd {
	_ = f.Set(f.InitialData)

	return tea.Batch(slicest.MapI(f.items, func(i int, item Item) tea.Cmd {
		var cmd tea.Cmd
		cmd, f.items[i].globalKeyMap = item.Element.Init()
		return cmd
	})...)
}

func (f *Form[T]) Update(msg tea.Msg) tea.Cmd {
	// handle size updates
	if f.size.UpdateFromMsg(msg) {
		return nil
	}

	if f.focused {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			// handle key updates for form
			switch {
			case key.Matches(msg, DefaultKeyMap.Next):
				return f.changeActiveIndex(1)
			case key.Matches(msg, DefaultKeyMap.Prev):
				return f.changeActiveIndex(-1)
			}

			// handle global keymaps
			for i, item := range f.items {
				if key.Matches(msg, item.globalKeyMap...) {
					return f.updateElement(i, msg)
				}
			}

			// pass remaining key msg to active input
			return f.updateElement(f.activeIndex, msg)
		case confirmDiscardMsg:
			switch msg.action {
			case ActionCancel:
				return f.Cancel(true)
			case ActionReset:
				return f.Reset(true)
			}
		}

		// pass msg to active input
		return f.updateElement(f.activeIndex, msg)
	}

	return nil
}

func (f Form[T]) View() string {
	if len(f.rows) <= 0 {
		return ""
	}

	// render rows
	rowViews := slicest.MapI(f.rows, func(rowIndex int, row row) string {
		var views []string

		switch row.align {
		case Left, Right, Center, SpaceBetween:
			views = f.viewRow(rowIndex, false)
		case Strech:
			views = f.viewRow(rowIndex, true)
		}

		switch row.align {
		case Left, Right, Center:
			return lipgloss.PlaceHorizontal(
				f.size.Width,
				rowAlignments[row.align],
				lipgloss.JoinHorizontal(lipgloss.Top, views...),
			)
		case Strech, SpaceBetween:
			viewsWidth := lipgloss.Width(lipgloss.JoinHorizontal(lipgloss.Top, views...))
			remainingWidth := max(0, f.size.Width-viewsWidth)
			spacedViews := make([]string, 0, (len(views)*2)-1)
			for i, view := range views {
				spacedViews = append(spacedViews, view)
				if i < len(views)-1 {
					spaceWidth := remainingWidth / (len(views) - i - 1)
					remainingWidth -= spaceWidth
					spacedViews = append(spacedViews, strings.Repeat(" ", spaceWidth))
				}
			}
			return lipgloss.JoinHorizontal(lipgloss.Top, spacedViews...)
		}
		return ""
	})

	// find active row, its offset and height
	activeRowIndex := max(0, slices.IndexFunc(f.rows, func(row row) bool {
		return slices.Contains(row.items, f.activeIndex)
	}))
	var activeRowOffset int
	for i := range activeRowIndex {
		activeRowOffset += lipgloss.Height(rowViews[i])
	}
	activeRowHeight := lipgloss.Height(rowViews[activeRowIndex])

	// combine rendered rows and render in limited viewport
	return util.RenderContentInViewportAligned(
		lipgloss.JoinVertical(lipgloss.Left, rowViews...),
		f.size.Height,
		activeRowOffset,
		activeRowHeight,
		lipgloss.Center,
	)
}

// *[Model] implements [util.Model]
var _ util.Model = (*Form[any])(nil)

func (f *Form[T]) Focus(parentKeyMap help.KeyMap) tea.Cmd {
	f.focused, f.parentKeyMap = true, parentKeyMap
	return f.items[f.activeIndex].Element.Focus(f.keymap())
}

func (f *Form[T]) Blur() {
	f.focused, f.parentKeyMap = false, nil
	f.items[f.activeIndex].Element.Blur()
}

// *[Model] implements [util.Focusable]
var _ util.Focusable = (*Form[any])(nil)

func (f *Form[T]) viewRow(rowIndex int, eager bool) []string {
	availableWidth := f.size.Width
	return slicest.MapI(f.rows[rowIndex].items, func(i int, itemIndex int) string {
		width := availableWidth / (len(f.rows[rowIndex].items) - i)
		availableWidth -= width
		return f.items[itemIndex].Element.View(width, eager)
	})
}

func (f *Form[T]) keymap() help.KeyMap {
	return util.MergeKeyMaps(
		f.parentKeyMap,
		DefaultKeyMap,
		slicest.Reduce(f.items, func(item Item, km GlobalKeyMap) GlobalKeyMap {
			return append(km, item.globalKeyMap...)
		}),
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
		resetCmd = f.Reset(true)
	}

	return tea.Sequence(onSubmitCmd, resetCmd)
}

func (f *Form[T]) Cancel(force bool) tea.Cmd {
	if !force {
		if cmd := f.guardUnsavedChanges(ActionCancel); cmd != nil {
			return cmd
		}
	}

	if f.OnCancel != nil {
		return f.OnCancel()
	}
	return nil
}

func (f *Form[T]) Reset(force bool) tea.Cmd {
	if !force {
		if cmd := f.guardUnsavedChanges(ActionReset); cmd != nil {
			return cmd
		}
	}

	for _, item := range f.items {
		item.Element.Reset()
	}

	if f.ResetToInitialData {
		_ = f.Set(f.InitialData)
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

func (f *Form[T]) guardUnsavedChanges(action Action) tea.Cmd {
	if data, _ := f.Get(); data != f.InitialData && f.DiscardGuard != nil {
		return f.DiscardGuard(func() tea.Msg { return confirmDiscardMsg{action} })
	}
	return nil
}

func (f *Form[T]) updateElement(index int, msg tea.Msg) tea.Cmd {
	var actionCmd tea.Cmd

	updateCmd, action := f.items[index].Element.Update(msg)

	switch action {
	case ActionNone:
	case ActionNext:
		actionCmd = f.changeActiveIndex(1)
	case ActionPrev:
		actionCmd = f.changeActiveIndex(-1)
	case ActionSubmit:
		actionCmd = f.Submit()
	case ActionCancel:
		actionCmd = f.Cancel(false)
	case ActionReset:
		actionCmd = f.Reset(false)
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
		if f.items[index].Element.Focusable() {
			newActiveIndex = index
			break
		}
	}

	if f.activeIndex == newActiveIndex {
		return nil
	}

	f.items[f.activeIndex].Element.Blur()
	f.activeIndex = newActiveIndex

	return f.items[f.activeIndex].Element.Focus(f.keymap())
}

func (f *Form[T]) Get() (T, error) {
	var data T
	values := make(map[string]any, len(f.items))

	for _, item := range f.items {
		values[item.Id] = item.Element.Get()
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
		if value, ok := values[f.items[i].Id]; ok {
			f.items[i].Element.Set(value)
		}
	}

	return nil
}

func (f *Form[T]) SetInitialData(data T) { f.InitialData = data }

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
