// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package tablecontroll

import (
	"slices"

	"github.com/charmbracelet/bubbles/table"
	"github.com/toeirei/keymaster/util/slicest"
)

type BubblesTableRenderer[T any] func(records []T, width int) ([]table.Column, []table.Row)

// [Controll.RenderBubblesTable] implements [BubblesTableRenderer]
var _ BubblesTableRenderer[any] = Controll[any]{}.RenderBubblesTable

type Column[T any] struct {
	// column title
	Title func() string

	// function to render content of column for each row
	View func(v T) string

	// When using a number 0..1 it will be used as partial max width relative to the totalb available width.
	// When using a number 1..* it will be used as fixed max width.
	MaxWidth float64

	// The order in wich the sizes will be evaluated.
	EvictionOrder int
}

type Columns[T any] []Column[T]

type Controll[T any] struct {
	Columns Columns[T]
}

func New[T any](columns Columns[T]) Controll[T] {
	return Controll[T]{columns}
}

func (c Controll[T]) spacingWidth() int {
	return 2 * len(c.Columns)
}

func (c Controll[T]) RenderRows(records []T) [][]string {
	return slicest.Map(records, func(record T) []string {
		return table.Row(slicest.Map(c.Columns, func(column Column[T]) string {
			return column.View(record)
		}))
	})
}

func (c Controll[T]) PreferredWidth(records []T, availableWidth int) int {
	rows := c.RenderRows(records)
	spacingWidth := c.spacingWidth()
	_, totalWidth := c.ColumnDimensions(rows, availableWidth-spacingWidth)
	return min(availableWidth, totalWidth+spacingWidth)
}

func (c Controll[T]) ColumnDimensions(rows [][]string, availableWidth int) ([]int, int) {
	// columnIndexsByPrio := make(map[int][]int)

	// for i, column := range c.Columns {
	// 	columnIndexsByPrio[column.Prio] = append(columnIndexsByPrio[column.Prio], i)
	// }

	// sortedPrios := slicest.MapKeys(columnIndexsByPrio)
	// slices.Sort(sortedPrios)
	// slices.Reverse(sortedPrios)

	// var totalWidth int
	// columnWidths := make([]int, len(c.Columns))

	// for _, prio := range sortedPrios {
	// 	columnIndexs := columnIndexsByPrio[prio]

	// 	for _, columnIndex := range columnIndexs {
	// 		column := &c.Columns[columnIndex]

	// 		columnWidth := max(
	// 			len(column.Title()),
	// 			slicest.Reduce(rows, func(row []string, w int) int { return max(w, len(row[columnIndex])) }),
	// 		)
	// 		// apply column width modifiers
	// 		if column.MaxWidth > 0 {
	// 			if column.MaxWidth < 1 {
	// 				columnWidth = min(columnWidth, int(column.MaxWidth*float64(availableWidth)))
	// 			} else {
	// 				columnWidth = min(columnWidth, int(column.MaxWidth))
	// 			}
	// 		}
	// 		// save result to slice and total sum
	// 		totalWidth += columnWidth
	// 		columnWidths[columnIndex] = columnWidth
	// 	}
	// }

	var totalWidth int
	columnWidths := slicest.MapI(c.Columns, func(i int, column Column[T]) int {
		// calculate max width of all cells and the header
		columnWidth := max(
			len(column.Title()),
			slicest.Reduce(rows, func(r []string, w int) int { return max(w, len(r[i])) }),
		)
		// apply column width modifiers
		if column.MaxWidth > 0 {
			if column.MaxWidth < 1 {
				columnWidth = min(columnWidth, int(column.MaxWidth*float64(availableWidth)))
			} else {
				columnWidth = min(columnWidth, int(column.MaxWidth))
			}
		}
		// save result to slice and total sum
		totalWidth += columnWidth
		return columnWidth
	})
	return columnWidths, totalWidth
}

func (c Controll[T]) RenderBubblesTable(records []T, width int) ([]table.Column, []table.Row) {
	rows := c.RenderRows(records)

	bubblesRows := slicest.Map(rows, func(row []string) table.Row { return row })

	availableWidth := width - c.spacingWidth()

	columnWidths, totalColumnWidth := c.ColumnDimensions(rows, availableWidth)

	remainingWidth := availableWidth - totalColumnWidth

	if remainingWidth >= 0 {
		// spread space evenly, when more than needed is available
		return slicest.MapI(c.Columns, func(i int, column Column[T]) table.Column {
			extraWidth := remainingWidth / (len(c.Columns) - i)
			remainingWidth -= extraWidth
			return table.Column{
				Title: column.Title(),
				Width: columnWidths[i] + extraWidth,
			}
		}), bubblesRows
	} else {
		// organize column indexes by their prio
		columnIndexsByPrio := make(map[int][]int)
		for i, column := range c.Columns {
			columnIndexsByPrio[column.EvictionOrder] = append(columnIndexsByPrio[column.EvictionOrder], i)
		}

		// retrieve and sort existing prios
		sortedPrios := slicest.MapKeys(columnIndexsByPrio)
		slices.Sort(sortedPrios)

		// size columns in each prio down weighted by their desired width until the columns fit into the provided width
		for _, prio := range sortedPrios {
			columnIndexs := columnIndexsByPrio[prio]

			// get total column width for all columns in this prio
			var totalPrioColumnWidth int
			for _, columnIndex := range columnIndexs {
				totalPrioColumnWidth += columnWidths[columnIndex]
			}

			// get the width needed to be stripped from this prio
			strippedWidth := min(totalPrioColumnWidth, -remainingWidth)
			remainingWidth += strippedWidth

			// strip strippedWidth from columns in this prio evenly
			for _, columnIndex := range columnIndexs {
				columnWidths[columnIndex] = ((totalPrioColumnWidth - strippedWidth) * columnWidths[columnIndex]) / totalPrioColumnWidth
			}

			// break when there is ennough remainingWidth left
			if remainingWidth >= 0 {
				break
			}
		}

		return slicest.MapI(c.Columns, func(i int, column Column[T]) table.Column {
			return table.Column{
				Title: column.Title(),
				Width: columnWidths[i],
			}
		}), bubblesRows

		// old implementation
		// // size columns down weighted by their desired width, when not ennough space is available
		// return slicest.MapI(c.Columns, func(i int, column Column[T]) table.Column {
		// 	return table.Column{
		// 		Title: column.Title(),
		// 		Width: (availableWidth * columnWidths[i]) / totalColumnWidth,
		// 	}
		// }), bubblesRows
	}
}
