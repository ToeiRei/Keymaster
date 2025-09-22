package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/model"
)

// A simple style for focused text inputs.
var focusedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("170"))
var disabledStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

// A message to signal that an account was modified (created or updated).
type accountModifiedMsg struct {
	isNew    bool
	hostname string
}

type accountFormModel struct {
	focusIndex     int
	inputs         []textinput.Model // 0: user, 1: host, 2: label
	err            error
	editingAccount *model.Account // If not nil, we are in edit mode.
}

func newAccountFormModel(accountToEdit *model.Account) accountFormModel {
	m := accountFormModel{
		inputs: make([]textinput.Model, 3),
	}

	var t textinput.Model
	for i := range m.inputs {
		t = textinput.New()
		t.Cursor.Style = focusedStyle
		t.CharLimit = 64
		t.Width = 40 // Give the input a fixed width

		switch i {
		case 0:
			t.Prompt = "Username: "
			t.Placeholder = "user"
		case 1:
			t.Prompt = "Hostname: "
			t.Placeholder = "www.example.com"
		case 2:
			t.Prompt = "Label (optional): "
			t.Placeholder = "prod-web-01"
		}
		m.inputs[i] = t
	}

	if accountToEdit != nil {
		m.editingAccount = accountToEdit
		m.inputs[0].SetValue(accountToEdit.Username)
		m.inputs[0].PromptStyle = disabledStyle
		m.inputs[0].TextStyle = disabledStyle
		m.inputs[1].SetValue(accountToEdit.Hostname)
		m.inputs[1].PromptStyle = disabledStyle
		m.inputs[1].TextStyle = disabledStyle
		m.inputs[2].SetValue(accountToEdit.Label)
		m.inputs[2].Focus()
		m.inputs[2].TextStyle = focusedStyle
		m.focusIndex = 2
	} else {
		m.inputs[0].Focus()
		m.inputs[0].TextStyle = focusedStyle
	}

	return m
}

func (m accountFormModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m accountFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		// Go back to the accounts list.
		case "esc":
			return m, func() tea.Msg { return backToListMsg{} }

		// Set focus to next input
		case "tab", "shift+tab", "enter", "up", "down":
			s := msg.String()

			// Did the user press enter while the submit button was focused?
			// If so, create the account.
			if s == "enter" && m.focusIndex == len(m.inputs) {
				if m.editingAccount != nil {
					// Update existing account
					label := m.inputs[2].Value()
					if err := db.UpdateAccountLabel(m.editingAccount.ID, label); err != nil {
						m.err = err
						return m, nil
					}
					// Signal that we're done.
					return m, func() tea.Msg { return accountModifiedMsg{isNew: false} }
				} else {
					// Add new account
					username := m.inputs[0].Value()
					hostname := m.inputs[1].Value()
					label := m.inputs[2].Value()

					if username == "" || hostname == "" {
						m.err = fmt.Errorf("username and hostname cannot be empty")
						return m, nil
					}

					if err := db.AddAccount(username, hostname, label); err != nil {
						m.err = err
						return m, nil
					}
					// Signal that we're done.
					return m, func() tea.Msg { return accountModifiedMsg{isNew: true, hostname: hostname} }
				}
			}

			// Cycle focus
			if m.editingAccount != nil {
				// In edit mode, only cycle between label and submit button
				if s == "up" || s == "shift+tab" {
					m.focusIndex--
					if m.focusIndex < 2 {
						m.focusIndex = len(m.inputs)
					}
				} else {
					m.focusIndex++
					if m.focusIndex > len(m.inputs) {
						m.focusIndex = 2
					}
				}
			} else {
				// In add mode, cycle through all fields
				if s == "up" || s == "shift+tab" {
					m.focusIndex--
				} else {
					m.focusIndex++
				}
				if m.focusIndex > len(m.inputs) {
					m.focusIndex = 0
				} else if m.focusIndex < 0 {
					m.focusIndex = len(m.inputs)
				}
			}

			cmds := make([]tea.Cmd, len(m.inputs))
			for i := 0; i <= len(m.inputs)-1; i++ {
				if i == m.focusIndex {
					cmds[i] = m.inputs[i].Focus()
					m.inputs[i].TextStyle = focusedStyle
					if m.editingAccount != nil && i < 2 {
						m.inputs[i].TextStyle = disabledStyle
					}
					continue
				}
				m.inputs[i].Blur()
				m.inputs[i].TextStyle = lipgloss.NewStyle()
			}

			return m, tea.Batch(cmds...)
		}
	}

	// Handle character input and blinking
	cmd := m.updateInputs(msg)
	return m, cmd
}

func (m *accountFormModel) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return tea.Batch(cmds...)
}

func (m accountFormModel) View() string {
	var b strings.Builder

	if m.editingAccount != nil {
		b.WriteString(titleStyle.Render("✏️ Edit Account"))
	} else {
		b.WriteString(titleStyle.Render("✨ Add New Account"))
	}
	b.WriteString("\n\n")

	for i := range m.inputs {
		b.WriteString(m.inputs[i].View())
		b.WriteString("\n")
	}

	button := itemStyle.Render("[ Submit ]")
	if m.focusIndex == len(m.inputs) {
		button = selectedItemStyle.Render("[ Submit ]")
	}
	b.WriteString(fmt.Sprintf("\n%s\n", button))

	if m.err != nil {
		b.WriteString(helpStyle.Render(fmt.Sprintf("\n\nError: %v", m.err)))
	}

	b.WriteString(helpStyle.Render("\n(tab to navigate, enter to submit, esc to cancel)"))

	return b.String()
}
