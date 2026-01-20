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
	stack.focussedIndex = util.Clamp(Focus(-1), stack.focussedIndex, Focus(len(stack.items)-1))
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

func WithFocus(focus Focus) NewOpt {
	return func(stack *Model) {
		stack.focussedIndex = focus
	}
}
