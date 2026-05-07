// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package stack

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/ui/tui/util"
)

type NewOpt = func(stack *Model)

func New(opts ...NewOpt) *Model {
	stack := Model{
		Orientation: Horizontal,
		Align:       lipgloss.Top,
	}
	for _, opt := range opts {
		opt(&stack)
	}
	stack.focussedIndex = util.Clamp(0, stack.focussedIndex, len(stack.items)-1)
	return &stack
}

func WithOrientation(orientation Orientation) NewOpt {
	return func(stack *Model) {
		stack.Orientation = orientation
	}
}

func WithAlign(align lipgloss.Position) NewOpt {
	return func(stack *Model) {
		stack.Align = align
	}
}

func WithGap(gap int) NewOpt {
	return func(stack *Model) {
		stack.Gap = gap
	}
}

// WARNING not implemented yet (does nothing when used)
func WithBorder(border lipgloss.Border, sides ...bool) NewOpt {
	return func(stack *Model) {
		stack.Border = border
		stack.BorderSides = sides
	}
}

// WARNING not implemented yet (does nothing when used)
func WithPadding(i ...int) NewOpt {
	return func(stack *Model) {
		stack.Padding = i
	}
}

// WARNING not implemented yet (does nothing when used)
func WithMargin(i ...int) NewOpt {
	return func(stack *Model) {
		stack.Margin = i
	}
}

func WithItem(model *util.Model, sizeConfig SizeConfig, msgFilters ...MsgFilter) NewOpt {
	return func(stack *Model) {
		stack.items = append(stack.items, Item{
			Model:      model,
			SizeConfig: sizeConfig,
			MsgFilters: msgFilters,
		})
	}
}

func WithMsgFilter(msgFilter MsgFilter) NewOpt {
	return func(stack *Model) {
		stack.MsgFilters = append(stack.MsgFilters, msgFilter)
	}
}

func WithFocus(i int) NewOpt {
	return func(stack *Model) {
		stack.focussedIndex = i
	}
}

func WithFocusNext() NewOpt {
	return func(stack *Model) {
		stack.focussedIndex = len(stack.items)
	}
}
