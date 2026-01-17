// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package tui // import "github.com/toeirei/keymaster/internal/tui"

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/internal/core"
	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/i18n"
	"github.com/toeirei/keymaster/internal/model"
	"github.com/toeirei/keymaster/internal/state"
	"github.com/toeirei/keymaster/internal/tui/adapters"
	"github.com/toeirei/keymaster/internal/uiadapters"
	"golang.org/x/term"
)

type deployState int

const (
	deployStateMenu deployState = iota
	deployStateSelectAccount
	deployStateSelectTag
	deployStateShowAuthorizedKeys
	deployStateFleetInProgress
	deployStateEnterPassphrase
	deployStateInProgress
	deployStateComplete
	deployStateEnterFilename
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
	passphraseInput    textinput.Model
	filenameInput      textinput.Model
	pendingCmd         tea.Cmd // Command to re-run after getting passphrase
	wasFleetDeploy     bool    // Flag to remember if the last operation was a fleet deployment
	width, height      int
	searcher           db.AccountSearcher
}

// newDeployModel creates a new model for the deployment view.
// newDeployModelWithSearcher creates a deploy model and accepts an optional
// AccountSearcher for server-side lookups.
func newDeployModelWithSearcher(s db.AccountSearcher) deployModel {
	pi := newPassphraseInput()
	fi := newFilenameInput()
	m := deployModel{
		state:           deployStateMenu,
		fleetResults:    make(map[int]error),
		passphraseInput: pi,
		filenameInput:   fi,
		searcher:        s,
	}

	// Try to initialize with the current terminal size so the view fills
	// the available area on first render. Fall back to 80x24 on error.
	if w, h, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
		m.width = w
		m.height = h
	} else {
		m.width = 80
		m.height = 24
	}

	return m
}

// newDeployModel removed — use `newDeployModelWithSearcher` with an explicit searcher.

// Init initializes the deploy model.
func (m deployModel) Init() tea.Cmd {
	return nil
}

// newPassphraseInput is a helper to create a styled textinput for passwords.
func newPassphraseInput() textinput.Model {
	pi := textinput.New()
	pi.Placeholder = i18n.T("rotate_key.passphrase_placeholder")
	pi.EchoMode = textinput.EchoPassword
	pi.CharLimit = 128
	pi.Width = 50
	return pi
}

// newFilenameInput is a helper to create a styled textinput for filenames.
func newFilenameInput() textinput.Model {
	fi := textinput.New()
	fi.Placeholder = i18n.T("deploy.filename_placeholder")
	fi.CharLimit = 256
	fi.Width = 50
	return fi
}

// Update handles messages and updates the deploy model's state.
func (m deployModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Update model size when terminal is resized so the view can reflow.
	if sizeMsg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = sizeMsg.Width
		m.height = sizeMsg.Height
	}
	switch m.state {
	case deployStateMenu:
		return m.updateMenu(msg)
	case deployStateSelectAccount:
		return m.updateAccountSelection(msg)
	case deployStateSelectTag:
		return m.updateSelectTag(msg)
	case deployStateShowAuthorizedKeys:
		return m.updateShowAuthorizedKeys(msg)
	case deployStateEnterFilename:
		return m.updateEnterFilename(msg)
	case deployStateFleetInProgress:
		if res, ok := msg.(deploymentResultMsg); ok {
			m.fleetResults[res.account.ID] = res.err

			// If any deployment requires a passphrase, stop and ask for it.
			if res.err != nil && deployAdapter.IsPassphraseRequired(res.err) {
				m.state = deployStateEnterPassphrase
				m.err = nil // Clear the error as we are handling it
				m.passphraseInput.Focus()
				// Store a command to re-run the entire fleet deployment.
				cmds := make([]tea.Cmd, len(m.accountsInFleet))
				for i, acc := range m.accountsInFleet {
					cmds[i] = performDeploymentCmd(acc)
				}
				m.pendingCmd = tea.Batch(cmds...)
				return m, textinput.Blink
			}

			// If this individual deployment succeeded, clear the dirty flag.
			if res.err == nil {
				_ = uiadapters.NewStoreAdapter().UpdateAccountIsDirty(res.account.ID, false)
			}

			if len(m.fleetResults) == len(m.accountsInFleet) {
				m.state = deployStateComplete
				m.status = i18n.T("deploy.fleet_complete")
			}
		}
		// No other input handled while fleet deployment is running
		return m, nil
	case deployStateEnterPassphrase:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				passphrase := m.passphraseInput.Value()
				state.PasswordCache.Set([]byte(passphrase))
				m.fleetResults = make(map[int]error) // Clear previous fleet results before retry
				m.state = deployStateInProgress      // Go back to in-progress
				m.status = i18n.T("deploy.retrying")
				return m, m.pendingCmd // Re-run the original command
			case "esc":
				m.state = deployStateMenu
				m.err = nil
				m.status = i18n.T("deploy.passphrase_cancelled")
				return m, nil
			}
		}
		var cmd tea.Cmd
		m.passphraseInput, cmd = m.passphraseInput.Update(msg)
		return m, cmd
	case deployStateInProgress:
		if res, ok := msg.(deploymentResultMsg); ok {
			if res.err != nil {
				// First, check for the specific passphrase error.
				if deployAdapter.IsPassphraseRequired(res.err) {
					// The deployer needs a passphrase. Switch to that state.
					m.state = deployStateEnterPassphrase
					m.err = nil // Clear the error as we are handling it
					m.passphraseInput.Focus()
					// Store the command that failed so we can retry it
					m.pendingCmd = performDeploymentCmd(res.account)
					return m, textinput.Blink
				}
				// It's a different, final error.
				m.state = deployStateComplete
				m.err = res.err
				// For single-host deploys, the status is just the error.
				// For fleet deploys, the summary screen will show the error.
				// So, we don't need to set m.status here.

			} else { // No error, success case for a single deployment.
				m.state = deployStateComplete
				if !m.wasFleetDeploy { // Only set this status for single, non-fleet deploys
					activeKey, err := adapters.KeyReader.GetActiveSystemKey()
					if err != nil {
						m.err = fmt.Errorf(i18n.T("deploy.error_get_serial_for_status"), err)
					} else {
						m.status = i18n.T("deploy.success", m.selectedAccount.String(), activeKey.Serial)
					}
				}
				// Clear is_dirty for the account on successful deploy
				_ = uiadapters.NewStoreAdapter().UpdateAccountIsDirty(m.selectedAccount.ID, false)
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
				// no-op: do not track hasInteracted for deploy view
			}
		case "down", "j":
			if m.menuCursor < 3 { // There are 4 menu items (0-3)
				m.menuCursor++
				// no-op: do not track hasInteracted for deploy view
			}
		case "enter":
			// no-op: do not track hasInteracted for deploy view
			switch m.menuCursor {
			case 0: // Deploy to Fleet (fully automatic)
				m.wasFleetDeploy = true
				m.state = deployStateFleetInProgress
				var err error
				m.accountsInFleet, err = uiadapters.NewStoreAdapter().GetAllActiveAccounts()
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
			case 1: // Deploy to Single Account
				m.wasFleetDeploy = false
				m.action = actionDeploySingle
				var err error
				m.accounts, err = uiadapters.NewStoreAdapter().GetAllActiveAccounts()
				if err != nil {
					m.err = err
					return m, nil
				}
				m.state = deployStateSelectAccount
				m.accountCursor = 0
				m.status = ""
				return m, nil
			case 2: // Deploy to Tag
				m.wasFleetDeploy = true // Deploying to a tag is a fleet operation
				m.state = deployStateSelectTag
				allAccounts, err := uiadapters.NewStoreAdapter().GetAllAccounts()
				if err != nil {
					m.err = err
					return m, nil
				}
				// Build unique, sorted tag list using core helpers
				m.tags = core.UniqueTags(allAccounts)
				m.tagCursor = 0
				m.status = ""
				return m, nil
			case 3: // Get authorized_keys for Account
				m.wasFleetDeploy = false
				m.action = actionGetKeys
				var err error
				// Only allow deploying to or viewing keys for active accounts.
				m.accounts, err = uiadapters.NewStoreAdapter().GetAllActiveAccounts()
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
				// Build authorized_keys using core plan builder (pure helper).
				// Fetch system key and public keys, then build a deterministic plan.
				sk, err := adapters.KeyReader.GetActiveSystemKey()
				if err != nil {
					m.err = err
					return m, nil
				}
				km := db.DefaultKeyManager()
				if km == nil {
					m.err = fmt.Errorf("no key manager available")
					return m, nil
				}
				globalKeys, err := km.GetGlobalPublicKeys()
				if err != nil {
					m.err = err
					return m, nil
				}
				accountKeys, err := km.GetKeysForAccount(m.selectedAccount.ID)
				if err != nil {
					m.err = err
					return m, nil
				}
				plan, err := core.BuildBootstrapDeploymentPlan(core.BootstrapParams{
					Username: m.selectedAccount.Username,
					Hostname: m.selectedAccount.Hostname,
					Label:    m.selectedAccount.Label,
					Tags:     m.selectedAccount.Tags,
				}, sk, globalKeys, accountKeys)
				if err != nil {
					m.err = err
					return m, nil
				}
				m.authorizedKeys = plan.AuthorizedKeysBlob
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
		if core.ContainsIgnoreCase(acc.String(), m.accountFilter) {
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
				// no-op: do not track hasInteracted for deploy view
			}
		case "down", "j":
			if m.tagCursor < len(m.tags)-1 {
				m.tagCursor++
				// no-op: do not track hasInteracted for deploy view
			}
		case "enter":
			// no-op: do not track hasInteracted for deploy view
			if len(m.tags) == 0 {
				return m, nil
			}
			selectedTag := m.tags[m.tagCursor]

			// Filter accounts by this tag
			allAccounts, err := uiadapters.NewStoreAdapter().GetAllActiveAccounts()
			if err != nil {
				m.err = err
				return m, nil
			}

			// Filter accounts by selected tag using core helper
			accountsByTag := core.BuildAccountsByTag(allAccounts)
			taggedAccounts := accountsByTag[selectedTag]

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
			m.status = "" // Clear copy status on exit
			m.state = deployStateSelectAccount
			m.err = nil
			return m, nil
		case "c":
			err := clipboard.WriteAll(m.authorizedKeys)
			if err != nil {
				m.status = i18n.T("deploy.status.copy_failed", err.Error())
			} else {
				m.status = i18n.T("deploy.status.copy_success")
				// no-op: do not track hasInteracted for deploy view
			}
			return m, nil
		case "s":
			// no-op: do not track hasInteracted for deploy view
			m.state = deployStateEnterFilename
			m.filenameInput.Focus()
			m.status = ""
			return m, textinput.Blink
		case "x":
			// Clear the is_dirty flag for this account (user-confirmed).
			if err := uiadapters.NewStoreAdapter().UpdateAccountIsDirty(m.selectedAccount.ID, false); err != nil {
				m.status = fmt.Sprintf("clear failed: %v", err)
			} else {
				m.status = i18n.T("deploy.status.cleared")
			}
			return m, nil
		}
	}
	return m, nil
}

// updateEnterFilename handles input when user is entering a filename.
func (m deployModel) updateEnterFilename(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			filename := m.filenameInput.Value()
			if filename == "" {
				return m, nil
			}

			err := WriteKeyFile(filename, []byte(m.authorizedKeys))
			if err != nil {
				m.state = deployStateComplete
				m.status = fmt.Sprintf(i18n.T("deploy.status.write_failed"), err.Error())
			} else {
				m.state = deployStateComplete
				m.status = fmt.Sprintf(i18n.T("deploy.status.write_success"), filename)
			}
			m.filenameInput.Reset()
			return m, nil
		case "esc":
			m.state = deployStateSelectAccount
			m.err = nil
			m.status = ""
			m.filenameInput.Reset()
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.filenameInput, cmd = m.filenameInput.Update(msg)
	return m, cmd
}

// updateComplete handles input after a deployment operation has finished.
func (m deployModel) updateComplete(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "enter":
			// If the last operation was a fleet deploy, go back to the main menu
			if m.wasFleetDeploy {
				m.fleetResults = make(map[int]error) // Clear results
				return m, func() tea.Msg { return backToMenuMsg{} }
			}
			// Otherwise, go back to the account selection for single deploys
			m.status = ""
			m.state = deployStateSelectAccount
			m.err = nil
			return m, nil
		case "x":
			// Allow user to clear the is_dirty flag from the completion screen.
			if err := uiadapters.NewStoreAdapter().UpdateAccountIsDirty(m.selectedAccount.ID, false); err != nil {
				m.status = fmt.Sprintf("clear failed: %v", err)
			} else {
				m.status = i18n.T("deploy.status.cleared")
			}
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
	paneWidth := m.width
	if paneWidth <= 0 {
		paneWidth = 80
	}

	if m.err != nil {
		title := titleStyle.Render(i18n.T("deploy.failed"))
		help := helpFooterStyle.Render(i18n.T("deploy.help_failed"))
		content := fmt.Sprintf(i18n.T("account_form.error"), m.err)
		mainPane := paneStyle.Width(paneWidth).Render(lipgloss.JoinVertical(lipgloss.Left, title, "", content))
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
		footerStr := m.footerFor(paneWidth)
		headerHeight := lipgloss.Height(title)
		footerHeight := lipgloss.Height(footerStr)
		paneHeight := m.height - headerHeight - footerHeight - 2
		if paneHeight <= 2 {
			paneHeight = 3
		}
		mainPane := paneStyle.Width(paneWidth).Height(paneHeight).Render(lipgloss.JoinVertical(lipgloss.Left, title, "", lipgloss.JoinVertical(lipgloss.Left, listItems...)))
		// Centralized footer for consistent right-aligned layout
		help := footerStr
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
		footerStr := m.footerFor(paneWidth)
		headerHeight := lipgloss.Height(title)
		footerHeight := lipgloss.Height(footerStr)
		paneHeight := m.height - headerHeight - footerHeight - 2
		if paneHeight <= 2 {
			paneHeight = 3
		}
		mainPane := paneStyle.Width(paneWidth).Height(paneHeight).Render(lipgloss.JoinVertical(lipgloss.Left, title, "", lipgloss.JoinVertical(lipgloss.Left, listItems...)))
		help := footerStr
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
		footerStr := m.footerFor(paneWidth)
		headerHeight := lipgloss.Height(title)
		footerHeight := lipgloss.Height(footerStr)
		paneHeight := m.height - headerHeight - footerHeight - 2
		if paneHeight <= 2 {
			paneHeight = 3
		}
		mainPane := paneStyle.Width(paneWidth).Height(paneHeight).Render(lipgloss.JoinVertical(lipgloss.Left, title, "", lipgloss.JoinVertical(lipgloss.Left, listItems...)))
		help := footerStr
		return lipgloss.JoinVertical(lipgloss.Left, mainPane, "", help)

	case deployStateShowAuthorizedKeys:
		// Render just the keys for easy copy-pasting, with a title and help outside the main content.
		title := titleStyle.Render(i18n.T("deploy.show_keys", m.selectedAccount.String()))
		var content []string
		if m.status != "" {
			content = append(content, statusMessageStyle.Render(m.status), "")
		}
		contentStr := lipgloss.JoinVertical(lipgloss.Left, title, "", lipgloss.JoinVertical(lipgloss.Left, content...), m.authorizedKeys)
		footerStr := m.footerFor(paneWidth)
		headerHeight := lipgloss.Height(title)
		footerHeight := lipgloss.Height(footerStr)
		paneHeight := m.height - headerHeight - footerHeight - 2
		if paneHeight <= 2 {
			paneHeight = 3
		}
		mainPane := paneStyle.Width(paneWidth).Height(paneHeight).Render(contentStr)
		help := footerStr
		return lipgloss.JoinVertical(lipgloss.Left, mainPane, "", help)

	case deployStateEnterFilename:
		var b strings.Builder
		b.WriteString(titleStyle.Render(i18n.T("deploy.enter_filename_title")))
		b.WriteString("\n\n")
		b.WriteString(i18n.T("deploy.enter_filename_prompt"))
		b.WriteString("\n\n")
		b.WriteString(m.filenameInput.View())
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render(i18n.T("deploy.help_enter_filename")))

		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			dialogBoxStyle.Render(b.String()),
		)

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
		footerStr := m.footerFor(paneWidth)
		headerHeight := lipgloss.Height(title)
		footerHeight := lipgloss.Height(footerStr)
		paneHeight := m.height - headerHeight - footerHeight - 2
		if paneHeight <= 2 {
			paneHeight = 3
		}
		mainPane := paneStyle.Width(paneWidth).Height(paneHeight).Render(lipgloss.JoinVertical(lipgloss.Left, title, "", lipgloss.JoinVertical(lipgloss.Left, statusLines...)))
		help := footerStr
		if m.status != "" {
			mainPane += "\n" + helpFooterStyle.Render(m.status)
		}
		return lipgloss.JoinVertical(lipgloss.Left, mainPane, "", help)

	case deployStateEnterPassphrase:
		var b strings.Builder
		b.WriteString(titleStyle.Render(i18n.T("deploy.passphrase_title")))
		b.WriteString("\n\n")
		b.WriteString(i18n.T("deploy.passphrase_prompt"))
		b.WriteString("\n\n")
		b.WriteString(m.passphraseInput.View())
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render(i18n.T("deploy.passphrase_help")))

		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			dialogBoxStyle.Render(b.String()),
		)

	case deployStateInProgress:
		title := titleStyle.Render(i18n.T("deploy.deploying"))
		footerStr := m.footerFor(paneWidth)
		headerHeight := lipgloss.Height(title)
		footerHeight := lipgloss.Height(footerStr)
		paneHeight := m.height - headerHeight - footerHeight - 2
		if paneHeight <= 2 {
			paneHeight = 3
		}
		mainPane := paneStyle.Width(paneWidth).Height(paneHeight).Render(lipgloss.JoinVertical(lipgloss.Left, title, "", m.status))
		return lipgloss.JoinVertical(lipgloss.Left, mainPane)

	case deployStateComplete:
		title := titleStyle.Render(i18n.T("deploy.complete"))
		// Build a consistent result block (primary status, warnings, errors)
		primary := m.status
		// Collect any fleet-level failures as an error summary
		var warnings []string
		if len(m.fleetResults) > 0 {
			successCount := 0
			var failedAccounts []string
			for _, acc := range m.accountsInFleet {
				if err, ok := m.fleetResults[acc.ID]; ok {
					if err == nil {
						successCount++
					} else {
						failedAccounts = append(failedAccounts, fmt.Sprintf("%s: %v", acc.String(), err))
					}
				}
			}
			// Add a brief summary to the primary message
			primary = primary + "\n" + i18n.T("deploy.summary", successCount, len(failedAccounts))
			if len(failedAccounts) > 0 {
				warnings = append(warnings, i18n.T("deploy.failed_accounts", strings.Join(failedAccounts, "\n")))
			}
		}

		resultBlock := renderResultBlock(primary, warnings, m.err)
		mainPane := paneStyle.Width(paneWidth).Render(lipgloss.JoinVertical(lipgloss.Left, title, "", resultBlock))
		help := m.footerFor(paneWidth)
		return lipgloss.JoinVertical(lipgloss.Left, mainPane, "", help)
	}
	return ""
}

// footerFor builds the standardized footer for the deploy view based on
// the current state and filter flags. It returns a fully styled string.
func (m deployModel) footerFor(width int) string {
	footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Background(lipgloss.Color("236")).Padding(0, 1).Italic(true)

	// Default tokens
	left := ""
	right := ""

	switch m.state {
	case deployStateMenu:
		left = i18n.T("deploy.help_menu")
	case deployStateSelectAccount:
		left = i18n.T("deploy.help_select")
		if m.isFilteringAccount {
			right = i18n.T("deploy.filtering", m.accountFilter)
		} else if m.accountFilter != "" {
			right = i18n.T("deploy.filter_active", m.accountFilter)
		} else {
			right = i18n.T("deploy.filter_hint")
		}
	case deployStateSelectTag:
		left = i18n.T("deploy.help_select")
	case deployStateShowAuthorizedKeys:
		left = i18n.T("deploy.help_keys")
		// Allow user to mark account as deployed from the keys view
		right = i18n.T("deploy.help_mark_deployed")
	case deployStateFleetInProgress:
		left = i18n.T("deploy.help_wait")
	case deployStateEnterPassphrase:
		left = i18n.T("deploy.passphrase_help")
	case deployStateEnterFilename:
		left = i18n.T("deploy.help_enter_filename")
	case deployStateInProgress:
		left = ""
	case deployStateComplete:
		left = i18n.T("deploy.help_complete")
		// Allow quick mark as deployed from the completion screen
		right = i18n.T("deploy.help_mark_deployed")
	default:
		left = i18n.T("deploy.help_menu")
	}

	return footerStyle.Render(AlignFooter(left, right, width))
}

// performDeploymentCmd is a tea.Cmd that executes the full deployment logic for a single account.
func performDeploymentCmd(account model.Account) tea.Cmd {
	return func() tea.Msg {
		err := core.RunDeployForAccount(context.Background(), deployAdapter, account, nil)
		return deploymentResultMsg{account: account, err: err}
	}
}

// startFilteringMsg is a message to trigger filter mode in the deploy single account view.
type startFilteringMsg struct{}
