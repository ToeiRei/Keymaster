package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/internal/db"
)

// A simple style for focused text inputs.
var focusedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("170"))

// A message to signal that an account was created and we should go back to the list.
type accountCreatedMsg struct{}

type accountFormModel struct {
	focusIndex int
	inputs     []textinput.Model
	err        error
}

func newAccountFormModel() accountFormModel {
	m := accountFormModel{
		inputs: make([]textinput.Model, 2),
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
			t.Placeholder = "user" // A more descriptive placeholder
			t.Focus()
			t.TextStyle = focusedStyle
		case 1:
			t.Prompt = "Hostname: "
			t.Placeholder = "www.example.com"
		}
		m.inputs[i] = t
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
			return m, func() tea.Msg { return backToAccountsMsg{} }

		// Set focus to next input
		case "tab", "shift+tab", "enter", "up", "down":
			s := msg.String()

			// Did the user press enter while the submit button was focused?
			// If so, create the account.
			if s == "enter" && m.focusIndex == len(m.inputs) {
				username := m.inputs[0].Value()
				hostname := m.inputs[1].Value()

				if username == "" || hostname == "" {
					m.err = fmt.Errorf("username and hostname cannot be empty")
					return m, nil
				}

				if err := db.AddAccount(username, hostname); err != nil {
					m.err = err
					return m, nil
				}
				// Signal that we're done.
				return m, func() tea.Msg { return accountCreatedMsg{} }
			}

			// Cycle focus
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

			cmds := make([]tea.Cmd, len(m.inputs))
			for i := 0; i <= len(m.inputs)-1; i++ {
				if i == m.focusIndex {
					cmds[i] = m.inputs[i].Focus()
					m.inputs[i].TextStyle = focusedStyle
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

	b.WriteString(titleStyle.Render("✨ Add New Account"))
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
