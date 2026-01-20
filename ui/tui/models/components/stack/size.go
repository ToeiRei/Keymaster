// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package stack

import (
	"math"
	slices "slices"

	// "github.com/bobg/go-generics/v4/slices"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/ui/tui/util"
	"github.com/toeirei/keymaster/util/slicest"
)

type SizeConfig interface {
	Priority() int
	Caltulate(model util.Model, remaining_size int, total_size int) int
}
type staticSize struct {
	Size int
}
type variableSize struct {
	Weight      int
	totalWeight int
}

func StaticSize(size int) SizeConfig     { return &staticSize{Size: size} }
func VariableSize(weigth int) SizeConfig { return &variableSize{Weight: weigth} }

func (sc *staticSize) Priority() int {
	return 0
}
func (sc *variableSize) Priority() int {
	return math.MaxInt
}

func (sc *staticSize) Caltulate(_ util.Model, _ int, _ int) int {
	return sc.Size
}
func (sc *variableSize) Caltulate(_ util.Model, remaining_size int, _ int) int {
	if sc.totalWeight == 0 {
		return remaining_size
	}
	// weird formula to avoid precision loss and floor result using int
	// original easy to read formula: size = remaining_size * (ms.Weight / max_weight)
	return (remaining_size * sc.Weight) / sc.totalWeight
}

func (s *Model) calculateItemSizes() {
	var total_size int
	if s.Orientation == Horizontal {
		total_size = s.size.Width
	} else {
		total_size = s.size.Height
	}

	// track remaining size
	remaining_size := total_size - (s.Gap * (len(s.items) - 1))

	// create pointer slice for later mutation
	sortedItems := make([]*Item, len(s.items))
	for i := 0; i < len(s.items); i++ {
		sortedItems[i] = &s.items[i]
	}

	// sorts item pointers (inplace slice mutation)
	slices.SortFunc(sortedItems, func(item1, item2 *Item) int {
		return item1.SizeConfig.Priority() - item2.SizeConfig.Priority()
	})

	// get total weight from variable SizeConfigs
	total_weight := slicest.Reduce(s.items, func(item Item, total int) int {
		if sizeConfigV, ok := item.SizeConfig.(*variableSize); ok {
			return total + sizeConfigV.Weight
		}
		return total
	})

	// calculate sizes
	for _, item := range sortedItems {
		sizeConfigV, ok := item.SizeConfig.(*variableSize)
		if ok {
			sizeConfigV.totalWeight = total_weight
		}

		size := min(item.SizeConfig.Caltulate(*item.Model, remaining_size, total_size), remaining_size)

		if ok {
			total_weight -= sizeConfigV.Weight
		}

		remaining_size -= size
		item.old_size = item.size
		item.size = size
	}
}

func (s *Model) updateResizedItems(force bool) []tea.Cmd {
	var cmds []tea.Cmd
	for _, item := range s.items {
		if force || item.size != item.old_size {
			var msg tea.WindowSizeMsg
			if s.Orientation == Horizontal {
				msg.Width = item.size
				msg.Height = s.size.Height
			} else {
				msg.Width = s.size.Width
				msg.Height = item.size
			}

			cmds = append(cmds, (*item.Model).Update(msg))
		}
	}
	return cmds
}
