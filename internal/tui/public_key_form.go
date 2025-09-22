package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/sshkey"
)

// A message to signal that a key was created and we should go back to the list.
type publicKeyCreatedMsg struct{}

type publicKeyFormModel struct {
	focusIndex int // 0 for input, 1 for checkbox
	input      textinput.Model
	isGlobal   bool
	err        error
}

func newPublicKeyFormModel() publicKeyFormModel {
	ti := textinput.New()
	ti.Placeholder = "ssh-ed25519 AAAA... user@host"
	ti.Focus()
	ti.CharLimit = 1024
	ti.Width = 80
	ti.Prompt = "Paste Public Key: "
	ti.TextStyle = focusedStyle
	ti.Cursor.Style = focusedStyle

	return publicKeyFormModel{
		focusIndex: 0,
		input:      ti,
	}
}

func (m publicKeyFormModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m publicKeyFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle focus and global events first
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return backToListMsg{} }
		case "tab", "shift+tab", "up", "down":
			if m.focusIndex == 0 {
				m.focusIndex = 1
				m.input.Blur()
			} else {
				m.focusIndex = 0
				cmd = m.input.Focus()
			}
			return m, cmd

		case "enter":
			// If checkbox is focused, it's a toggle.
			if m.focusIndex == 1 {
				m.isGlobal = !m.isGlobal
				return m, nil
			}
			// If input is focused, it's a submit.
			rawKey := m.input.Value()
			alg, keyData, comment, err := sshkey.Parse(rawKey)
			if err != nil {
				m.err = err
				return m, nil
			}

			if comment == "" {
				m.err = fmt.Errorf("key must have a comment")
				return m, nil
			}

			if err := db.AddPublicKey(alg, keyData, comment, m.isGlobal); err != nil {
				m.err = err
				return m, nil
			}
			return m, func() tea.Msg { return publicKeyCreatedMsg{} }
		}
	}

	// Update the focused component
	if m.focusIndex == 0 {
		m.input, cmd = m.input.Update(msg)
	} else {
		// The checkbox is toggled with space when focused
		if key, ok := msg.(tea.KeyMsg); ok && key.String() == " " {
			m.isGlobal = !m.isGlobal
		}
	}
	return m, cmd
}

func (m publicKeyFormModel) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("✨ Add New Public Key"))
	b.WriteString("\n\n")
	b.WriteString(m.input.View())
	b.WriteString("\n\n")

	checkbox := "[ ] Set as Global Key (will be deployed to all accounts)"
	if m.isGlobal {
		checkbox = "[x] Set as Global Key (will be deployed to all accounts)"
	}

	if m.focusIndex == 1 {
		b.WriteString(selectedItemStyle.Render(checkbox))
	} else {
		b.WriteString(itemStyle.Render(checkbox))
	}

	if m.err != nil {
		b.WriteString(helpStyle.Render(fmt.Sprintf("\n\nError: %v", m.err)))
	}

	b.WriteString(helpStyle.Render("\n\n(tab to navigate, space/enter to toggle checkbox, enter on input to submit, esc to cancel)"))

	return b.String()
}
