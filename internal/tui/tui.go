// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// package tui provides the terminal user interface for Keymaster.
// This file, tui.go, is the main entry point for the TUI, containing the
// top-level model that acts as a router to all other sub-views.
package tui // import "github.com/toeirei/keymaster/internal/tui"

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/i18n"
	"github.com/toeirei/keymaster/internal/model"

	tea "github.com/charmbracelet/bubbletea"
)

// viewState represents which part of the UI is currently active.
type viewState int

const (
	// menuView is the main dashboard and navigation menu.
	menuView viewState = iota
	accountsView
	publicKeysView
	assignKeysView
	rotateKeyView
	deployView
	auditLogView
	tagsView
)

// dashboardDataMsg is a message containing the data for the main menu dashboard.
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

// mainModel is the top-level model for the TUI. It acts as a state machine
// and router, delegating updates and view rendering to the currently active sub-model.
type mainModel struct {
	state      viewState
	menu       menuModel
	deployer   deployModel
	rotator    *rotateKeyModel
	assignment assignKeysModel
	keys       publicKeysModel
	accounts   accountsModel
	auditLog   *auditLogModel
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

// initialModel creates the starting state of the TUI, beginning at the main menu.
func initialModel() mainModel {
	return mainModel{
		state: menuView,
		menu: menuModel{
			choices: []string{
				i18n.T("menu.manage_accounts"),
				i18n.T("menu.manage_public_keys"),
				i18n.T("menu.assign_keys"),
				i18n.T("menu.rotate_system_keys"),
				i18n.T("menu.deploy_to_fleet"),
				i18n.T("menu.view_audit_log"),
				i18n.T("menu.view_accounts_by_tag"),
			},
		},
	}
}

// Init is the first function that will be called by the Bubble Tea runtime.
// It kicks off the initial command to load data for the dashboard.
func (m mainModel) Init() tea.Cmd {
	return refreshDashboardCmd()
}

// Update is the main message loop. It handles all events (like key presses and
// window size changes) and delegates them to the active sub-model.
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
		m.rotator = newRotatorModel.(*rotateKeyModel)

	case deployView:
		// If we received a "back" message, switch the state.
		if _, ok := msg.(backToMenuMsg); ok {
			m.state = menuView
			return m, refreshDashboardCmd()
		}
		var newDeployModel tea.Model
		newDeployModel, cmd = m.deployer.Update(msg)
		m.deployer = newDeployModel.(deployModel)

	case auditLogView:
		if _, ok := msg.(backToMenuMsg); ok {
			m.state = menuView
			return m, refreshDashboardCmd()
		}
		var newAuditLogModel tea.Model
		newAuditLogModel, cmd = m.auditLog.Update(msg)
		m.auditLog = newAuditLogModel.(*auditLogModel)

	case tagsView:
		if _, ok := msg.(backToMenuMsg); ok {
			m.state = menuView
			return m, refreshDashboardCmd()
		}
		var newTagsModel tea.Model
		newTagsModel, cmd = m.tags.Update(msg)
		m.tags = newTagsModel.(tagsViewModel)

	default: // menuView
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
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
					m.auditLog = newAuditLogModel.(*auditLogModel)
					return m, cmd
				case 6: // View Accounts by Tag
					m.state = tagsView
					m.tags = newTagsViewModel()
					return m, nil
				default:
					// For now, other options just quit.
					return m, tea.Quit
				}
			case "L":
				// Toggle language between English and German
				if i18n.T("menu.manage_accounts") == "Manage Accounts" {
					i18n.SetLang("de")
				} else {
					i18n.SetLang("en")
				}
				m.menu.choices = []string{
					i18n.T("menu.manage_accounts"),
					i18n.T("menu.manage_public_keys"),
					i18n.T("menu.assign_keys"),
					i18n.T("menu.rotate_system_keys"),
					i18n.T("menu.deploy_to_fleet"),
					i18n.T("menu.view_audit_log"),
					i18n.T("menu.view_accounts_by_tag"),
				}
				return m, nil
			}
		}
	}

	return m, cmd
}

// View renders the TUI. It's called after every Update and delegates rendering
// to the currently active sub-model.
func (m mainModel) View() string {
	if m.err != nil {
		// A simple error view
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Padding(1, 2)
		return errorStyle.Render(fmt.Sprintf("An error occurred: %v", m.err))
	}

	// Delegate rendering to the currently active view.
	switch m.state {
	case accountsView:
		return m.accounts.View()
	case publicKeysView:
		return m.keys.View()
	case assignKeysView:
		return m.assignment.View()
	case rotateKeyView:
		return m.rotator.View()
	case deployView:
		return m.deployer.View()
	case auditLogView:
		return m.auditLog.View()
	case tagsView:
		return m.tags.View()
	default: // menuView
		return m.menu.View(m.dashboard, m.width)
	}
}

// View renders the main menu and dashboard.
func (m menuModel) View(data dashboardData, width int) string {
	// Title (i18n)
	title := mainTitleStyle.Render("🔑 " + i18n.T("dashboard.title"))
	subTitle := helpStyle.Render(i18n.T("dashboard.subtitle"))
	titleBlock := lipgloss.JoinVertical(lipgloss.Center, title, subTitle)
	header := lipgloss.NewStyle().Align(lipgloss.Center).Render(titleBlock)

	// --- Panes ---
	paneTitleStyle := lipgloss.NewStyle().Bold(true)

	// Menu List (Left Pane)
	var menuItems []string
	menuItems = append(menuItems, paneTitleStyle.Render(i18n.T("menu.navigation")), "") // Add title and a blank line for spacing
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
	dashboardItems = append(dashboardItems, paneTitleStyle.Render(i18n.T("dashboard.system_status")), "")

	// Status Items
	sysKeyStatus := errorStyle.Render(i18n.T("dashboard.system_key.not_generated"))
	if data.systemKeySerial > 0 {
		sysKeyStatus = successStyle.Render(fmt.Sprintf(i18n.T("dashboard.system_key.active"), data.systemKeySerial))
	}
	dashboardItems = append(dashboardItems, lipgloss.JoinVertical(lipgloss.Left,
		fmt.Sprintf(i18n.T("dashboard.accounts"), data.accountCount, data.activeAccountCount),
		fmt.Sprintf(i18n.T("dashboard.public_keys"), data.publicKeyCount, data.globalKeyCount),
		fmt.Sprintf(i18n.T("dashboard.system_key"), sysKeyStatus),
	))

	// Recent Activity
	dashboardItems = append(dashboardItems, "", "", paneTitleStyle.Render(i18n.T("dashboard.recent_activity")), "")

	// --- Layout ---
	paneStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorSubtle).
		Padding(1, 2)

	menuWidth := 38
	dashboardWidth := width - 4 - menuWidth - 2

	if len(data.recentLogs) == 0 {
		dashboardItems = append(dashboardItems, helpStyle.Render(i18n.T("dashboard.no_recent_activity")))
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
			innerDashboardWidth := dashboardWidth - 4 - 2
			maxDetailsLen := innerDashboardWidth - len(ts) - maxActionLen - 2
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

	leftPane := paneStyle.Width(menuWidth).Render(menuContent)
	rightPane := paneStyle.Width(dashboardWidth).MarginLeft(2).Render(dashboardContent)

	mainArea := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	// Styled footer/help line
	footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Background(lipgloss.Color("236")).Padding(0, 1).Italic(true)
	footer := footerStyle.Render(i18n.T("dashboard.footer"))

	return lipgloss.JoinVertical(lipgloss.Left, header, "\n", mainArea, "\n", footer)
}

// Run is the main entrypoint for the TUI. It initializes and runs the Bubble Tea program.
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
