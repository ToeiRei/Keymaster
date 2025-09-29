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
	"github.com/spf13/viper"
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
	auditView
	tagsView
	bootstrapView
	languageView
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
	auditor    auditModel
	rotator    *rotateKeyModel
	assignment *assignKeysModel
	keys       *publicKeysModel
	accounts   *accountsModel
	auditLog   *auditLogModel
	tags       tagsViewModel
	bootstrap  *bootstrapModel
	language   languageModel
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

// languageModel holds the state for the language selection menu.
type languageModel struct {
	choices     map[string]string // map of lang code to display name
	orderedKeys []string          // for stable iteration
	cursor      int
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
				i18n.T("menu.audit_hosts"),
				i18n.T("menu.view_accounts_by_tag"),
				i18n.T("menu.language"),
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

	case bootstrapRequestedMsg:
		// Start bootstrap workflow
		m.state = bootstrapView
		m.bootstrap = newBootstrapModel(msg.username, msg.hostname, msg.label, msg.tags)
		return m, m.bootstrap.Init()
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
		// The Update method for accounts now has a pointer receiver, so we expect a pointer back.
		if newModel, ok := newAccountsModel.(*accountsModel); ok {
			m.accounts = newModel
		}

	case publicKeysView:
		// If we received a "back" message, switch the state.
		if _, ok := msg.(backToMenuMsg); ok {
			m.state = menuView
			return m, refreshDashboardCmd()
		}
		var newKeysModel tea.Model
		newKeysModel, cmd = m.keys.Update(msg)
		if newModel, ok := newKeysModel.(*publicKeysModel); ok {
			m.keys = newModel
		}

	case assignKeysView:
		// If we received a "back" message, switch the state.
		if _, ok := msg.(backToMenuMsg); ok {
			m.state = menuView
			return m, refreshDashboardCmd()
		}
		var newModel tea.Model
		newModel, cmd = m.assignment.Update(msg)
		// The Update method for assignment now has a pointer receiver, so we expect a pointer back.
		// We can directly assign the result of Update to m.assignment.
		m.assignment = newModel.(*assignKeysModel)

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

	case auditView:
		if _, ok := msg.(backToMenuMsg); ok {
			m.state = menuView
			return m, refreshDashboardCmd()
		}
		var newAuditModel tea.Model
		newAuditModel, cmd = m.auditor.Update(msg)
		m.auditor = newAuditModel.(auditModel)

	case tagsView:
		if _, ok := msg.(backToMenuMsg); ok {
			m.state = menuView
			return m, refreshDashboardCmd()
		}
		var newTagsModel tea.Model
		newTagsModel, cmd = m.tags.Update(msg)
		m.tags = newTagsModel.(tagsViewModel)

	case bootstrapView:
		// Handle back message or account completion
		if _, ok := msg.(backToListMsg); ok {
			m.state = accountsView
			// Reinitialize accounts view to show updated list
			newModel := newAccountsModel()
			m.accounts = &newModel
			var updatedModel tea.Model
			updatedModel, cmd = m.accounts.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
			m.accounts = updatedModel.(*accountsModel)
			return m, cmd
		}
		// Handle successful account creation from bootstrap
		if accountMsg, ok := msg.(accountModifiedMsg); ok {
			m.state = accountsView
			// Reinitialize accounts view to show new account
			newModel := newAccountsModel()
			m.accounts = &newModel
			var updatedModel tea.Model
			updatedModel, cmd = m.accounts.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
			m.accounts = updatedModel.(*accountsModel)
			// Also pass the account modified message to accounts view
			updatedModel, cmd2 := m.accounts.Update(accountMsg)
			m.accounts = updatedModel.(*accountsModel)
			return m, tea.Batch(cmd, cmd2)
		}
		var newBootstrapModel tea.Model
		newBootstrapModel, cmd = m.bootstrap.Update(msg)
		m.bootstrap = newBootstrapModel.(*bootstrapModel)

	case languageView:
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case "q", "esc":
				m.state = menuView
				return m, refreshDashboardCmd()
			case "up", "k":
				if m.language.cursor > 0 {
					m.language.cursor--
				}
			case "down", "j":
				if m.language.cursor < len(m.language.orderedKeys)-1 {
					m.language.cursor++
				}
			case "enter":
				langCode := m.language.orderedKeys[m.language.cursor]
				i18n.SetLang(langCode)
				viper.Set("language", langCode)
				// We can ignore the error here as it's not critical for the session.
				_ = viper.WriteConfig()

				// Re-initialize the main menu with new translations
				m.menu = initialModel().menu

				// Go back to main menu
				m.state = menuView
				return m, refreshDashboardCmd()
			}
		}
		var newLangModel tea.Model
		newLangModel, cmd = m.language.Update(msg)
		m.language = newLangModel.(languageModel)

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
					// newAccountsModel returns a value, but we need a pointer.
					newModel := newAccountsModel()
					m.accounts = &newModel
					// Manually update the new sub-model with the current window size
					// to ensure the viewport is initialized correctly.
					var updatedModel tea.Model
					updatedModel, cmd = m.accounts.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
					m.accounts = updatedModel.(*accountsModel)
					return m, cmd
				case 1: // Manage Public Keys
					m.state = publicKeysView
					newModel := newPublicKeysModel()
					m.keys = &newModel
					var updatedModel tea.Model
					updatedModel, cmd = m.keys.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
					m.keys = updatedModel.(*publicKeysModel)
					return m, cmd
				case 2: // Assign Keys to Accounts
					m.state = assignKeysView
					newModel := newAssignKeysModel()
					m.assignment = newModel
					var updatedModel tea.Model
					updatedModel, cmd = m.assignment.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
					m.assignment = updatedModel.(*assignKeysModel)
					return m, cmd
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
				case 6: // Audit Hosts
					m.state = auditView
					m.auditor = newAuditModel()
					return m, nil
				case 7: // View Accounts by Tag
					m.state = tagsView
					m.tags = newTagsViewModel()
					return m, nil
				case 8: // Language
					m.state = languageView
					m.language = newLanguageModel()
					return m, nil
				}
			case "L":
				// "L" now opens the language menu
				m.state = languageView
				m.language = newLanguageModel()
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
	case auditView:
		return m.auditor.View()
	case tagsView:
		return m.tags.View()
	case bootstrapView:
		return m.bootstrap.View()
	case languageView:
		return m.language.View()
	default: // menuView
		return m.menu.View(m.dashboard, m.width, m.height)
	}
}

// View renders the main menu and dashboard.
func (m menuModel) View(data dashboardData, width, height int) string {
	// Title (i18n)
	title := mainTitleStyle.Render("🔑 " + i18n.T("dashboard.title"))
	subTitle := helpStyle.Render(i18n.T("dashboard.subtitle"))
	header := lipgloss.JoinVertical(lipgloss.Left, title, subTitle)

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
		sysKeyStatus = successStyle.Render(i18n.T("dashboard.system_key.active", data.systemKeySerial))
	}
	dashboardItems = append(dashboardItems, lipgloss.JoinVertical(lipgloss.Left,
		i18n.T("dashboard.accounts", data.accountCount, data.activeAccountCount),
		i18n.T("dashboard.public_keys", data.publicKeyCount, data.globalKeyCount),
		i18n.T("dashboard.system_key", sysKeyStatus),
	))

	// Recent Activity
	dashboardItems = append(dashboardItems, "", "", paneTitleStyle.Render(i18n.T("dashboard.recent_activity")), "")

	// --- Layout ---
	paneStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorSubtle).
		Padding(1, 2)

	// Calculate height for the panes to fill the screen
	headerHeight := lipgloss.Height(header)
	footerHeight := lipgloss.Height(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Background(lipgloss.Color("236")).Padding(0, 1).Italic(true).Render(""))
	paneHeight := height - headerHeight - footerHeight - 2 // -2 for newlines around mainArea

	menuWidth := 38
	dashboardWidth := width - 4 - menuWidth - 2

	if len(data.recentLogs) == 0 {
		dashboardItems = append(dashboardItems, helpStyle.Render(i18n.T("dashboard.no_recent_activity")))
	} else {
		for _, log := range data.recentLogs {
			// --- Refactored log line rendering with color and better truncation ---
			ts := log.Timestamp[5:16] // Format as MM-DD HH:MM

			// Calculate available space inside the pane for the log line content.
			innerDashboardWidth := dashboardWidth - 4 - 2
			availableWidth := innerDashboardWidth - len(ts) - 1 // Subtract timestamp and a space

			// Get the styled action from audit_log.go's logic and its plain-text length.
			actionStyle := auditActionStyle(log.Action)
			styledAction := actionStyle.Render(log.Action)
			actionLen := len(log.Action)

			// Calculate the remaining space for the details string.
			detailsWidth := availableWidth - actionLen - 1 // -1 for space after action
			if detailsWidth < 10 {
				detailsWidth = 10 // Ensure we show at least a little detail.
			}

			// Gracefully truncate the details if they are too long.
			details := log.Details
			if len(details) > detailsWidth {
				details = details[:detailsWidth-3] + "..."
			}

			// Use lipgloss.JoinHorizontal to correctly lay out the styled and unstyled parts.
			logLine := lipgloss.JoinHorizontal(lipgloss.Left,
				helpStyle.Render(ts), " ", styledAction, " ", helpStyle.Render(details))

			dashboardItems = append(dashboardItems, logLine)
		}
	}
	dashboardContent := lipgloss.JoinVertical(lipgloss.Left, dashboardItems...)

	leftPane := paneStyle.Width(menuWidth).Height(paneHeight).Render(menuContent)
	rightPane := paneStyle.Width(dashboardWidth).Height(paneHeight).MarginLeft(2).Render(dashboardContent)

	mainArea := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	// Styled footer/help line
	footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Background(lipgloss.Color("236")).Padding(0, 1).Italic(true)
	footer := footerStyle.Render(i18n.T("dashboard.footer"))

	return lipgloss.JoinVertical(lipgloss.Top, header, mainArea, footer)
}

// newLanguageModel creates a new model for the language selection view.
func newLanguageModel() languageModel {
	// The keys are the language codes (e.g., "en"), and values are the display names.
	choices := map[string]string{
		"en":      "English",
		"de":      "Deutsch",
		"en-olde": "Ænglisc (Olde English)",
	}
	// We need a stable order for the cursor.
	keys := []string{"en", "de", "en-olde"}

	return languageModel{
		choices:     choices,
		orderedKeys: keys,
		cursor:      0,
	}
}

// Init for languageModel.
func (m languageModel) Init() tea.Cmd { return nil }

// Update for languageModel.
func (m languageModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

// View for languageModel.
func (m languageModel) View() string {
	title := mainTitleStyle.Render("🌐 " + i18n.T("menu.language"))

	var listItems []string
	listItems = append(listItems, titleStyle.Render(i18n.T("language.select")), "")

	for i, langCode := range m.orderedKeys {
		displayName := m.choices[langCode]
		line := "  " + displayName
		if m.cursor == i {
			line = "▸ " + displayName
			listItems = append(listItems, selectedItemStyle.Render(line))
		} else {
			listItems = append(listItems, itemStyle.Render(line))
		}
	}

	paneStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorSubtle).Padding(1, 2)
	listPane := paneStyle.Width(60).Render(lipgloss.JoinVertical(lipgloss.Left, listItems...))

	footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Background(lipgloss.Color("236")).Padding(0, 1).Italic(true)
	helpLine := footerStyle.Render(i18n.T("language.help"))

	return lipgloss.JoinVertical(lipgloss.Left, title, "", listPane, "", helpLine)
}

// Run is the main entrypoint for the TUI. It initializes and runs the Bubble Tea program.
func Run() {
	// Initialize i18n with the language from config
	i18n.Init(viper.GetString("language"))

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
