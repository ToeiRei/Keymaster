// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// package tui provides the terminal user interface for Keymaster.
// This file contains the logic for the public key creation form, which allows
// users to paste a raw public key and add it to the database.
package tui // import "github.com/toeirei/keymaster/internal/tui"

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/i18n"
	"github.com/toeirei/keymaster/internal/sshkey"
)

// publicKeyCreatedMsg is a message to signal that a key was created successfully
// and the view should return to the public key list.
type publicKeyCreatedMsg struct{}

// publicKeyFormModel holds the state for the public key creation form.
type publicKeyFormModel struct {
	focusIndex int             // 0 for input, 1 for checkbox
	input      textinput.Model // The text input for pasting the raw key.
	isGlobal   bool            // Whether the 'global' checkbox is checked.
	err        error           // Any error that occurred during submission.
}

// newPublicKeyFormModel creates a new, empty form model for adding a public key.
func newPublicKeyFormModel() publicKeyFormModel {
	ti := textinput.New()
	ti.Placeholder = i18n.T("public_key_form.placeholder")
	ti.Focus()
	// Allow substantially longer inputs to support large SSH public keys
	// (including long comments). 8 KiB should be more than enough.
	ti.CharLimit = 8192
	ti.Width = 80
	ti.Prompt = i18n.T("public_key_form.prompt")
	ti.TextStyle = focusedStyle
	ti.Cursor.Style = focusedStyle

	return publicKeyFormModel{
		focusIndex: 0,
		input:      ti,
	}
}

// Init initializes the form model, returning a command to start the cursor blinking.
func (m publicKeyFormModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages and updates the form model's state.
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

// View renders the public key form UI.
func (m publicKeyFormModel) View() string {
	title := mainTitleStyle.Render("✨ " + i18n.T("public_key_form.add_title"))
	header := lipgloss.NewStyle().Align(lipgloss.Center).Render(title)

	// Left pane: input and checkbox
	var leftItems []string
	leftItems = append(leftItems, m.input.View())
	leftItems = append(leftItems, "")

	checkbox := i18n.T("public_key_form.checkbox_unchecked")
	if m.isGlobal {
		checkbox = i18n.T("public_key_form.checkbox_checked")
	}
	if m.focusIndex == 1 {
		leftItems = append(leftItems, formSelectedItemStyle.Render(checkbox))
	} else {
		leftItems = append(leftItems, formItemStyle.Render(checkbox))
	}

	leftPane := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorSubtle).
		Padding(1, 2).
		Width(60).
		Render(lipgloss.JoinVertical(lipgloss.Left, leftItems...))

	// Right pane: error/info preview
	var rightItems []string
	if m.err != nil {
		rightItems = append(rightItems, statusMessageStyle.Render(fmt.Sprintf(i18n.T("public_key_form.error"), m.err)))
	} else {
		rightItems = append(rightItems, helpStyle.Render(i18n.T("public_key_form.info")))
	}
	rightPane := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorSubtle).
		Padding(1, 2).
		Width(40).
		MarginLeft(2).
		Render(lipgloss.JoinVertical(lipgloss.Left, rightItems...))

	mainArea := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	// Help/footer line always at the bottom
	footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Background(lipgloss.Color("236")).Padding(0, 1).Italic(true)
	helpLine := footerStyle.Render(i18n.T("public_key_form.help"))

	return lipgloss.JoinVertical(lipgloss.Left, header, "\n", mainArea, "\n", helpLine)
}
