// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// package tui provides the terminal user interface for Keymaster.
// This file contains the logic for the deployment view, which allows users
// to deploy keys to single accounts, tags, or the entire fleet.
package tui // import "github.com/toeirei/keymaster/internal/tui"

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/deploy"
	"github.com/toeirei/keymaster/internal/i18n"
	"github.com/toeirei/keymaster/internal/model"
)

// deployState represents the current view within the deployment workflow.
type deployState int

const (
	deployStateMenu deployState = iota
	deployStateSelectAccount
	deployStateSelectTag
	deployStateShowAuthorizedKeys
	deployStateFleetInProgress
	deployStateInProgress
	deployStateComplete
)

// deployAction differentiates between different actions that can be taken
// on a selected account.
type deployAction int

const (
	actionGetKeys deployAction = iota
	actionDeploySingle
)

// deploymentResultMsg is a message to signal deployment is complete for one account.
type deploymentResultMsg struct {
	account model.Account
	err     error
}

// deployModel represents the state of the deployment view.
// It manages menus, account selection, and the status of deployment operations.
type deployModel struct {
	state              deployState
	action             deployAction
	menuCursor         int
	accountCursor      int
	tagCursor          int
	accounts           []model.Account
	accountsInFleet    []model.Account // Keep order for display
	fleetResults       map[int]error   // map account ID to error for quick lookup
	selectedAccount    model.Account
	tags               []string
	authorizedKeys     string // The generated authorized_keys content
	status             string
	err                error
	accountFilter      string
	isFilteringAccount bool
}

// newDeployModel creates a new model for the deployment view.
func newDeployModel() deployModel {
	return deployModel{
		state:        deployStateMenu,
		fleetResults: make(map[int]error),
	}
}

// Init initializes the deploy model.
func (m deployModel) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the deploy model's state.
func (m deployModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.state {
	case deployStateMenu:
		return m.updateMenu(msg)
	case deployStateSelectAccount:
		return m.updateAccountSelection(msg)
	case deployStateSelectTag:
		return m.updateSelectTag(msg)
	case deployStateShowAuthorizedKeys:
		return m.updateShowAuthorizedKeys(msg)
	case deployStateFleetInProgress:
		if res, ok := msg.(deploymentResultMsg); ok {
			m.fleetResults[res.account.ID] = res.err
			if len(m.fleetResults) == len(m.accountsInFleet) {
				m.state = deployStateComplete
				m.status = i18n.T("deploy.fleet_complete")
			}
		}
		// No other input handled while fleet deployment is running
		return m, nil
	case deployStateInProgress:
		if res, ok := msg.(deploymentResultMsg); ok {
			m.state = deployStateComplete
			if res.err != nil {
				m.err = res.err
			} else {
				activeKey, err := db.GetActiveSystemKey()
				if err != nil {
					m.err = fmt.Errorf(i18n.T("deploy.error_get_serial_for_status"), err)
				} else {
					m.status = i18n.T("deploy.success", m.selectedAccount.String(), activeKey.Serial)
				}
			}
		}
		// Don't process other input while deployment is running
		return m, nil
	case deployStateComplete:
		return m.updateComplete(msg)
	}
	return m, nil
}

// updateMenu handles input when the main deployment menu is active.
func (m deployModel) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			return m, func() tea.Msg { return backToMenuMsg{} }
		case "up", "k":
			if m.menuCursor > 0 {
				m.menuCursor--
			}
		case "down", "j":
			if m.menuCursor < 3 { // There are 4 menu items (0-3)
				m.menuCursor++
			}
		case "enter":
			switch m.menuCursor {
			case 0: // Deploy to Fleet (fully automatic)
				m.state = deployStateFleetInProgress
				var err error
				m.accountsInFleet, err = db.GetAllActiveAccounts()
				if err != nil {
					m.err = err
					return m, nil
				}
				if len(m.accountsInFleet) == 0 {
					m.status = i18n.T("deploy.no_accounts")
					return m, nil
				}
				m.fleetResults = make(map[int]error, len(m.accountsInFleet))
				m.status = i18n.T("deploy.starting_fleet")
				cmds := make([]tea.Cmd, len(m.accountsInFleet))
				for i, acc := range m.accountsInFleet {
					cmds[i] = performDeploymentCmd(acc)
				}
				return m, tea.Batch(cmds...)
			case 2: // Deploy to Tag
				m.state = deployStateSelectTag
				allAccounts, err := db.GetAllAccounts()
				if err != nil {
					m.err = err
					return m, nil
				}
				uniqueTags := make(map[string]struct{})
				for _, acc := range allAccounts {
					if acc.Tags != "" {
						for _, tag := range strings.Split(acc.Tags, ",") {
							trimmedTag := strings.TrimSpace(tag)
							if trimmedTag != "" {
								uniqueTags[trimmedTag] = struct{}{}
							}
						}
					}
				}
				m.tags = make([]string, 0, len(uniqueTags))
				for tag := range uniqueTags {
					m.tags = append(m.tags, tag)
				}
				sort.Strings(m.tags)
				m.tagCursor = 0
				m.status = ""
				return m, nil
			case 1: // Deploy to Single Account
				m.action = actionDeploySingle
				var err error
				m.accounts, err = db.GetAllActiveAccounts()
				if err != nil {
					m.err = err
					return m, nil
				}
				m.state = deployStateSelectAccount
				m.accountCursor = 0
				m.status = ""
				return m, nil
			case 3: // Get authorized_keys for Account
				m.action = actionGetKeys
				var err error
				// Only allow deploying to or viewing keys for active accounts.
				m.accounts, err = db.GetAllActiveAccounts()
				if err != nil {
					m.err = err
					return m, nil
				}
				m.state = deployStateSelectAccount
				m.accountCursor = 0
				m.status = ""
				return m, nil
			}
		}
	}
	return m, nil
}

// updateAccountSelection handles input when the user is selecting an account.
func (m deployModel) updateAccountSelection(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "/":
			m.isFilteringAccount = true
			m.accountFilter = ""
			m.status = ""
			return m, nil
		case "up", "k":
			filteredAccounts := m.getFilteredAccounts()
			if m.accountCursor > 0 {
				m.accountCursor--
			} else if len(filteredAccounts) > 0 {
				// Wrap around to the bottom
				m.accountCursor = len(filteredAccounts) - 1
			}
			return m, nil
		case "down", "j":
			filteredAccounts := m.getFilteredAccounts()
			if len(filteredAccounts) > 0 {
				if m.accountCursor < len(filteredAccounts)-1 {
					m.accountCursor++
				} else {
					m.accountCursor = 0 // Wrap around to the top
				}
			}
			return m, nil
		case "esc":
			if m.isFilteringAccount {
				m.isFilteringAccount = false
				// Do NOT clear m.accountFilter; persist filter after exiting filter mode
				m.status = ""
				return m, nil
			}
			if m.accountFilter != "" {
				m.accountFilter = ""
				m.status = ""
				return m, nil
			}
			m.status = ""
			m.state = deployStateMenu
			m.err = nil
			return m, nil
		case "backspace":
			if m.isFilteringAccount && m.accountFilter != "" {
				runes := []rune(m.accountFilter)
				if len(runes) > 0 {
					m.accountFilter = string(runes[:len(runes)-1])
				}
				return m, nil
			}
		case "enter":
			if m.isFilteringAccount {
				m.isFilteringAccount = false
				return m, nil
			}
			filteredAccounts := m.getFilteredAccounts()
			if len(filteredAccounts) == 0 {
				return m, nil
			}
			m.selectedAccount = filteredAccounts[m.accountCursor]

			switch m.action {
			case actionGetKeys:
				m.state = deployStateShowAuthorizedKeys
				content, err := deploy.GenerateKeysContent(m.selectedAccount.ID)
				if err != nil {
					m.err = err
					return m, nil
				}
				m.authorizedKeys = content
			case actionDeploySingle:
				m.state = deployStateInProgress
				m.status = i18n.T("deploy.deploying_to", m.selectedAccount.String())
				return m, performDeploymentCmd(m.selectedAccount)
			}
			return m, nil
		default:
			if m.isFilteringAccount && len(msg.String()) == 1 && msg.Type == tea.KeyRunes {
				m.accountFilter += msg.String()
				return m, nil
			}
		}
	case startFilteringMsg:
		// no-op, just to trigger filter mode
		return m, nil
	}
	return m, nil
}

// getFilteredAccounts is a helper to get the list of accounts based on the current filter.
func (m *deployModel) getFilteredAccounts() []model.Account {
	if m.accountFilter == "" {
		return m.accounts
	}
	var filteredAccounts []model.Account
	for _, acc := range m.accounts {
		if strings.Contains(strings.ToLower(acc.String()), strings.ToLower(m.accountFilter)) {
			filteredAccounts = append(filteredAccounts, acc)
		}
	}
	return filteredAccounts
}

// updateSelectTag handles input when the user is selecting a tag.
func (m deployModel) updateSelectTag(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.status = ""
			m.state = deployStateMenu
			m.err = nil
			return m, nil
		case "up", "k":
			if m.tagCursor > 0 {
				m.tagCursor--
			}
		case "down", "j":
			if m.tagCursor < len(m.tags)-1 {
				m.tagCursor++
			}
		case "enter":
			if len(m.tags) == 0 {
				return m, nil
			}
			selectedTag := m.tags[m.tagCursor]

			// Filter accounts by this tag
			allAccounts, err := db.GetAllActiveAccounts()
			if err != nil {
				m.err = err
				return m, nil
			}

			var taggedAccounts []model.Account
			for _, acc := range allAccounts {
				accountTags := make(map[string]struct{})
				for _, t := range strings.Split(acc.Tags, ",") {
					accountTags[strings.TrimSpace(t)] = struct{}{}
				}
				if _, ok := accountTags[selectedTag]; ok {
					taggedAccounts = append(taggedAccounts, acc)
				}
			}

			// Now use the fleet deployment logic with these accounts
			m.state = deployStateFleetInProgress
			m.accountsInFleet = taggedAccounts
			if len(m.accountsInFleet) == 0 {
				m.status = i18n.T("deploy.no_accounts_tag", selectedTag)
				m.state = deployStateMenu // go back to menu
				return m, nil
			}
			m.fleetResults = make(map[int]error, len(m.accountsInFleet))
			m.status = i18n.T("deploy.starting_tag", selectedTag)
			cmds := make([]tea.Cmd, len(m.accountsInFleet))
			for i, acc := range m.accountsInFleet {
				cmds[i] = performDeploymentCmd(acc)
			}
			return m, tea.Batch(cmds...)
		}
	}
	return m, nil
}

// updateShowAuthorizedKeys handles input when viewing the generated keys.
func (m deployModel) updateShowAuthorizedKeys(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.status = ""
			m.state = deployStateSelectAccount
			m.err = nil
			return m, nil
		}
	}
	return m, nil
}

// updateComplete handles input after a deployment operation has finished.
func (m deployModel) updateComplete(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "enter":
			// If we came from a fleet deploy, go back to the main menu
			if len(m.fleetResults) > 0 {
				m.fleetResults = make(map[int]error) // Clear results
				return m, func() tea.Msg { return backToMenuMsg{} }
			}
			// Otherwise, go back to the account selection for single deploys
			m.status = ""
			m.state = deployStateSelectAccount
			m.err = nil
			return m, nil
		}
	}
	return m, nil
}

// View renders the deployment UI based on the current model state.
func (m deployModel) View() string {
	// ...existing code...

	paneStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorSubtle).Padding(1, 2)
	helpFooterStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Background(lipgloss.Color("236")).Padding(0, 1).Italic(true)

	if m.err != nil {
		title := titleStyle.Render(i18n.T("deploy.failed"))
		help := helpFooterStyle.Render(i18n.T("deploy.help_failed"))
		content := fmt.Sprintf(i18n.T("account_form.error"), m.err)
		mainPane := paneStyle.Width(60).Render(lipgloss.JoinVertical(lipgloss.Left, title, "", content))
		return lipgloss.JoinVertical(lipgloss.Left, mainPane, "", help)
	}

	switch m.state {
	case deployStateMenu:
		title := titleStyle.Render(i18n.T("deploy.title"))
		var listItems []string
		menuItems := []string{"deploy.menu.deploy_fleet", "deploy.menu.deploy_single", "deploy.menu.deploy_tag", "deploy.menu.get_keys"}
		for i, itemKey := range menuItems {
			label := i18n.T(itemKey)
			if m.menuCursor == i {
				listItems = append(listItems, selectedItemStyle.Render("▸ "+label))
			} else {
				listItems = append(listItems, itemStyle.Render("  "+label))
			}
		}
		mainPane := paneStyle.Width(60).Render(lipgloss.JoinVertical(lipgloss.Left, title, "", lipgloss.JoinVertical(lipgloss.Left, listItems...)))
		help := helpFooterStyle.Render(i18n.T("deploy.help_menu"))
		if m.status != "" {
			mainPane += "\n" + helpFooterStyle.Render(m.status)
		}
		return lipgloss.JoinVertical(lipgloss.Left, mainPane, "", help)

	case deployStateSelectAccount:
		title := titleStyle.Render(i18n.T("deploy.select_account"))
		var listItems []string
		filteredAccounts := m.getFilteredAccounts()
		if m.accountCursor >= len(filteredAccounts) {
			m.accountCursor = 0
		}
		if len(filteredAccounts) == 0 {
			listItems = append(listItems, helpStyle.Render(i18n.T("deploy.no_accounts")))
		} else {
			for i, acc := range filteredAccounts {
				line := acc.String()
				if m.accountCursor == i {
					listItems = append(listItems, selectedItemStyle.Render("▸ "+line))
				} else {
					listItems = append(listItems, itemStyle.Render("  "+line))
				}
			}
		}
		mainPane := paneStyle.Width(60).Render(lipgloss.JoinVertical(lipgloss.Left, title, "", lipgloss.JoinVertical(lipgloss.Left, listItems...)))
		var filterStatus string
		if m.isFilteringAccount {
			filterStatus = i18n.T("deploy.filtering", m.accountFilter)
		} else if m.accountFilter != "" {
			filterStatus = fmt.Sprintf(i18n.T("deploy.filter_active"), m.accountFilter)
		} else {
			filterStatus = i18n.T("deploy.filter_hint")
		}
		help := helpFooterStyle.Render(i18n.T("deploy.help_select") + "  " + filterStatus)
		return lipgloss.JoinVertical(lipgloss.Left, mainPane, "", help)

	case deployStateSelectTag:
		title := titleStyle.Render(i18n.T("deploy.select_tag"))
		var listItems []string
		if len(m.tags) == 0 {
			listItems = append(listItems, helpStyle.Render(i18n.T("deploy.no_tags")))
		} else {
			for i, tag := range m.tags {
				if m.tagCursor == i {
					listItems = append(listItems, selectedItemStyle.Render("▸ "+tag))
				} else {
					listItems = append(listItems, itemStyle.Render("  "+tag))
				}
			}
		}
		mainPane := paneStyle.Width(60).Render(lipgloss.JoinVertical(lipgloss.Left, title, "", lipgloss.JoinVertical(lipgloss.Left, listItems...)))
		help := helpFooterStyle.Render(i18n.T("deploy.help_select"))
		return lipgloss.JoinVertical(lipgloss.Left, mainPane, "", help)

	case deployStateShowAuthorizedKeys:
		// Render just the keys for easy copy-pasting, with a title and help outside the main content.
		title := titleStyle.Render(i18n.T("deploy.show_keys", m.selectedAccount.String()))
		mainPane := lipgloss.JoinVertical(lipgloss.Left, title, "", m.authorizedKeys)
		help := helpFooterStyle.Render(i18n.T("deploy.help_keys"))
		return lipgloss.JoinVertical(lipgloss.Left, mainPane, "", help)

	case deployStateFleetInProgress:
		title := titleStyle.Render(i18n.T("deploy.deploying_fleet"))
		var statusLines []string
		for _, acc := range m.accountsInFleet {
			res, ok := m.fleetResults[acc.ID]
			var status string
			if !ok {
				status = helpStyle.Render(i18n.T("deploy.pending"))
			} else if res != nil {
				status = "💥 " + helpStyle.Render(i18n.T("deploy.failed_short"))
			} else {
				status = "✅ " + successStyle.Render(i18n.T("deploy.success_short"))
			}
			statusLines = append(statusLines, fmt.Sprintf("  %s %s", acc.String(), status))
		}
		mainPane := paneStyle.Width(60).Render(lipgloss.JoinVertical(lipgloss.Left, title, "", lipgloss.JoinVertical(lipgloss.Left, statusLines...)))
		help := helpFooterStyle.Render(i18n.T("deploy.help_wait"))
		if m.status != "" {
			mainPane += "\n" + helpFooterStyle.Render(m.status)
		}
		return lipgloss.JoinVertical(lipgloss.Left, mainPane, "", help)

	case deployStateInProgress:
		title := titleStyle.Render(i18n.T("deploy.deploying"))
		mainPane := paneStyle.Width(60).Render(lipgloss.JoinVertical(lipgloss.Left, title, "", m.status))
		return lipgloss.JoinVertical(lipgloss.Left, mainPane)

	case deployStateComplete:
		title := titleStyle.Render(i18n.T("deploy.complete"))
		mainPane := paneStyle.Width(60).Render(lipgloss.JoinVertical(lipgloss.Left, title, "", m.status))
		// If it was a fleet deployment, show a detailed summary
		if len(m.fleetResults) > 0 {
			successCount := 0
			var failedAccounts []string
			for _, acc := range m.accountsInFleet {
				if err, ok := m.fleetResults[acc.ID]; ok {
					if err == nil {
						successCount++
					} else {
						failedAccounts = append(failedAccounts, fmt.Sprintf("  - %s: %v", acc.String(), err))
					}
				}
			}
			mainPane += i18n.T("deploy.summary", successCount, len(failedAccounts))
			if len(failedAccounts) > 0 {
				mainPane += i18n.T("deploy.failed_accounts", strings.Join(failedAccounts, "\n"))
			}
		}
		help := helpFooterStyle.Render(i18n.T("deploy.help_complete"))
		return lipgloss.JoinVertical(lipgloss.Left, mainPane, "", help)
	}
	return ""
}

// performDeploymentCmd is a tea.Cmd that executes the full deployment logic for a single account.
func performDeploymentCmd(account model.Account) tea.Cmd {
	return func() tea.Msg {
		var connectKey *model.SystemKey
		var err error
		if account.Serial == 0 {
			// Bootstrap: use the active system key.
			connectKey, err = db.GetActiveSystemKey()
			if err != nil {
				return deploymentResultMsg{account: account, err: fmt.Errorf(i18n.T("deploy.error_get_bootstrap_key"), err)}
			}
			if connectKey == nil {
				return deploymentResultMsg{account: account, err: fmt.Errorf(i18n.T("deploy.error_no_bootstrap_key_tui"))}
			}
		} else {
			// Normal deployment: use the key matching the account's current serial.
			connectKey, err = db.GetSystemKeyBySerial(account.Serial)
			if err != nil {
				return deploymentResultMsg{account: account, err: fmt.Errorf(i18n.T("deploy.error_get_serial_key"), account.Serial, err)}
			}
			if connectKey == nil {
				return deploymentResultMsg{account: account, err: fmt.Errorf(i18n.T("deploy.error_no_serial_key_tui"), account.Serial, account.String())}
			}
		}

		// 2. Generate the target authorized_keys content (always uses the *active* key).
		content, err := deploy.GenerateKeysContent(account.ID)
		if err != nil {
			return deploymentResultMsg{account: account, err: err} // Already i18n
		}
		activeKey, err := db.GetActiveSystemKey() // Need this for the new serial.
		if err != nil || activeKey == nil {
			return deploymentResultMsg{account: account, err: fmt.Errorf(i18n.T("deploy.error_get_active_key_for_serial"))}
		}

		// 3. Establish connection and deploy.
		deployer, err := deploy.NewDeployer(account.Hostname, account.Username, connectKey.PrivateKey)
		if err != nil {
			return deploymentResultMsg{account: account, err: fmt.Errorf(i18n.T("deploy.error_connection_failed_tui"), account.String(), err)}
		}
		defer deployer.Close()

		if err := deployer.DeployAuthorizedKeys(content); err != nil {
			return deploymentResultMsg{account: account, err: fmt.Errorf(i18n.T("deploy.error_deployment_failed_tui"), account.String(), err)}
		}

		// 4. Update the database on success.
		if err := db.UpdateAccountSerial(account.ID, activeKey.Serial); err != nil {
			return deploymentResultMsg{account: account, err: fmt.Errorf(i18n.T("deploy.error_db_update_failed_tui"), err)}
		}

		return deploymentResultMsg{account: account, err: nil} // Success
	}
}

// startFilteringMsg is a message to trigger filter mode in the deploy single account view.
type startFilteringMsg struct{}
