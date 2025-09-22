package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/deploy"
	"github.com/toeirei/keymaster/internal/model"
)

type deployState int

const (
	deployStateMenu deployState = iota
	deployStateSelectAccount
	deployStateShowAuthorizedKeys
	deployStateFleetInProgress
	deployStateInProgress
	deployStateComplete
)

type deployAction int

const (
	actionGetKeys deployAction = iota
	actionDeploySingle
)

// A message to signal deployment is complete.
type deploymentResultMsg struct {
	account model.Account
	err     error
}

type deployModel struct {
	state           deployState
	action          deployAction
	menuCursor      int
	accountCursor   int
	accounts        []model.Account
	accountsInFleet []model.Account // Keep order for display
	fleetResults    map[int]error   // map account ID to error for quick lookup
	selectedAccount model.Account
	authorizedKeys  string // The generated authorized_keys content
	status          string
	err             error
	menuChoices     []string
}

func newDeployModel() deployModel {
	return deployModel{
		state:        deployStateMenu,
		fleetResults: make(map[int]error),
		menuChoices: []string{
			"Deploy to Fleet (fully automatic)",
			"Deploy to Single Account",
			"Get authorized_keys for Account",
		},
	}
}

func (m deployModel) Init() tea.Cmd {
	return nil
}

func (m deployModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case deploymentResultMsg:
		if m.state == deployStateInProgress { // Single deployment finished
			m.state = deployStateComplete
			if msg.err != nil {
				m.err = msg.err
			} else {
				activeKey, err := db.GetActiveSystemKey()
				if err != nil {
					m.err = fmt.Errorf("deployment succeeded, but could not get new serial for status message: %w", err)
				} else {
					m.status = fmt.Sprintf("Successfully deployed to %s and updated serial to #%d.", m.selectedAccount.String(), activeKey.Serial)
				}
			}
		} else if m.state == deployStateFleetInProgress { // One fleet member finished
			m.fleetResults[msg.account.ID] = msg.err
			if len(m.fleetResults) == len(m.accountsInFleet) {
				m.state = deployStateComplete
				m.status = "Fleet deployment complete."
			}
		}
		return m, nil
	}

	switch m.state {
	case deployStateMenu:
		return m.updateMenu(msg)
	case deployStateSelectAccount:
		return m.updateAccountSelection(msg)
	case deployStateShowAuthorizedKeys:
		return m.updateShowAuthorizedKeys(msg)
	case deployStateFleetInProgress:
		// No input handled while fleet deployment is running
	case deployStateInProgress:
		return m, nil // Don't process input while deployment is running
	case deployStateComplete:
		return m.updateComplete(msg)
	}
	return m, nil
}

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
			if m.menuCursor < len(m.menuChoices)-1 {
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
					m.status = "No active accounts to deploy to."
					return m, nil
				}
				m.fleetResults = make(map[int]error, len(m.accountsInFleet))
				m.status = "Starting fleet deployment..."
				cmds := make([]tea.Cmd, len(m.accountsInFleet))
				for i, acc := range m.accountsInFleet {
					cmds[i] = performDeploymentCmd(acc)
				}
				return m, tea.Batch(cmds...)
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
			case 2: // Get authorized_keys for Account
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

func (m deployModel) updateAccountSelection(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.status = ""
			m.state = deployStateMenu
			m.err = nil
			return m, nil
		case "up", "k":
			if m.accountCursor > 0 {
				m.accountCursor--
			}
		case "down", "j":
			if m.accountCursor < len(m.accounts)-1 {
				m.accountCursor++
			}
		case "enter":
			if len(m.accounts) == 0 {
				return m, nil
			}
			m.selectedAccount = m.accounts[m.accountCursor]

			switch m.action {
			case actionGetKeys:
				m.state = deployStateShowAuthorizedKeys
				content, err := generateKeysContent(m.selectedAccount.ID)
				if err != nil {
					m.err = err
					return m, nil
				}
				m.authorizedKeys = content
			case actionDeploySingle:
				m.state = deployStateInProgress
				m.status = fmt.Sprintf("Deploying to %s...", m.selectedAccount.String())
				return m, performDeploymentCmd(m.selectedAccount)
			}
			return m, nil
		}
	}
	return m, nil
}

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

func (m deployModel) setErr(err error) {
	m.state = deployStateComplete
	m.err = err
	m.status = "Deployment failed."
}

func (m deployModel) View() string {
	var b strings.Builder

	if m.err != nil {
		b.WriteString(titleStyle.Render("💥 Deployment Failed"))
		b.WriteString("\n\n")
		b.WriteString(fmt.Sprintf("Error: %v", m.err))
		b.WriteString(helpStyle.Render("\n(esc to go back)"))
		return b.String()
	}

	switch m.state {
	case deployStateMenu:
		b.WriteString(titleStyle.Render("🚀 Deploy to Fleet"))
		b.WriteString("\n\n")
		for i, choice := range m.menuChoices {
			if m.menuCursor == i {
				b.WriteString(selectedItemStyle.Render("» " + choice))
			} else {
				b.WriteString(itemStyle.Render(choice))
			}
			b.WriteString("\n")
		}
		b.WriteString(helpStyle.Render("\n(j/k or up/down, enter to select, q to quit)"))
		if m.status != "" {
			b.WriteString(helpStyle.Render("\n\n" + m.status))
		}
	case deployStateSelectAccount:
		b.WriteString(titleStyle.Render("🚀 Deploy: Select Account"))
		b.WriteString("\n\n")
		if len(m.accounts) == 0 {
			b.WriteString(helpStyle.Render("No active accounts found. Please add one or enable an existing one."))
		} else {
			maxUserLen := 0
			for _, acc := range m.accounts {
				if len(acc.Username) > maxUserLen {
					maxUserLen = len(acc.Username)
				}
			}
			for i, acc := range m.accounts {
				userPart := fmt.Sprintf("%*s", maxUserLen, acc.Username)
				hostPart := fmt.Sprintf("@%s", acc.Hostname)
				line := userPart + hostPart
				if m.accountCursor == i {
					b.WriteString(selectedItemStyle.Render("» " + line))
				} else {
					b.WriteString(itemStyle.Render(line))
				}
				b.WriteString("\n")
			}
		}
		b.WriteString(helpStyle.Render("\n(enter to select, esc to go back)"))
	case deployStateShowAuthorizedKeys:
		b.WriteString(titleStyle.Render(fmt.Sprintf("📄 authorized_keys for %s", m.selectedAccount.String())))
		b.WriteString("\n\n")
		b.WriteString(m.authorizedKeys)
		b.WriteString(helpStyle.Render("\n(esc to go back)"))
	case deployStateFleetInProgress:
		b.WriteString(titleStyle.Render("🚀 Deploying to Fleet..."))
		b.WriteString("\n\n")
		for _, acc := range m.accountsInFleet {
			res, ok := m.fleetResults[acc.ID]
			var status string
			if !ok {
				status = helpStyle.Render("pending...")
			} else if res != nil {
				status = "💥 " + helpStyle.Render("failed")
			} else {
				status = "✅ " + selectedItemStyle.Render("success")
			}
			b.WriteString(fmt.Sprintf("  %s %s\n", acc.String(), status))
		}
		b.WriteString(helpStyle.Render("\n(Please wait...)\n"))
		if m.status != "" {
			b.WriteString(helpStyle.Render("\n" + m.status))
		}

	case deployStateInProgress:
		b.WriteString(titleStyle.Render("🚀 Deploying..."))
		b.WriteString("\n\n")
		b.WriteString(m.status)
	case deployStateComplete:
		b.WriteString(titleStyle.Render("✅ Deployment Complete"))
		b.WriteString("\n\n")
		b.WriteString(m.status)
		b.WriteString(helpStyle.Render("\n(press enter or esc to continue)"))
	}

	return b.String()
}

// performDeploymentCmd is a tea.Cmd that executes the full deployment logic for a single account.
func performDeploymentCmd(account model.Account) tea.Cmd {
	return func() tea.Msg {
		// 1. Determine which system key to use for the SSH connection.
		var connectKey *model.SystemKey
		var err error
		if account.Serial == 0 {
			// Bootstrap: use the active key. The user must have placed this manually.
			connectKey, err = db.GetActiveSystemKey()
			if err != nil {
				return deploymentResultMsg{account: account, err: fmt.Errorf("failed to get active system key for bootstrap: %w", err)}
			}
			if connectKey == nil {
				return deploymentResultMsg{account: account, err: fmt.Errorf("no active system key found for bootstrap. Please generate one.")}
			}
		} else {
			// Normal deployment: use the key matching the account's current serial.
			connectKey, err = db.GetSystemKeyBySerial(account.Serial)
			if err != nil {
				return deploymentResultMsg{account: account, err: fmt.Errorf("failed to get system key with serial %d: %w", account.Serial, err)}
			}
			if connectKey == nil {
				return deploymentResultMsg{account: account, err: fmt.Errorf("database inconsistency: no system key found for serial %d on account %s", account.Serial, account.String())}
			}
		}

		// 2. Generate the target authorized_keys content (always uses the *active* key).
		content, err := generateKeysContent(account.ID)
		if err != nil {
			return deploymentResultMsg{account: account, err: err}
		}
		activeKey, err := db.GetActiveSystemKey() // Need this for the new serial.
		if err != nil || activeKey == nil {
			return deploymentResultMsg{account: account, err: fmt.Errorf("could not retrieve active system key for serial update")}
		}

		// 3. Establish connection and deploy.
		deployer, err := deploy.NewDeployer(account.Hostname, account.Username, connectKey.PrivateKey)
		if err != nil {
			return deploymentResultMsg{account: account, err: fmt.Errorf("failed to connect to %s: %w", account.String(), err)}
		}
		defer deployer.Close()

		if err := deployer.DeployAuthorizedKeys(content); err != nil {
			return deploymentResultMsg{account: account, err: fmt.Errorf("deployment to %s failed: %w", account.String(), err)}
		}

		// 4. Update the database on success.
		if err := db.UpdateAccountSerial(account.ID, activeKey.Serial); err != nil {
			return deploymentResultMsg{account: account, err: fmt.Errorf("deployment succeeded, but failed to update database serial: %w", err)}
		}

		return deploymentResultMsg{account: account, err: nil} // Success
	}
}

// generateAuthorizedKeysContent constructs the full authorized_keys file content
// for the currently selected account.
func generateKeysContent(accountID int) (string, error) {
	var b strings.Builder

	// 1. Add the *active* Keymaster system key. This shows the ideal state.
	systemKey, err := db.GetActiveSystemKey()
	if err != nil {
		return "", fmt.Errorf("failed to get active system key: %w", err)
	}
	if systemKey == nil {
		return "", fmt.Errorf("no active system key found. Please generate one via the 'Rotate System Keys' menu.")
	}
	b.WriteString(fmt.Sprintf("# Keymaster System Key (Active Serial: %d)\n", systemKey.Serial))
	b.WriteString(systemKey.PublicKey)
	b.WriteString("\n\n")

	// 2. Add all user-assigned public keys
	userKeys, err := db.GetKeysForAccount(accountID)
	if err != nil {
		return "", fmt.Errorf("failed to get user keys for account %d: %w", accountID, err)
	}
	if len(userKeys) > 0 {
		b.WriteString("# User-assigned Public Keys\n")
		for _, key := range userKeys {
			b.WriteString(key.String())
			b.WriteString("\n")
		}
	} else {
		b.WriteString("# No user-assigned public keys for this account.\n")
	}

	return b.String(), nil
}
