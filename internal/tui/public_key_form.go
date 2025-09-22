package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/internal/db"
)

// A message to signal that a key was created and we should go back to the list.
type publicKeyCreatedMsg struct{}

type publicKeyFormModel struct {
	input textinput.Model
	err   error
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
		input: ti,
	}
}

func (m publicKeyFormModel) Init() tea.Cmd {
	return textinput.Blink
}

// parsePublicKey splits a raw public key string into its components.
func parsePublicKey(rawKey string) (algorithm, keyData, comment string, err error) {
	parts := strings.Fields(rawKey)
	if len(parts) < 2 {
		err = fmt.Errorf("invalid public key format: must have at least algorithm and key data")
		return
	}

	algorithm = parts[0]
	keyData = parts[1]
	if len(parts) > 2 {
		comment = strings.Join(parts[2:], " ")
	}

	// Basic validation
	if !strings.HasPrefix(algorithm, "ssh-") && !strings.HasPrefix(algorithm, "ecdsa-") {
		err = fmt.Errorf("invalid key type: does not look like a valid SSH key algorithm")
		return
	}

	return
}

func (m publicKeyFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		// Go back to the keys list.
		case "esc":
			return m, func() tea.Msg { return backToKeysMsg{} }

		case "enter":
			rawKey := m.input.Value()
			alg, keyData, comment, err := parsePublicKey(rawKey)
			if err != nil {
				m.err = err
				return m, nil
			}

			if err := db.AddPublicKey(alg, keyData, comment); err != nil {
				m.err = err
				return m, nil
			}
			// Signal that we're done.
			return m, func() tea.Msg { return publicKeyCreatedMsg{} }
		}
	}

	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m publicKeyFormModel) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("✨ Add New Public Key"))
	b.WriteString("\n\n")
	b.WriteString(m.input.View())

	if m.err != nil {
		b.WriteString(helpStyle.Render(fmt.Sprintf("\n\nError: %v", m.err)))
	}

	b.WriteString(helpStyle.Render("\n\n(paste key and press enter to submit, esc to cancel)"))

	return b.String()
}
