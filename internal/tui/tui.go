package tui

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/model"

	tea "github.com/charmbracelet/bubbletea"
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
	auditLogView
	tagsView
)

// A message containing the data for the main menu dashboard.
type dashboardDataMsg struct {
	data dashboardData
}

// dashboardData holds the summary information for the main menu view.
type dashboardData struct {
	accountCount       int
	activeAccountCount int
	publicKeyCount     int
	globalKeyCount     int
	systemKeySerial    int // 0 if none
	recentLogs         []model.AuditLogEntry
	err                error
}

// mainModel is the top-level model for the TUI. It manages which view is currently active.
type mainModel struct {
	state      viewState
	menu       menuModel
	deployer   deployModel
	rotator    rotateKeyModel
	assignment assignKeysModel
	keys       publicKeysModel
	accounts   accountsModel
	auditLog   auditLogModel
	tags       tagsViewModel
	dashboard  dashboardData
	width      int
	height     int
	err        error
}

// menuModel holds the state for the main menu.
type menuModel struct {
	choices []string // The menu items to show.
	cursor  int      // Which menu item our cursor is pointing at.
}

// initialModel returns the starting state of the TUI.
func initialModel() mainModel {
	return mainModel{
		state: menuView,
		menu: menuModel{
			choices: []string{
				"Manage Accounts",
				"Manage Public Keys",
				"Assign Keys to Accounts",
				"Rotate System Keys",
				"Deploy to Fleet",
				"View Audit Log",
				"View Accounts by Tag",
			},
		},
	}
}

// Init is the first function that will be called. It can be used to perform
// some I/O operations on program startup.
func (m mainModel) Init() tea.Cmd {
	return refreshDashboardCmd()
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
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case dashboardDataMsg:
		m.dashboard = msg.data
		if msg.data.err != nil {
			m.err = msg.data.err
		}
		return m, nil
	}

	// Delegate updates to the currently active view.
	switch m.state {
	case accountsView:
		// If we received a "back" message, switch the state.
		if _, ok := msg.(backToMenuMsg); ok {
			m.state = menuView
			return m, refreshDashboardCmd()
		}
		var newAccountsModel tea.Model
		newAccountsModel, cmd = m.accounts.Update(msg)
		m.accounts = newAccountsModel.(accountsModel)

	case publicKeysView:
		// If we received a "back" message, switch the state.
		if _, ok := msg.(backToMenuMsg); ok {
			m.state = menuView
			return m, refreshDashboardCmd()
		}
		var newKeysModel tea.Model
		newKeysModel, cmd = m.keys.Update(msg)
		m.keys = newKeysModel.(publicKeysModel)

	case assignKeysView:
		// If we received a "back" message, switch the state.
		if _, ok := msg.(backToMenuMsg); ok {
			m.state = menuView
			return m, refreshDashboardCmd()
		}
		var newAssignmentModel tea.Model
		newAssignmentModel, cmd = m.assignment.Update(msg)
		m.assignment = newAssignmentModel.(assignKeysModel)

	case rotateKeyView:
		// If we received a "back" message, switch the state.
		if _, ok := msg.(backToMenuMsg); ok {
			m.state = menuView
			return m, refreshDashboardCmd()
		}
		var newRotatorModel tea.Model
		newRotatorModel, cmd = m.rotator.Update(msg)
		m.rotator = newRotatorModel.(rotateKeyModel)

	case deployView:
		// If we received a "back" message, switch the state.
		if _, ok := msg.(backToMenuMsg); ok {
			m.state = menuView
			return m, refreshDashboardCmd()
		}
		var newDeployerModel tea.Model
		newDeployerModel, cmd = m.deployer.Update(msg)
		m.deployer = newDeployerModel.(deployModel)

	case auditLogView:
		// If we received a "back" message, switch the state.
		if _, ok := msg.(backToMenuMsg); ok {
			m.state = menuView
			return m, refreshDashboardCmd()
		}
		var newAuditLogModel tea.Model
		newAuditLogModel, cmd = m.auditLog.Update(msg)
		m.auditLog = newAuditLogModel.(auditLogModel)

	case tagsView:
		if _, ok := msg.(backToMenuMsg); ok {
			m.state = menuView
			return m, refreshDashboardCmd()
		}
		var newTagsModel tea.Model
		newTagsModel, cmd = m.tags.Update(msg)
		m.tags = newTagsModel.(tagsViewModel)

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
				case 5: // View Audit Log
					m.state = auditLogView
					m.auditLog = newAuditLogModel()
					// Manually update the new sub-model with the current window size
					// to ensure the viewport is initialized correctly.
					var newAuditLogModel tea.Model
					newAuditLogModel, cmd = m.auditLog.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
					m.auditLog = newAuditLogModel.(auditLogModel)
					return m, cmd
				case 6: // View Accounts by Tag
					m.state = tagsView
					m.tags = newTagsViewModel()
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

	var view string
	// Delegate rendering to the active view.
	switch m.state {
	case accountsView:
		view = m.accounts.View()
	case publicKeysView:
		view = m.keys.View()
	case assignKeysView:
		view = m.assignment.View()
	case deployView:
		view = m.deployer.View()
	case rotateKeyView:
		view = m.rotator.View()
	case auditLogView:
		view = m.auditLog.View()
	case tagsView:
		view = m.tags.View()
	default: // menuView
		view = m.menu.View(m.dashboard, m.width)
	}
	return docStyle.Render(view)
}

func (m menuModel) View(data dashboardData, width int) string {
	// Title
	title := mainTitleStyle.Render("🔑 Keymaster")
	subTitle := helpStyle.Render("An agentless SSH key manager that just does the job.")
	titleBlock := lipgloss.JoinVertical(lipgloss.Center, title, subTitle)
	header := lipgloss.NewStyle().Align(lipgloss.Center).Render(titleBlock)

	// --- Panes ---
	paneTitleStyle := lipgloss.NewStyle().Bold(true)

	// Menu List (Left Pane)
	var menuItems []string
	menuItems = append(menuItems, paneTitleStyle.Render("Navigation"), "") // Add title and a blank line for spacing
	for i, choice := range m.choices {
		if m.cursor == i {
			menuItems = append(menuItems, selectedItemStyle.Render("▸ "+choice))
		} else {
			menuItems = append(menuItems, itemStyle.Render("  "+choice))
		}
	}
	menuContent := lipgloss.JoinVertical(lipgloss.Left, menuItems...)

	// Dashboard (Right Pane)
	var dashboardItems []string
	dashboardItems = append(dashboardItems, paneTitleStyle.Render("System Status"), "")

	// Status Items
	sysKeyStatus := errorStyle.Render("Not Generated")
	if data.systemKeySerial > 0 {
		sysKeyStatus = successStyle.Render(fmt.Sprintf("Active (Serial #%d)", data.systemKeySerial))
	}
	dashboardItems = append(dashboardItems, lipgloss.JoinVertical(lipgloss.Left,
		fmt.Sprintf("Managed Accounts: %d (%d active)", data.accountCount, data.activeAccountCount),
		fmt.Sprintf("     Public Keys: %d (%d global)", data.publicKeyCount, data.globalKeyCount),
		fmt.Sprintf("      System Key: %s", sysKeyStatus),
	))

	// Recent Activity
	dashboardItems = append(dashboardItems, "", "", paneTitleStyle.Render("Recent Activity"), "")

	// --- Layout ---
	paneStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorSubtle).
		Padding(1, 2)

	menuWidth := 38
	// available width = total width - docstyle margins - gap between panes
	dashboardWidth := width - 4 - menuWidth - 2

	if len(data.recentLogs) == 0 {
		dashboardItems = append(dashboardItems, helpStyle.Render("No recent activity."))
	} else {
		for _, log := range data.recentLogs {
			ts := log.Timestamp
			ts = ts[5:16] // MM-DD HH:MM

			action := log.Action
			maxActionLen := 15
			if len(action) > maxActionLen {
				action = action[:maxActionLen]
			}

			details := log.Details
			// Truncate details based on available width in the pane
			// Inner width = dashboardWidth - padding - border
			innerDashboardWidth := dashboardWidth - 4 - 2
			maxDetailsLen := innerDashboardWidth - len(ts) - maxActionLen - 2 // -2 for spaces
			if maxDetailsLen < 5 {
				maxDetailsLen = 5
			}
			if len(details) > maxDetailsLen {
				details = details[:maxDetailsLen-3] + "..."
			}

			logLine := fmt.Sprintf("%s %-15s %s", helpStyle.Render(ts), action, details)
			dashboardItems = append(dashboardItems, logLine)
		}
	}
	dashboardContent := lipgloss.JoinVertical(lipgloss.Left, dashboardItems...)

	leftPane := paneStyle.Copy().Width(menuWidth).Render(menuContent)
	rightPane := paneStyle.Copy().Width(dashboardWidth).MarginLeft(2).Render(dashboardContent)

	mainArea := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	footer := helpStyle.Render("j/k up/down: navigate  enter: select  q: quit")

	return lipgloss.JoinVertical(lipgloss.Left, header, "\n", mainArea, "\n", footer)
}

// Run is the entrypoint for the TUI.
func Run() {
	if _, err := tea.NewProgram(initialModel()).Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}

// refreshDashboardCmd is a tea.Cmd that fetches summary data for the main menu.
func refreshDashboardCmd() tea.Cmd {
	return func() tea.Msg {
		accounts, err := db.GetAllAccounts()
		if err != nil {
			return dashboardDataMsg{data: dashboardData{err: err}}
		}

		keys, err := db.GetAllPublicKeys()
		if err != nil {
			return dashboardDataMsg{data: dashboardData{err: err}}
		}

		sysKey, err := db.GetActiveSystemKey()
		if err != nil {
			return dashboardDataMsg{data: dashboardData{err: err}}
		}

		logs, err := db.GetAllAuditLogEntries()
		if err != nil {
			return dashboardDataMsg{data: dashboardData{err: err}}
		}

		// Process data
		data := dashboardData{}
		data.accountCount = len(accounts)
		for _, acc := range accounts {
			if acc.IsActive {
				data.activeAccountCount++
			}
		}

		data.publicKeyCount = len(keys)
		for _, key := range keys {
			if key.IsGlobal {
				data.globalKeyCount++
			}
		}

		if sysKey != nil {
			data.systemKeySerial = sysKey.Serial
		}

		const maxLogs = 5
		if len(logs) > maxLogs {
			data.recentLogs = logs[:maxLogs]
		} else {
			data.recentLogs = logs
		}

		return dashboardDataMsg{data: data}
	}
}
