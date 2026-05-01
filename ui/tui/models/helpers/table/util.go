// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package table

import (
	"github.com/charmbracelet/bubbles/table"
	"github.com/toeirei/keymaster/util/slicest"
)

type BubblesTableRenderer[T any] func(records []T, width int) ([]table.Column, []table.Row)

type Column[T any] struct {
	// column title
	Title string

	// function to render content of column for each row
	View func(v T) string

	// When using a number 0..1 it will be used as partial max width relative to the totalb available width.
	// When using a number 1..* it will be used as fixed max width.
	MaxWidth float64
}

type Columns[T any] []Column[T]

func NewBubblesTableRenderer[T any](columns Columns[T]) BubblesTableRenderer[T] {
	return func(records []T, width int) ([]table.Column, []table.Row) {
		bubblesRows := slicest.Map(records, func(record T) table.Row {
			return table.Row(slicest.Map(columns, func(column Column[T]) string {
				return column.View(record)
			}))
		})

		spacingWidth := 2 * len(columns)
		availableWidth := width - spacingWidth

		var totalColumnWidth int
		columnWidths := slicest.MapI(columns, func(i int, column Column[T]) int {
			// calculate max width of all cells and the header
			columnWidth := max(
				len(column.Title),
				slicest.Reduce(bubblesRows, func(r table.Row, w int) int { return max(w, len(r[i])) }),
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
			totalColumnWidth += columnWidth
			return columnWidth
		})

		remainingWidth := availableWidth - totalColumnWidth

		if remainingWidth >= 0 {
			// spread space evenly, when more than needed is available
			return slicest.MapI(columns, func(i int, column Column[T]) table.Column {
				extraWidth := remainingWidth / (len(columns) - i)
				remainingWidth -= extraWidth
				return table.Column{
					Title: column.Title,
					Width: columnWidths[i] + extraWidth,
				}
			}), bubblesRows
		} else {
			// size columns down weighted by their desired width, when not ennough space is available
			return slicest.MapI(columns, func(i int, column Column[T]) table.Column {
				return table.Column{
					Title: column.Title,
					Width: int(float64(availableWidth) * (float64(columnWidths[i]) / float64(totalColumnWidth))),
				}
			}), bubblesRows
		}
	}
}
