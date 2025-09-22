package tui

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/internal/db"
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

	// Style for an inactive/disabled item
	inactiveItemStyle = itemStyle.Strikethrough(true).Foreground(lipgloss.Color("240"))

	// Style for the help text at the bottom
	helpStyle = lipgloss.NewStyle().
			MarginLeft(4).
			Foreground(lipgloss.Color("240")) // A muted gray
)

// viewState represents which part of the UI is currently active.
type viewState int

const (
	menuView viewState = iota
	accountsView
	publicKeysView
	assignKeysView
	rotateKeyView
	deployView
)

// mainModel is the top-level model for the TUI. It manages which view is currently active.
type mainModel struct {
	state      viewState
	menu       menuModel
	deployer   deployModel
	rotator    rotateKeyModel
	assignment assignKeysModel
	keys       publicKeysModel
	accounts   accountsModel
	db         *sql.DB
	err        error
}

// menuModel holds the state for the main menu.
type menuModel struct {
	choices []string // The menu items to show.
	cursor  int      // Which menu item our cursor is pointing at.
}

// initialModel returns the starting state of the TUI.
func initialModel(db *sql.DB) mainModel {
	return mainModel{
		state: menuView,
		menu: menuModel{
			choices: []string{
				"Manage Accounts (user@host)",
				"Manage Public Keys",
				"Assign Keys to Accounts",
				"Rotate System Keys",
				"Deploy to Fleet",
			},
		},
		db: db,
	}
}

// Init is the first function that will be called. It can be used to perform
// some I/O operations on program startup.
func (m mainModel) Init() tea.Cmd {
	// We don't need to do anything on startup, so return nil.
	return nil
}

// Update is called when "things happen." It's where we handle all events,
// like key presses.
func (m mainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Global keybindings that work everywhere.
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}
	}

	// Delegate updates to the currently active view.
	switch m.state {
	case accountsView:
		// If we received a "back" message, switch the state.
		if _, ok := msg.(backToMenuMsg); ok {
			m.state = menuView
			return m, nil
		}
		var newAccountsModel tea.Model
		newAccountsModel, cmd = m.accounts.Update(msg)
		m.accounts = newAccountsModel.(accountsModel)

	case publicKeysView:
		// If we received a "back" message, switch the state.
		if _, ok := msg.(backToMenuMsg); ok {
			m.state = menuView
			return m, nil
		}
		var newKeysModel tea.Model
		newKeysModel, cmd = m.keys.Update(msg)
		m.keys = newKeysModel.(publicKeysModel)

	case assignKeysView:
		// If we received a "back" message, switch the state.
		if _, ok := msg.(backToMenuMsg); ok {
			m.state = menuView
			return m, nil
		}
		var newAssignmentModel tea.Model
		newAssignmentModel, cmd = m.assignment.Update(msg)
		m.assignment = newAssignmentModel.(assignKeysModel)

	case rotateKeyView:
		// If we received a "back" message, switch the state.
		if _, ok := msg.(backToMenuMsg); ok {
			m.state = menuView
			return m, nil
		}
		var newRotatorModel tea.Model
		newRotatorModel, cmd = m.rotator.Update(msg)
		m.rotator = newRotatorModel.(rotateKeyModel)

	case deployView:
		// If we received a "back" message, switch the state.
		if _, ok := msg.(backToMenuMsg); ok {
			m.state = menuView
			return m, nil
		}
		var newDeployerModel tea.Model
		newDeployerModel, cmd = m.deployer.Update(msg)
		m.deployer = newDeployerModel.(deployModel)
	default: // menuView
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "q":
				return m, tea.Quit
			case "up", "k":
				if m.menu.cursor > 0 {
					m.menu.cursor--
				}
			case "down", "j":
				if m.menu.cursor < len(m.menu.choices)-1 {
					m.menu.cursor++
				}
			case "enter":
				switch m.menu.cursor {
				case 0: // Manage Accounts
					m.state = accountsView
					m.accounts = newAccountsModel() // This loads data from the DB.
					return m, nil
				case 1: // Manage Public Keys
					m.state = publicKeysView
					m.keys = newPublicKeysModel()
					return m, nil
				case 2: // Assign Keys to Accounts
					m.state = assignKeysView
					m.assignment = newAssignKeysModel()
					return m, nil
				case 3: // Rotate System Keys
					m.state = rotateKeyView
					m.rotator = newRotateKeyModel()
					return m, nil
				case 4: // Deploy to Fleet
					m.state = deployView
					m.deployer = newDeployModel()
					return m, nil
				default:
					// For now, other options just quit.
					return m, tea.Quit
				}
			}
		}
	}

	return m, cmd
}

// View is called to render the UI. It's a string that gets printed to the
// terminal.
func (m mainModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	// Delegate rendering to the active view.
	switch m.state {
	case accountsView:
		return m.accounts.View()
	case publicKeysView:
		return m.keys.View()
	case assignKeysView:
		return m.assignment.View()
	case deployView:
		return m.deployer.View()
	case rotateKeyView:
		return m.rotator.View()
	default: // menuView
		return m.menu.View()
	}
}

func (m menuModel) View() string {
	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("🔑 Keymaster TUI"))
	b.WriteString("\n\n")

	for i, choice := range m.choices {
		if m.cursor == i {
			b.WriteString(selectedItemStyle.Render("» " + choice))
		} else {
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
	// Initialize the database. A file named keymaster.db will be created if it doesn't exist.
	database, err := db.InitDB("./keymaster.db")
	if err != nil {
		fmt.Printf("Error initializing database: %v\n", err)
		os.Exit(1)
	}

	// Create a new Bubble Tea program.
	p := tea.NewProgram(initialModel(database))

	// Run the program.
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
