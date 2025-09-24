// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// package tui provides the terminal user interface for Keymaster.
// This file contains the logic for the system key rotation view, which handles
// both the initial generation of a system key and the rotation of an existing one.
package tui // import "github.com/toeirei/keymaster/internal/tui"

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/internal/crypto/ssh"
	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/i18n"
)

// rotateState represents the current view within the key rotation workflow.
type rotateState int

const (
	// rotateStateChecking is the initial state where the model checks if a system key exists.
	rotateStateChecking rotateState = iota
	// rotateStateReadyToGenerate is an intermediate state before showing the generation confirmation.
	rotateStateReadyToGenerate
	// rotateStateReadyToRotate is an intermediate state before showing the rotation confirmation.
	rotateStateReadyToRotate
	// rotateStateGenerating shows a "generating..." message while the key is being created.
	rotateStateGenerating
	// rotateStateGenerated shows the result of the initial key generation, including the public key.
	rotateStateGenerated
	// rotateStateRotating shows a "rotating..." message while the key is being rotated.
	rotateStateRotating
	// rotateStateRotated shows the result of a successful key rotation.
	rotateStateRotated
)

// rotateKeyModel holds the state for the system key rotation view.
// It manages the workflow from checking for an existing key to confirming and
// performing the generation or rotation.
type rotateKeyModel struct {
	state        rotateState
	newPublicKey string // The public key generated during initial creation.
	newKeySerial int    // The serial number of the newly created/rotated key.
	err          error
	// For confirmation modal
	isConfirmingGenerate bool // True if the "generate initial key" modal is active.
	isConfirmingRotate   bool // True if the "rotate existing key" modal is active.
	confirmCursor        int  // 0 for No, 1 for Yes in the confirmation modal.
	width, height        int
}

// newRotateKeyModel creates a new model for the key rotation view.
// It immediately checks the database to see if a system key already exists
// and sets the appropriate confirmation state (generate vs. rotate).
func newRotateKeyModel() *rotateKeyModel {
	m := &rotateKeyModel{state: rotateStateChecking, confirmCursor: 0} // Default to No
	hasKey, err := db.HasSystemKeys()
	if err != nil {
		m.err = err
		return m
	}

	if hasKey {
		m.state = rotateStateReadyToRotate
		m.isConfirmingRotate = true
	} else {
		m.state = rotateStateReadyToGenerate
		m.isConfirmingGenerate = true
	}
	return m
}

// Init initializes the model.
func (m *rotateKeyModel) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model's state.
func (m *rotateKeyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	// Handle confirmation modals
	if m.isConfirmingGenerate || m.isConfirmingRotate {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "q", "esc":
				return m, func() tea.Msg { return backToMenuMsg{} }
			case "right", "tab", "l":
				m.confirmCursor = 1 // Yes
				return m, nil
			case "left", "shift+tab", "h":
				m.confirmCursor = 0 // No
				return m, nil
			case "enter":
				if m.confirmCursor == 0 { // "No" is selected
					return m, func() tea.Msg { return backToMenuMsg{} }
				}

				// "Yes" is selected
				if m.isConfirmingGenerate {
					m.state = rotateStateGenerating
					m.isConfirmingGenerate = false
					return m, generateInitialKey
				}
				if m.isConfirmingRotate {
					m.state = rotateStateRotating
					m.isConfirmingRotate = false
					return m, performRotation
				}
			}
		}
		return m, nil
	}

	// Handle other states (generating, rotating, done)
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			// Allow quitting from any state except during generation/rotation
			if m.state != rotateStateGenerating && m.state != rotateStateRotating {
				return m, func() tea.Msg { return backToMenuMsg{} }
			}
		}
	// This message is sent by the generateInitialKey command on completion
	case initialKeyGeneratedMsg:
		m.state = rotateStateGenerated
		m.err = msg.err
		m.newPublicKey = msg.publicKey
		m.newKeySerial = msg.serial
	case keyRotatedMsg:
		m.state = rotateStateRotated
		m.err = msg.err
		m.newKeySerial = msg.serial
	}
	return m, nil
}

// viewConfirmationRotate renders the modal dialog for confirming a key rotation.
func (m *rotateKeyModel) viewConfirmationRotate() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(i18n.T("rotate_key.confirm_rotate_title")))
	b.WriteString("\n\n")
	b.WriteString(specialStyle.Render(i18n.T("rotate_key.confirm_rotate_question")))
	b.WriteString("\n\n")

	var yesButton, noButton string
	if m.confirmCursor == 1 { // Yes
		yesButton = activeButtonStyle.Render(i18n.T("rotate_key.yes_rotate"))
		noButton = buttonStyle.Render(i18n.T("rotate_key.no_cancel"))
	} else { // No
		yesButton = buttonStyle.Render(i18n.T("rotate_key.yes_rotate"))
		noButton = activeButtonStyle.Render(i18n.T("rotate_key.no_cancel"))
	}

	buttons := lipgloss.JoinHorizontal(lipgloss.Top, noButton, "  ", yesButton)
	b.WriteString(buttons)

	b.WriteString("\n" + helpStyle.Render(i18n.T("rotate_key.help_modal")))

	return lipgloss.Place(m.width, m.height,
		lipgloss.Left, lipgloss.Center,
		dialogBoxStyle.Render(b.String()),
	)
}

// viewConfirmationGenerate renders the modal dialog for generating the initial system key.
func (m *rotateKeyModel) viewConfirmationGenerate() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(i18n.T("rotate_key.confirm_generate_title")))
	b.WriteString("\n\n")
	b.WriteString(specialStyle.Render(i18n.T("rotate_key.confirm_generate_question")))
	b.WriteString("\n\n")

	var yesButton, noButton string
	if m.confirmCursor == 1 { // Yes
		yesButton = activeButtonStyle.Render(i18n.T("rotate_key.yes_generate"))
		noButton = buttonStyle.Render(i18n.T("rotate_key.no_cancel"))
	} else { // No
		yesButton = buttonStyle.Render(i18n.T("rotate_key.yes_generate"))
		noButton = activeButtonStyle.Render(i18n.T("rotate_key.no_cancel"))
	}

	buttons := lipgloss.JoinHorizontal(lipgloss.Top, noButton, "  ", yesButton)
	b.WriteString(buttons)

	b.WriteString("\n" + helpStyle.Render(i18n.T("rotate_key.help_modal")))

	return lipgloss.Place(m.width, m.height,
		lipgloss.Left, lipgloss.Center,
		dialogBoxStyle.Render(b.String()),
	)
}

// View renders the key rotation UI based on the current model state.
func (m *rotateKeyModel) View() string {
	if m.isConfirmingGenerate {
		return m.viewConfirmationGenerate()
	}
	if m.isConfirmingRotate {
		return m.viewConfirmationRotate()
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("🔑 " + i18n.T("rotate_key.title")))

	if m.err != nil {
		return errorStyle.Render(i18n.T("rotate_key.error", m.err))
	}

	switch m.state {
	case rotateStateChecking:
		b.WriteString(i18n.T("rotate_key.checking"))
	case rotateStateGenerating:
		b.WriteString(specialStyle.Render(i18n.T("rotate_key.generating")))
	case rotateStateGenerated:
		b.WriteString(successStyle.Render(i18n.T("rotate_key.generated", m.newKeySerial)))
		b.WriteString("\n\n")

		var box strings.Builder
		box.WriteString(lipgloss.NewStyle().Foreground(colorSpecial).Bold(true).Render(i18n.T("rotate_key.bootstrap_header")))
		box.WriteString("\n\n")
		box.WriteString(i18n.T("rotate_key.bootstrap_body") + "\n\n")
		box.WriteString(lipgloss.NewStyle().Background(lipgloss.Color("235")).Padding(0, 1).Render(m.newPublicKey))

		b.WriteString(lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorSpecial).Padding(1).Render(box.String()))

		b.WriteString("\n\n" + helpStyle.Render(i18n.T("rotate_key.help_done")))
	case rotateStateRotating:
		b.WriteString(specialStyle.Render(i18n.T("rotate_key.rotating")))
	case rotateStateRotated:
		b.WriteString(successStyle.Render(i18n.T("rotate_key.rotated", m.newKeySerial)))
		b.WriteString("\n\n")
		b.WriteString(specialStyle.Render(i18n.T("rotate_key.deploy_reminder") + "\n"))
		b.WriteString(helpStyle.Render(i18n.T("rotate_key.help_done")))
	}

	return b.String()
}

// --- Commands and Messages ---

// initialKeyGeneratedMsg is a message sent when the initial key generation is complete.
type initialKeyGeneratedMsg struct {
	publicKey string
	serial    int
	err       error
}

// keyRotatedMsg is a message sent when the key rotation is complete.
type keyRotatedMsg struct {
	serial int
	err    error
}

// generateInitialKey is a tea.Cmd that performs the key generation and DB write.
// It sends an initialKeyGeneratedMsg when complete.
func generateInitialKey() tea.Msg {
	publicKeyString, privateKeyString, err := ssh.GenerateAndMarshalEd25519Key("keymaster-system-key")
	if err != nil {
		return initialKeyGeneratedMsg{err: fmt.Errorf(i18n.T("rotate_key.error_generate"), err)}
	}
	serial, err := db.CreateSystemKey(publicKeyString, privateKeyString)
	if err != nil {
		return initialKeyGeneratedMsg{err: fmt.Errorf(i18n.T("rotate_key.error_save"), err)}
	}

	return initialKeyGeneratedMsg{publicKey: publicKeyString, serial: serial}
}

// performRotation is a tea.Cmd that generates a new key and performs the DB rotation.
// It sends a keyRotatedMsg when complete.
func performRotation() tea.Msg {
	publicKeyString, privateKeyString, err := ssh.GenerateAndMarshalEd25519Key("keymaster-system-key")
	if err != nil {
		return keyRotatedMsg{err: fmt.Errorf(i18n.T("rotate_key.error_generate"), err)}
	}
	serial, err := db.RotateSystemKey(publicKeyString, privateKeyString)
	if err != nil {
		return keyRotatedMsg{err: fmt.Errorf(i18n.T("rotate_key.error_save_rotated"), err)}
	}

	return keyRotatedMsg{serial: serial}
}
