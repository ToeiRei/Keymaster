package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/i18n"
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
	ti.Placeholder = i18n.T("public_key_form.placeholder")
	ti.Focus()
	ti.CharLimit = 1024
	ti.Width = 80
	ti.Prompt = i18n.T("public_key_form.prompt")
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
