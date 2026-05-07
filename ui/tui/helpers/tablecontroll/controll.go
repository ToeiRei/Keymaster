// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package tablecontroll

import (
	"github.com/charmbracelet/bubbles/table"
	"github.com/toeirei/keymaster/util/slicest"
)

type BubblesTableRenderer[T any] func(records []T, width int) ([]table.Column, []table.Row)

// [Controll.RenderBubblesTable] implements [BubblesTableRenderer]
var _ BubblesTableRenderer[any] = Controll[any]{}.RenderBubblesTable

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
	var totalWidth int
	columnWidths := slicest.MapI(c.Columns, func(i int, column Column[T]) int {
		// calculate max width of all cells and the header
		columnWidth := max(
			len(column.Title),
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
				Title: column.Title,
				Width: columnWidths[i] + extraWidth,
			}
		}), bubblesRows
	} else {
		// size columns down weighted by their desired width, when not ennough space is available
		return slicest.MapI(c.Columns, func(i int, column Column[T]) table.Column {
			return table.Column{
				Title: column.Title,
				Width: (availableWidth * columnWidths[i]) / totalColumnWidth,
			}
		}), bubblesRows
	}
}
