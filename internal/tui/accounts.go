package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/model"
)

// A message to signal that we should go back to the main menu.
type backToMenuMsg struct{}

// accountsModel is the model for the account management view.
type accountsModel struct {
	accounts []model.Account
	cursor   int
	status   string // For showing status messages like "Deleted..."
	err      error
}

func newAccountsModel() accountsModel {
	m := accountsModel{}
	var err error
	m.accounts, err = db.GetAllAccounts()
	if err != nil {
		m.err = err
	}
	return m
}

func (m accountsModel) Init() tea.Cmd {
	return nil
}

func (m accountsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		// Go back to the main menu.
		case "q", "esc":
			return m, func() tea.Msg { return backToMenuMsg{} }

		// Navigate up.
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		// Navigate down.
		case "down", "j":
			if m.cursor < len(m.accounts)-1 {
				m.cursor++
			}

		// Delete an account.
		case "d", "delete":
			if len(m.accounts) > 0 {
				accToDelete := m.accounts[m.cursor]
				if err := db.DeleteAccount(accToDelete.ID); err != nil {
					m.err = err
				} else {
					// Refresh the list after deletion.
					m.status = fmt.Sprintf("Deleted account: %s", accToDelete.String())
					m.accounts, m.err = db.GetAllAccounts()
					// Make sure cursor is not out of bounds.
					if m.cursor >= len(m.accounts) && len(m.accounts) > 0 {
						m.cursor = len(m.accounts) - 1
					}
				}
			}
			return m, nil

		// TODO: Add a new account.
		case "a":
			m.status = "Add functionality is not yet implemented."
			return m, nil
		}
	}
	return m, nil
}

func (m accountsModel) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("🔑 Manage Accounts"))
	b.WriteString("\n\n")

	if m.err != nil {
		b.WriteString(fmt.Sprintf("Error: %v\n", m.err))
		return b.String()
	}

	for i, acc := range m.accounts {
		if m.cursor == i {
			b.WriteString(selectedItemStyle.Render("» " + acc.String()))
		} else {
			b.WriteString(itemStyle.Render(acc.String()))
		}
		b.WriteString("\n")
	}

	if len(m.accounts) == 0 {
		b.WriteString(helpStyle.Render("No accounts found. Press 'a' to add one."))
	}

	b.WriteString(helpStyle.Render("\n(a)dd, (d)elete, (q)uit to menu"))
	if m.status != "" {
		b.WriteString(helpStyle.Render("\n\n" + m.status))
	}

	return b.String()
}
