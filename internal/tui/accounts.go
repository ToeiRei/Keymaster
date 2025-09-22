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

// A message to signal that we should go back to the accounts list from the form.
type backToAccountsMsg struct{}

type accountsViewState int

const (
	accountsListView accountsViewState = iota
	accountsFormView
)

// accountsModel is the model for the account management view.
type accountsModel struct {
	state    accountsViewState
	form     accountFormModel
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
	var cmd tea.Cmd

	// Delegate updates to the form if it's active.
	if m.state == accountsFormView {
		// If the form signals an account was created, switch back to the list and refresh.
		if _, ok := msg.(accountCreatedMsg); ok {
			m.state = accountsListView
			m.status = "Successfully added new account."
			m.accounts, m.err = db.GetAllAccounts()
			return m, nil
		}
		// If the form signals to go back, just switch the view.
		if _, ok := msg.(backToAccountsMsg); ok {
			m.state = accountsListView
			m.status = "" // Clear any status
			return m, nil
		}

		var newFormModel tea.Model
		newFormModel, cmd = m.form.Update(msg)
		m.form = newFormModel.(accountFormModel)
		return m, cmd
	}

	// --- This is the list view update logic ---
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

		// Toggle active status.
		case "t":
			if len(m.accounts) > 0 {
				accToToggle := m.accounts[m.cursor]
				if err := db.ToggleAccountStatus(accToToggle.ID); err != nil {
					m.err = err
				} else {
					// Refresh the list after toggling.
					m.status = fmt.Sprintf("Toggled status for: %s", accToToggle.String())
					m.accounts, m.err = db.GetAllAccounts()
				}
			}
			return m, nil

		// Switch to the form view to add a new account.
		case "a":
			m.state = accountsFormView
			m.form = newAccountFormModel()
			m.status = "" // Clear status before showing form
			return m, m.form.Init()
		}
	}
	return m, nil
}

func (m accountsModel) View() string {
	// If we're in the form view, render that instead.
	if m.state == accountsFormView {
		return m.form.View()
	}

	// --- This is the list view rendering ---
	var b strings.Builder

	b.WriteString(titleStyle.Render("🔑 Manage Accounts"))
	b.WriteString("\n\n")

	if m.err != nil {
		b.WriteString(fmt.Sprintf("Error: %v\n", m.err))
		return b.String()
	}

	// Find the longest username for alignment purposes.
	maxUserLen := 0
	for _, acc := range m.accounts {
		if len(acc.Username) > maxUserLen {
			maxUserLen = len(acc.Username)
		}
	}

	for i, acc := range m.accounts {
		userPart := fmt.Sprintf("%*s", maxUserLen, acc.Username)
		hostPart := fmt.Sprintf("@%s", acc.Hostname)

		if m.cursor == i {
			line := "» " + userPart + hostPart
			b.WriteString(selectedItemStyle.Render(line))
		} else {
			line := userPart + hostPart
			if acc.IsActive {
				b.WriteString(itemStyle.Render(line))
			} else {
				b.WriteString(inactiveItemStyle.Render(line))
			}
		}
		b.WriteString("\n")
	}

	if len(m.accounts) == 0 {
		b.WriteString(helpStyle.Render("No accounts found. Press 'a' to add one."))
	}

	b.WriteString(helpStyle.Render("\n(a)dd, (d)elete, (t)oggle active, (q)uit to menu"))
	if m.status != "" {
		b.WriteString(helpStyle.Render("\n\n" + m.status))
	}

	return b.String()
}
