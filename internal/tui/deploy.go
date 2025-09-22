package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/model"
)

type deployState int

const (
	deployStateMenu deployState = iota
	deployStateSelectAccount
	deployStateShowAuthorizedKeys
)

type deployModel struct {
	state           deployState
	menuCursor      int
	accountCursor   int
	accounts        []model.Account
	selectedAccount model.Account
	authorizedKeys  string // The generated authorized_keys content
	err             error
	menuChoices     []string
}

func newDeployModel() deployModel {
	return deployModel{
		state: deployStateMenu,
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
	switch m.state {
	case deployStateMenu:
		return m.updateMenu(msg)
	case deployStateSelectAccount:
		return m.updateAccountSelection(msg)
	case deployStateShowAuthorizedKeys:
		return m.updateShowAuthorizedKeys(msg)
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
				m.err = fmt.Errorf("not yet implemented: Deploy to Fleet")
				return m, nil
			case 1: // Deploy to Single Account
				m.err = fmt.Errorf("not yet implemented: Deploy to Single Account")
				return m, nil
			case 2: // Get authorized_keys for Account
				var err error
				m.accounts, err = db.GetAllAccounts()
				if err != nil {
					m.err = err
					return m, nil
				}
				m.state = deployStateSelectAccount
				m.accountCursor = 0
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
			m.state = deployStateShowAuthorizedKeys
			m.authorizedKeys = m.generateAuthorizedKeysContent()
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
			m.state = deployStateSelectAccount
			m.err = nil
			return m, nil
		}
	}
	return m, nil
}

func (m deployModel) View() string {
	var b strings.Builder

	if m.err != nil {
		b.WriteString(fmt.Sprintf("Error: %v\n", m.err))
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
		b.WriteString(helpStyle.Render("\n(j/k or up/down to navigate, enter to select, q to quit to menu)"))
	case deployStateSelectAccount:
		b.WriteString(titleStyle.Render("🚀 Deploy: Select Account"))
		b.WriteString("\n\n")
		if len(m.accounts) == 0 {
			b.WriteString(helpStyle.Render("No accounts found. Please add one first."))
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
		b.WriteString(helpStyle.Render("\n\n(esc to go back)"))
	}

	return b.String()
}

// generateAuthorizedKeysContent constructs the full authorized_keys file content
// for the currently selected account.
func (m *deployModel) generateAuthorizedKeysContent() string {
	var b strings.Builder

	// 1. Add the *active* Keymaster system key. This shows the ideal state.
	systemKey, err := db.GetActiveSystemKey()
	if err != nil {
		m.err = fmt.Errorf("failed to get active system key: %w", err)
		return ""
	}
	if systemKey == nil {
		m.err = fmt.Errorf("no active system key found. Please generate one via the 'Rotate System Keys' menu.")
		return ""
	}
	b.WriteString(fmt.Sprintf("# Keymaster System Key (Active Serial: %d)\n", systemKey.Serial))
	b.WriteString(systemKey.PublicKey)
	b.WriteString("\n\n")

	// 2. Add all user-assigned public keys
	userKeys, err := db.GetKeysForAccount(m.selectedAccount.ID)
	if err != nil {
		m.err = fmt.Errorf("failed to get user keys for account %s: %w", m.selectedAccount.String(), err)
		return ""
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

	return b.String()
}
