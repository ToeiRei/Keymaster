// package tui provides the terminal user interface for Keymaster.
// This file defines the shared lipgloss styles used across the different
// views to ensure a consistent look and feel.
package tui // import "github.com/toeirei/keymaster/internal/tui"

import "github.com/charmbracelet/lipgloss"

// colorPalette defines the core colors used in the TUI.
const (
	colorSubtle    = lipgloss.Color("240") // Muted gray
	colorHighlight = lipgloss.Color("81")  // A nice teal/cyan
	colorSpecial   = lipgloss.Color("208") // An orange for special attention
	colorError     = lipgloss.Color("196") // A bright red
	colorSuccess   = lipgloss.Color("40")  // A nice green
	colorWhite     = lipgloss.Color("231")
)

// Styles defines the reusable lipgloss styles for various UI components.
var (
	// General
	docStyle = lipgloss.NewStyle().Margin(1, 2)

	// Help text
	helpStyle = lipgloss.NewStyle().Foreground(colorSubtle)

	// Inactive/disabled items
	inactiveItemStyle = lipgloss.NewStyle().
				Strikethrough(true).
				Foreground(colorSubtle)

	// Error messages
	errorStyle = lipgloss.NewStyle().Foreground(colorError)

	// Success messages
	successStyle = lipgloss.NewStyle().Foreground(colorSuccess)

	// Special attention messages (e.g., destructive actions)
	specialStyle = lipgloss.NewStyle().Foreground(colorSpecial)

	// Main title on the dashboard
	mainTitleStyle = lipgloss.NewStyle().
			Foreground(colorHighlight).
			Bold(true).
			Padding(1, 3)

	// Titles
	titleStyle = lipgloss.NewStyle().
			Foreground(colorHighlight).
			Bold(true).
			Padding(1, 2)

	// Lists
	itemStyle         = lipgloss.NewStyle()
	selectedItemStyle = lipgloss.NewStyle().Foreground(colorHighlight)

	// Form elements
	formItemStyle         = lipgloss.NewStyle()
	formSelectedItemStyle = lipgloss.NewStyle().Foreground(colorHighlight)

	// Modal Dialogs
	dialogBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.ThickBorder()).
			BorderForeground(colorHighlight).
			Padding(1, 2).
			BorderTop(true).
			BorderLeft(true).
			BorderRight(true).
			BorderBottom(true).
			Width(60)

	// Buttons for modals
	buttonStyle = lipgloss.NewStyle().
			Foreground(colorWhite).
			Background(lipgloss.Color("237")). // Dark gray
			Padding(0, 3).
			MarginTop(1)

	activeButtonStyle = buttonStyle.Copy().
				Background(colorHighlight).
				Foreground(colorWhite).
				Underline(true)

	// Status messages
	statusMessageStyle = lipgloss.NewStyle().
				Padding(0, 1).
				Foreground(colorWhite).
				Background(colorHighlight)
)
