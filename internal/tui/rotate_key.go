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

func newRotateKeyModel() rotateKeyModel {
	m := rotateKeyModel{state: rotateStateChecking, confirmCursor: 0} // Default to No
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

func (m rotateKeyModel) Init() tea.Cmd {
	return nil
}

func (m rotateKeyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func (m rotateKeyModel) View() string {
	if m.isConfirmingGenerate {
		return m.viewConfirmationGenerate()
	}
	if m.isConfirmingRotate {
		return m.viewConfirmationRotate()
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("🔑 System Key Management"))

	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	switch m.state {
	// This state is now so brief it will likely never be seen.
	case rotateStateChecking:
		b.WriteString("Checking for existing system key...")
	case rotateStateGenerating:
		b.WriteString("Generating new ed25519 key pair, please wait...")
	case rotateStateGenerated:
		b.WriteString(successStyle.Render(fmt.Sprintf("✅ Successfully generated and saved system key with serial #%d.", m.newKeySerial)))
		b.WriteString("\n\n")

		var box strings.Builder
		box.WriteString(lipgloss.NewStyle().Foreground(colorSpecial).Bold(true).Render("🚨 BOOTSTRAP ACTION REQUIRED 🚨"))
		box.WriteString("\n\n")
		box.WriteString("You must now manually add the following public key to the `authorized_keys` file for every account you intend to manage with Keymaster:\n\n")
		box.WriteString(lipgloss.NewStyle().Background(lipgloss.Color("235")).Padding(0, 1).Render(m.newPublicKey))

		b.WriteString(lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorSpecial).Padding(1).Render(box.String()))

		b.WriteString("\n\n" + helpStyle.Render("Press 'q' to return to the main menu."))
	case rotateStateRotating:
		b.WriteString("Rotating system key, please wait...")
	case rotateStateRotated:
		b.WriteString(successStyle.Render(fmt.Sprintf("✅ Successfully rotated system key. The new active key is serial #%d.", m.newKeySerial)))
		b.WriteString("\n\n")
		b.WriteString("Deploy to your fleet to apply the new key.\n")
		b.WriteString(helpStyle.Render("Press 'q' to return to the main menu."))
	}

	return b.String()
}

func (m rotateKeyModel) viewConfirmationRotate() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("⚙️ Confirm System Key Rotation"))

	question := "Are you sure you want to rotate the system key?\n\nThis will deactivate the current key and create a new one.\nHosts will need to be redeployed to get the new key."
	b.WriteString(question)
	b.WriteString("\n\n")

	var yesButton, noButton string
	if m.confirmCursor == 1 { // Yes
		yesButton = activeButtonStyle.Render("Yes, Rotate")
		noButton = buttonStyle.Render("No, Cancel")
	} else { // No
		yesButton = buttonStyle.Render("Yes, Rotate")
		noButton = activeButtonStyle.Render("No, Cancel")
	}

	buttons := lipgloss.JoinHorizontal(lipgloss.Top, noButton, "  ", yesButton)
	b.WriteString(buttons)

	b.WriteString("\n" + helpStyle.Render("\n(left/right to navigate, enter to confirm, esc to cancel)"))

	// Center the whole dialog
	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		dialogBoxStyle.Render(b.String()),
	)
}

func (m rotateKeyModel) viewConfirmationGenerate() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("⚙️ Generate Initial System Key"))

	question := "No Keymaster system key found.\n\nThis key is required for Keymaster to connect to managed hosts.\n\nWould you like to generate one now?"
	b.WriteString(question)
	b.WriteString("\n\n")

	var yesButton, noButton string
	if m.confirmCursor == 1 { // Yes
		yesButton = activeButtonStyle.Render("Yes, Generate")
		noButton = buttonStyle.Render("No, Cancel")
	} else { // No
		yesButton = buttonStyle.Render("Yes, Generate")
		noButton = activeButtonStyle.Render("No, Cancel")
	}

	buttons := lipgloss.JoinHorizontal(lipgloss.Top, noButton, "  ", yesButton)
	b.WriteString(buttons)

	b.WriteString("\n" + helpStyle.Render("\n(left/right to navigate, enter to confirm, esc to cancel)"))

	// Center the whole dialog
	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		dialogBoxStyle.Render(b.String()),
	)
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
