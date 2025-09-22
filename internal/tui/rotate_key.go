package tui

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
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
)

type rotateKeyModel struct {
	state        rotateState
	hasSystemKey bool
	newPublicKey string
	newKeySerial int
	status       string
	err          error
}

func newRotateKeyModel() rotateKeyModel {
	m := rotateKeyModel{state: rotateStateChecking}
	hasKey, err := db.HasSystemKeys()
	if err != nil {
		m.err = err
		return m
	}
	m.hasSystemKey = hasKey
	if m.hasSystemKey {
		m.state = rotateStateReadyToRotate
	} else {
		m.state = rotateStateReadyToGenerate
	}
	return m
}

func (m rotateKeyModel) Init() tea.Cmd {
	return nil
}

func (m rotateKeyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			// Allow quitting from any state except during generation
			if m.state != rotateStateGenerating {
				return m, func() tea.Msg { return backToMenuMsg{} }
			}
		case "y":
			if m.state == rotateStateReadyToGenerate {
				m.state = rotateStateGenerating
				// Return a command to perform the generation
				return m, generateInitialKey
			}
		}
	// This message is sent by the generateInitialKey command on completion
	case initialKeyGeneratedMsg:
		m.state = rotateStateGenerated
		m.err = msg.err
		m.newPublicKey = msg.publicKey
		m.newKeySerial = msg.serial
	}
	return m, nil
}

func (m rotateKeyModel) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("🔑 System Key Management"))
	b.WriteString("\n\n")

	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	switch m.state {
	case rotateStateChecking:
		b.WriteString("Checking for existing system key...")
	case rotateStateReadyToGenerate:
		b.WriteString("No Keymaster system key found in the database.\n\n")
		b.WriteString("This key is required for Keymaster to connect to managed hosts.\n")
		b.WriteString("Would you like to generate the initial system key now? (y/n)")
	case rotateStateReadyToRotate:
		b.WriteString("An active system key already exists.\n\n")
		b.WriteString(helpStyle.Render("Key rotation is a multi-step process and is not yet fully implemented.\n"))
		b.WriteString(helpStyle.Render("Press 'q' to return to the main menu."))
	case rotateStateGenerating:
		b.WriteString("Generating new ed25519 key pair, please wait...")
	case rotateStateGenerated:
		b.WriteString(fmt.Sprintf("✅ Successfully generated and saved system key with serial #%d.\n\n", m.newKeySerial))
		b.WriteString(selectedItemStyle.Render("🚨 BOOTSTRAP ACTION REQUIRED 🚨\n\n"))
		b.WriteString("You must now manually add the following public key to the authorized_keys file\n")
		b.WriteString("for every account you intend to manage with Keymaster:\n\n")
		b.WriteString(fmt.Sprintf("    %s\n\n", m.newPublicKey))
		b.WriteString(helpStyle.Render("Press 'q' to return to the main menu."))
	}

	return b.String()
}

// --- Commands and Messages ---

type initialKeyGeneratedMsg struct {
	publicKey string
	serial    int
	err       error
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
