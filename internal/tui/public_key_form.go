// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// package tui provides the terminal user interface for Keymaster.
// This file contains the logic for the public key creation form, which allows
// users to paste a raw public key and add it to the database.
package tui // import "github.com/toeirei/keymaster/internal/tui"

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/internal/i18n"
	"github.com/toeirei/keymaster/internal/sshkey"
	"github.com/toeirei/keymaster/internal/tui/adapters"
)

// publicKeyCreatedMsg is a message to signal that a key was created successfully
// and the view should return to the public key list.
type publicKeyCreatedMsg struct{}

// publicKeyFormModel holds the state for the public key creation form.
type publicKeyFormModel struct {
	focusIndex int             // 0 for input, 1 for expiration, 2 for checkbox
	input      textinput.Model // The text input for pasting the raw key.
	expInput   textinput.Model // The text input for optional expiration date.
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

	ei := textinput.New()
	ei.Placeholder = "YYYY-MM-DD or RFC3339"
	ei.CharLimit = 64
	ei.Width = 30
	ei.Prompt = "Expires: "
	ei.TextStyle = focusedStyle

	return publicKeyFormModel{
		focusIndex: 0,
		input:      ti,
		expInput:   ei,
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
			// cycle focus through input -> expiration -> checkbox
			m.input.Blur()
			m.expInput.Blur()
			m.focusIndex = (m.focusIndex + 1) % 3
			switch m.focusIndex {
			case 0:
				cmd = m.input.Focus()
			case 1:
				cmd = m.expInput.Focus()
			case 2:
				// checkbox; no focus command
			}
			return m, cmd

		case "enter":
			// If checkbox is focused, it's a toggle.
			if m.focusIndex == 2 {
				m.isGlobal = !m.isGlobal
				return m, nil
			}
			// If input or expiration is focused, submit.
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

			// parse expiration field if present
			var expiresAt time.Time
			expVal := strings.TrimSpace(m.expInput.Value())
			if expVal != "" {
				t, err := parseExpiryInput(expVal)
				if err != nil {
					m.err = fmt.Errorf("invalid expiration format; use YYYY-MM-DD or RFC3339")
					return m, nil
				}
				expiresAt = t
			}

			mgr := adapters.DefaultKeyManager()
			if mgr == nil {
				m.err = fmt.Errorf("no key manager available")
				return m, nil
			}
			if err := mgr.AddPublicKey(alg, keyData, comment, m.isGlobal, expiresAt); err != nil {
				m.err = err
				return m, nil
			}
			return m, func() tea.Msg { return publicKeyCreatedMsg{} }
		}
	}

	// Update the focused component
	if m.focusIndex == 0 {
		m.input, cmd = m.input.Update(msg)
	} else if m.focusIndex == 1 {
		m.expInput, cmd = m.expInput.Update(msg)
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

	// Left pane: input, expiration, and checkbox
	var leftItems []string
	leftItems = append(leftItems, m.input.View())
	leftItems = append(leftItems, "")
	// expiration input row
	leftItems = append(leftItems, m.expInput.View())
	leftItems = append(leftItems, "")

	checkbox := i18n.T("public_key_form.checkbox_unchecked")
	if m.isGlobal {
		checkbox = i18n.T("public_key_form.checkbox_checked")
	}
	if m.focusIndex == 2 {
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

// parseExpiryInput parses expiration strings (RFC3339 or YYYY-MM-DD).
func parseExpiryInput(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	if t2, err := time.Parse("2006-01-02", s); err == nil {
		return time.Date(t2.Year(), t2.Month(), t2.Day(), 0, 0, 0, 0, time.UTC), nil
	}
	return time.Time{}, fmt.Errorf("invalid expiry format")
}
