package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/model"
)

type assignState int

const (
	assignStateSelectAccount assignState = iota
	assignStateSelectKeys
)

type assignKeysModel struct {
	state           assignState
	accounts        []model.Account
	keys            []model.PublicKey
	accountCursor   int
	keyCursor       int
	selectedAccount model.Account
	assignedKeys    map[int]struct{} // Set of key IDs assigned to the selected account
	status          string
	err             error
}

func newAssignKeysModel() assignKeysModel {
	m := assignKeysModel{
		state:        assignStateSelectAccount,
		assignedKeys: make(map[int]struct{}),
	}

	var err error
	m.accounts, err = db.GetAllAccounts()
	if err != nil {
		m.err = err
		return m
	}
	// We also fetch all keys now, so we don't have to do it later.
	m.keys, err = db.GetAllPublicKeys()
	if err != nil {
		m.err = err
	}
	return m
}

func (m assignKeysModel) Init() tea.Cmd {
	return nil
}

func (m assignKeysModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.state {
	case assignStateSelectAccount:
		return m.updateAccountSelection(msg)
	case assignStateSelectKeys:
		return m.updateKeySelection(msg)
	}
	return m, nil
}

// updateAccountSelection handles input when the user is selecting an account.
func (m assignKeysModel) updateAccountSelection(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			return m, func() tea.Msg { return backToMenuMsg{} }
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
			// User has selected an account, move to key selection view.
			m.selectedAccount = m.accounts[m.accountCursor]
			m.state = assignStateSelectKeys
			m.keyCursor = 0 // Reset key cursor
			m.status = ""   // Clear status

			// Populate the set of currently assigned keys for this account.
			assigned, err := db.GetKeysForAccount(m.selectedAccount.ID)
			if err != nil {
				m.err = err
				return m, nil
			}
			m.assignedKeys = make(map[int]struct{})
			for _, key := range assigned {
				m.assignedKeys[key.ID] = struct{}{}
			}
			return m, nil
		}
	}
	return m, nil
}

// updateKeySelection handles input when the user is assigning keys to an account.
func (m assignKeysModel) updateKeySelection(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			// Go back to account selection.
			m.state = assignStateSelectAccount
			m.status = ""
			return m, nil
		case "up", "k":
			if m.keyCursor > 0 {
				m.keyCursor--
			}
		case "down", "j":
			if m.keyCursor < len(m.keys)-1 {
				m.keyCursor++
			}
		case " ", "enter":
			if len(m.keys) == 0 {
				return m, nil
			}
			keyToToggle := m.keys[m.keyCursor]
			_, isAssigned := m.assignedKeys[keyToToggle.ID]

			var err error
			if isAssigned {
				// Unassign it
				err = db.UnassignKeyFromAccount(keyToToggle.ID, m.selectedAccount.ID)
				if err == nil {
					delete(m.assignedKeys, keyToToggle.ID)
					m.status = fmt.Sprintf("Unassigned key '%s'", keyToToggle.Comment)
				}
			} else {
				// Assign it
				err = db.AssignKeyToAccount(keyToToggle.ID, m.selectedAccount.ID)
				if err == nil {
					m.assignedKeys[keyToToggle.ID] = struct{}{}
					m.status = fmt.Sprintf("Assigned key '%s'", keyToToggle.Comment)
				}
			}
			if err != nil {
				m.err = err
			}
			return m, nil
		}
	}
	return m, nil
}

func (m assignKeysModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	switch m.state {
	case assignStateSelectAccount:
		return m.viewAccountSelection()
	case assignStateSelectKeys:
		return m.viewKeySelection()
	}
	return "Something went wrong."
}

func (m assignKeysModel) viewAccountSelection() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("🔑 Assign Keys: Select an Account"))
	b.WriteString("\n\n")

	if len(m.accounts) == 0 {
		b.WriteString(helpStyle.Render("No accounts found. Please add one first."))
	} else {
		// Using the same alignment logic from accounts.go
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

	b.WriteString(helpStyle.Render("\n(enter to select, q to quit to menu)"))
	return b.String()
}

func (m assignKeysModel) viewKeySelection() string {
	var b strings.Builder
	title := fmt.Sprintf("🔑 Assign Keys for: %s", m.selectedAccount.String())
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")

	if len(m.keys) == 0 {
		b.WriteString(helpStyle.Render("No public keys found. Please add one first."))
	} else {
		for i, key := range m.keys {
			_, isAssigned := m.assignedKeys[key.ID]
			checkbox := "[ ]"
			if isAssigned {
				checkbox = "[x]"
			}

			line := fmt.Sprintf("%s %s (%s)", checkbox, key.Comment, key.Algorithm)

			if m.keyCursor == i {
				b.WriteString(selectedItemStyle.Render("» " + line))
			} else {
				b.WriteString(itemStyle.Render(line))
			}
			b.WriteString("\n")
		}
	}

	b.WriteString(helpStyle.Render("\n(space/enter to toggle, esc to go back)"))
	if m.status != "" {
		b.WriteString(helpStyle.Render("\n\n" + m.status))
	}

	return b.String()
}
