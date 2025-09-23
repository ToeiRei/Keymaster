package tui

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/internal/crypto/ssh"
	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/i18n"
)

type rotateState int

const (
	rotateStateChecking rotateState = iota
	rotateStateReadyToGenerate
	rotateStateReadyToRotate
	rotateStateGenerating
	rotateStateGenerated
	rotateStateRotating
	rotateStateRotated
)

type rotateKeyModel struct {
	state        rotateState
	newPublicKey string
	newKeySerial int
	err          error
	// For confirmation modal
	isConfirmingGenerate bool
	isConfirmingRotate   bool
	confirmCursor        int
	width, height        int
}

func newRotateKeyModel() *rotateKeyModel {
	m := &rotateKeyModel{state: rotateStateChecking, confirmCursor: 0} // Default to No
	hasKey, err := db.HasSystemKeys()
	if err != nil {
		m.err = err
		return m
	}

	if hasKey {
		m.isConfirmingRotate = true
	} else {
		m.isConfirmingGenerate = true
	}
	return m
}

func (m *rotateKeyModel) Init() tea.Cmd {
	return nil
}

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
		return errorStyle.Render(fmt.Sprintf(i18n.T("rotate_key.error"), m.err))
	}

	switch m.state {
	case rotateStateChecking:
		b.WriteString("Checking for existing system key...")
	case rotateStateGenerating:
		b.WriteString(specialStyle.Render(i18n.T("rotate_key.generating")))
	case rotateStateGenerated:
		b.WriteString(successStyle.Render(fmt.Sprintf(i18n.T("rotate_key.generated"), m.newKeySerial)))
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
		b.WriteString(successStyle.Render(fmt.Sprintf(i18n.T("rotate_key.rotated"), m.newKeySerial)))
		b.WriteString("\n\n")
		b.WriteString(specialStyle.Render(i18n.T("rotate_key.deploy_reminder") + "\n"))
		b.WriteString(helpStyle.Render(i18n.T("rotate_key.help_done")))
	}

	return b.String()
}

// --- Commands and Messages ---

type initialKeyGeneratedMsg struct {
	publicKey string
	serial    int
	err       error
}

type keyRotatedMsg struct {
	serial int
	err    error
}

// generateInitialKey is a tea.Cmd that performs the key generation and DB write.
func generateInitialKey() tea.Msg {
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return initialKeyGeneratedMsg{err: fmt.Errorf("failed to generate key pair: %w", err)}
	}

	sshPubKey, err := ssh.NewPublicKey(pubKey)
	if err != nil {
		return initialKeyGeneratedMsg{err: fmt.Errorf("failed to create SSH public key: %w", err)}
	}
	pubKeyBytes := ssh.MarshalAuthorizedKey(sshPubKey)
	publicKeyString := fmt.Sprintf("%s keymaster-system-key", strings.TrimSpace(string(pubKeyBytes)))

	pemBlock, err := ssh.MarshalEd25519PrivateKey(privKey, "")
	if err != nil {
		return initialKeyGeneratedMsg{err: fmt.Errorf("failed to marshal private key: %w", err)}
	}
	privateKeyString := string(pem.EncodeToMemory(pemBlock))

	serial, err := db.CreateSystemKey(publicKeyString, privateKeyString)
	if err != nil {
		return initialKeyGeneratedMsg{err: fmt.Errorf("failed to save system key to database: %w", err)}
	}

	return initialKeyGeneratedMsg{publicKey: publicKeyString, serial: serial}
}

// performRotation is a tea.Cmd that generates a new key and performs the DB rotation.
func performRotation() tea.Msg {
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return keyRotatedMsg{err: fmt.Errorf("failed to generate key pair: %w", err)}
	}

	sshPubKey, err := ssh.NewPublicKey(pubKey)
	if err != nil {
		return keyRotatedMsg{err: fmt.Errorf("failed to create SSH public key: %w", err)}
	}
	pubKeyBytes := ssh.MarshalAuthorizedKey(sshPubKey)
	publicKeyString := fmt.Sprintf("%s keymaster-system-key", strings.TrimSpace(string(pubKeyBytes)))

	pemBlock, err := ssh.MarshalEd25519PrivateKey(privKey, "")
	if err != nil {
		return keyRotatedMsg{err: fmt.Errorf("failed to marshal private key: %w", err)}
	}
	privateKeyString := string(pem.EncodeToMemory(pemBlock))

	serial, err := db.RotateSystemKey(publicKeyString, privateKeyString)
	if err != nil {
		return keyRotatedMsg{err: fmt.Errorf("failed to save rotated system key to database: %w", err)}
	}

	return keyRotatedMsg{serial: serial}
}
