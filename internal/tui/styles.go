package tui

import "github.com/charmbracelet/lipgloss"

var (
	// General
	docStyle = lipgloss.NewStyle().Margin(1, 2)

	// Style for the help text at the bottom
	helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240")) // A muted gray

	// Style for an inactive/disabled item
	inactiveItemStyle = lipgloss.NewStyle().PaddingLeft(2).Strikethrough(true).Foreground(lipgloss.Color("240"))

	// Error style
	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))

	// Titles
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("170")). // A nice purple
			Bold(true)

	// Lists
	itemStyle         = lipgloss.NewStyle().PaddingLeft(2)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(0).Foreground(lipgloss.Color("170"))

	// Modal Dialog
	dialogBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(1, 2).
			Width(50)

	buttonStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("231")).
			Background(lipgloss.Color("57")).
			Padding(0, 3).
			MarginTop(1)

	activeButtonStyle = buttonStyle.Copy().
				Background(lipgloss.Color("170")).
				Underline(true)
)
