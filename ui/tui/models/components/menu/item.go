package menu

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/util/slicest"
)

func WithItem(id string, name string, sub_items ...Item) Item {
	return Item{
		Id:       id,
		Name:     name,
		SubItems: sub_items,
	}
}

type Item struct {
	Id       string
	Name     string
	SubItems []Item
	Cmd      tea.Cmd
}

func (i Item) View(is_active bool, active_stack []int) string {
	content := i.Name

	item_style := lipgloss.NewStyle()
	if len(i.SubItems) > 0 {
		item_style = item_style.
			Underline(true).
			Italic(true)
	}
	if is_active {
		if len(active_stack) > 0 {
			item_style = item_style.Foreground(lipgloss.Color("#8655B1"))
		} else {
			// item_style = item_style.UnsetForeground(lipgloss.Color("#8655B1"))
			item_style = item_style.Foreground(lipgloss.Color("#000000"))
			item_style = item_style.Background(lipgloss.Color("#8655B1"))
		}
	}

	content = item_style.Render(content)

	// add sub items when active
	if is_active && len(i.SubItems) > 0 && len(active_stack) > 0 {
		style := lipgloss.
			NewStyle().
			BorderLeft(true).
			BorderStyle(lipgloss.NormalBorder()).
			PaddingLeft(1)
		content = lipgloss.JoinVertical(lipgloss.Left,
			content,
			style.Render(renderItems(i.SubItems, active_stack)),
		)
	}

	return content
}

type ItemSelected struct {
	Id string
}

func renderItems(items []Item, active_stack []int) string {
	active_i := -1
	if len(active_stack) > 0 {
		active_i, active_stack = active_stack[0], active_stack[1:]
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		slicest.MapI(items, func(i int, item Item) string {
			return item.View(active_i == i, active_stack)
		})...,
	)
}
