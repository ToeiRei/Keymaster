package tui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// --- STYLING ---
var (
	// The main title style
	titleStyle = lipgloss.NewStyle().
			MarginLeft(2).
			Foreground(lipgloss.Color("170")). // A nice purple
			Bold(true)

	// Style for a regular menu item
	itemStyle = lipgloss.NewStyle().PaddingLeft(4)

	// Style for the selected menu item
	selectedItemStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				Foreground(lipgloss.Color("170"))

	// Style for the help text at the bottom
	helpStyle = lipgloss.NewStyle().
			MarginLeft(4).
			Foreground(lipgloss.Color("240")) // A muted gray
)

// model holds the state of our TUI.
type model struct {
	choices []string // The menu items to show.
	cursor  int      // Which menu item our cursor is pointing at.
}

// initialModel returns the starting state of the TUI.
func initialModel() model {
	return model{
		// Updated menu items based on your feedback
		choices: []string{
			"Manage Accounts (user@host)",
			"Manage Public Keys",
			"Assign Keys to Accounts",
			"Rotate System Keys",
			"Deploy to Fleet",
		},
	}
}

// Init is the first function that will be called. It can be used to perform
// some I/O operations on program startup.
func (m model) Init() tea.Cmd {
	// We don't need to do anything on startup, so return nil.
	return nil
}

// Update is called when "things happen." It's where we handle all events,
// like key presses.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// The message is a key press.
	case tea.KeyMsg:

		// What was the key pressed?
		switch msg.String() {

		// These keys should exit the program.
		case "ctrl+c", "q":
			return m, tea.Quit

		// The "up" and "k" keys move the cursor up.
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		// The "down" and "j" keys move the cursor down.
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}

		// The "enter" key will select the item.
		// For now, it just quits. We'll add functionality later.
		case "enter":
			// TODO: Handle selection by navigating to a new "view" or "model".
			return m, tea.Quit
		}
	}

	// Return the updated model to the Bubble Tea runtime for processing.
	return m, nil
}

// View is called to render the UI. It's a string that gets printed to the
// terminal.
func (m model) View() string {
	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("🔑 Keymaster TUI"))
	b.WriteString("\n\n")

	// Iterate over our choices.
	for i, choice := range m.choices {
		// Is the cursor pointing at this choice?
		if m.cursor == i {
			// Render the selected row
			b.WriteString(selectedItemStyle.Render("» " + choice))
		} else {
			// Render a regular row
			b.WriteString(itemStyle.Render(choice))
		}
		b.WriteString("\n")
	}

	// The footer.
	b.WriteString(helpStyle.Render("\n(j/k or up/down to navigate, enter to select, q to quit)"))

	// Send the UI for rendering.
	return b.String()
}

// Run is the entrypoint for the TUI.
func Run() {
	// Create a new Bubble Tea program.
	p := tea.NewProgram(initialModel())

	// Run the program.
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
