// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// package tui provides the terminal user interface for Keymaster.
// This file contains the logic for the public key management view, which allows
// users to list, add, delete, and see the usage of public keys.
package tui // import "github.com/toeirei/keymaster/internal/tui"

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	internalkey "github.com/toeirei/keymaster/internal/crypto/ssh"
	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/i18n"
	"github.com/toeirei/keymaster/internal/model"
	"golang.org/x/crypto/ssh"
)

// publicKeysViewState represents the current view within the public key management workflow.
type publicKeysViewState int

const (
	// publicKeysListView is the default view, showing a filterable list of all keys.
	publicKeysListView publicKeysViewState = iota
	// publicKeysFormView shows the form for adding a new public key.
	publicKeysFormView
	// publicKeysUsageView shows a report of which accounts a specific key is assigned to.
	publicKeysUsageView
)

// publicKeysModel holds the state for the public key management view.
// It manages the list of keys, the form for adding new keys, and the usage report view.
type publicKeysModel struct {
	state            publicKeysViewState
	form             publicKeyFormModel
	keys             []model.PublicKey // Master list
	displayedKeys    []model.PublicKey // Filtered list
	cursor           int
	status           string
	err              error
	usageReportKey   model.PublicKey
	usageReportAccts []model.Account
	filter           string
	isFiltering      bool
	// For delete confirmation
	isConfirmingDelete bool
	keyToDelete        model.PublicKey
	confirmCursor      int // 0 for No, 1 for Yes
	width, height      int
}

// newPublicKeysModel creates a new model for the public key view, pre-loading keys from the database.
func newPublicKeysModel() publicKeysModel {
	m := publicKeysModel{}
	var err error
	m.keys, err = db.GetAllPublicKeys()
	if err != nil {
		m.err = err
	}
	m.rebuildDisplayedKeys()
	return m
}

// Init initializes the model.
func (m publicKeysModel) Init() tea.Cmd {
	return nil
}

// rebuildDisplayedKeys constructs the list of keys to be displayed, applying the
// current filter text. It also ensures the cursor remains within bounds.
func (m *publicKeysModel) rebuildDisplayedKeys() {
	if m.filter == "" {
		m.displayedKeys = m.keys
	} else {
		m.displayedKeys = []model.PublicKey{}
		lowerFilter := strings.ToLower(m.filter)
		for _, key := range m.keys {
			if strings.Contains(strings.ToLower(key.Comment), lowerFilter) ||
				strings.Contains(strings.ToLower(key.Algorithm), lowerFilter) {
				m.displayedKeys = append(m.displayedKeys, key)
			}
		}
	}

	// Reset cursor if it's out of bounds
	if m.cursor >= len(m.displayedKeys) {
		if len(m.displayedKeys) > 0 {
			m.cursor = len(m.displayedKeys) - 1
		} else {
			m.cursor = 0
		}
	}
}

// Update handles messages and updates the model's state.
func (m publicKeysModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Handle window size messages first, as they affect layout.
	if sizeMsg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = sizeMsg.Width
		m.height = sizeMsg.Height
	}

	if m.state == publicKeysFormView {
		if _, ok := msg.(publicKeyCreatedMsg); ok {
			m.state = publicKeysListView
			m.status = "Successfully added new public key."
			m.keys, m.err = db.GetAllPublicKeys()
			m.rebuildDisplayedKeys()
			return m, nil
		}
		if _, ok := msg.(backToListMsg); ok {
			m.state = publicKeysListView
			m.status = ""
			return m, nil
		}
		var newFormModel tea.Model
		newFormModel, cmd = m.form.Update(msg)
		m.form = newFormModel.(publicKeyFormModel)
		return m, cmd
	}

	if m.state == publicKeysUsageView {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "q", "esc":
				m.state = publicKeysListView
				m.status = ""
				return m, nil
			}
		}
		return m, nil
	}

	// List view logic
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle delete confirmation
		if m.isConfirmingDelete {
			switch msg.String() {
			case "y":
				// Fallthrough to confirm
			case "n", "q", "esc":
				m.isConfirmingDelete = false
				m.status = "Deletion cancelled."
				return m, nil
			case "right", "tab", "l":
				m.confirmCursor = 1 // Yes
				return m, nil
			case "left", "shift+tab", "h":
				m.confirmCursor = 0 // No
				return m, nil
			case "enter":
				if m.confirmCursor == 1 { // Yes is selected
					if err := db.DeletePublicKey(m.keyToDelete.ID); err != nil {
						m.err = err
					} else {
						m.status = fmt.Sprintf("Deleted key: %s", m.keyToDelete.Comment)
						m.keys, m.err = db.GetAllPublicKeys()
						m.rebuildDisplayedKeys()
					}
				}
				m.isConfirmingDelete = false
				return m, nil
			}
			return m, nil
		}

		// If we are in filtering mode, capture all input for the filter.
		if m.isFiltering {
			switch msg.Type {
			case tea.KeyEsc:
				m.isFiltering = false
				m.filter = ""
				m.rebuildDisplayedKeys()
			case tea.KeyEnter:
				m.isFiltering = false
			case tea.KeyBackspace:
				if len(m.filter) > 0 {
					m.filter = m.filter[:len(m.filter)-1]
					m.rebuildDisplayedKeys()
				}
			case tea.KeyRunes:
				m.filter += string(msg.Runes)
				m.rebuildDisplayedKeys()
			}
			return m, nil
		}

		// Not in filtering mode, handle commands.
		switch msg.String() {
		case "/":
			m.isFiltering = true
			m.filter = "" // Start with a fresh filter
			m.rebuildDisplayedKeys()
			return m, nil
		case "q", "esc":
			if m.filter != "" && !m.isFiltering {
				m.filter = ""
				m.rebuildDisplayedKeys()
				return m, nil
			}
			return m, func() tea.Msg { return backToMenuMsg{} }
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.displayedKeys)-1 {
				m.cursor++
			}
		case "a":
			m.state = publicKeysFormView
			m.form = newPublicKeyFormModel()
			m.status = ""
			return m, m.form.Init()
		case "d", "delete":
			if len(m.displayedKeys) > 0 {
				m.keyToDelete = m.displayedKeys[m.cursor]
				m.isConfirmingDelete = true
				m.confirmCursor = 0 // Default to No
			}
			return m, nil
		case "g": // Toggle global status
			if len(m.displayedKeys) > 0 {
				keyToToggle := m.displayedKeys[m.cursor]
				if err := db.TogglePublicKeyGlobal(keyToToggle.ID); err != nil {
					m.err = err
				} else {
					m.status = fmt.Sprintf("Toggled global status for key: %s", keyToToggle.Comment)
					// Refresh the list to show the new status
					m.keys, m.err = db.GetAllPublicKeys()
					m.rebuildDisplayedKeys()
				}
			}
			return m, nil
		case "u": // Usage report
			if len(m.displayedKeys) > 0 {
				m.usageReportKey = m.displayedKeys[m.cursor]
				accounts, err := db.GetAccountsForKey(m.usageReportKey.ID)
				if err != nil {
					m.err = err
					return m, nil
				}
				m.usageReportAccts = accounts
				m.state = publicKeysUsageView
				m.status = ""
			}
			return m, nil
		}
	}
	return m, nil
}

// View renders the public key management UI based on the current model state.
func (m publicKeysModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	if m.isConfirmingDelete {
		return m.viewConfirmation()
	}

	switch m.state {
	case publicKeysFormView:
		return m.form.View()
	case publicKeysUsageView:
		return m.viewUsageReport()
	default: // publicKeysListView
		return m.viewKeyList()
	}
}

// viewConfirmation renders the modal dialog for confirming a key deletion.
func (m publicKeysModel) viewConfirmation() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("🗑️ Confirm Deletion"))

	question := fmt.Sprintf("Are you sure you want to delete the public key\n\n%s?", m.keyToDelete.Comment)
	b.WriteString(question)
	b.WriteString("\n\nThis will remove it from all accounts it is assigned to.")
	b.WriteString("\n\n")

	var yesButton, noButton string
	if m.confirmCursor == 1 { // Yes
		yesButton = activeButtonStyle.Render("Yes, Delete")
		noButton = buttonStyle.Render("No, Cancel")
	} else { // No
		yesButton = buttonStyle.Render("Yes, Delete")
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

// viewKeyList renders the main two-pane view with the key list and details.
func (m publicKeysModel) viewKeyList() string {
	title := mainTitleStyle.Render("🔑 " + i18n.T("public_keys.title"))
	header := lipgloss.NewStyle().Align(lipgloss.Center).Render(title)

	// List pane (left)
	var listItems []string
	for i, key := range m.displayedKeys {
		var globalMarker string
		if key.IsGlobal {
			globalMarker = "🌐 "
		}
		line := fmt.Sprintf("%s%s (%s)", globalMarker, key.Comment, key.Algorithm)
		if m.cursor == i {
			listItems = append(listItems, selectedItemStyle.Render("▸ "+line))
		} else {
			listItems = append(listItems, itemStyle.Render("  "+line))
		}
	}

	if len(m.displayedKeys) == 0 && m.filter == "" {
		listItems = append(listItems, helpStyle.Render(i18n.T("public_keys.empty")))
	} else if len(m.displayedKeys) == 0 && m.filter != "" {
		listItems = append(listItems, helpStyle.Render(i18n.T("public_keys.empty_filtered")))
	}

	listPaneTitle := lipgloss.NewStyle().Bold(true).Render(i18n.T("public_keys.list_title"))
	listPane := lipgloss.JoinVertical(lipgloss.Left, listPaneTitle, "", lipgloss.JoinVertical(lipgloss.Left, listItems...))

	paneStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorSubtle).
		Padding(1, 2)

	menuWidth := 48
	detailWidth := m.width - 4 - menuWidth - 2

	leftPane := paneStyle.Width(menuWidth).Render(listPane)

	// Details/status pane (right)
	var detailsItems []string
	if m.err != nil {
		detailsItems = append(detailsItems, helpStyle.Render(fmt.Sprintf(i18n.T("public_keys.error"), m.err)))
	} else if m.status != "" {
		detailsItems = append(detailsItems, statusMessageStyle.Render(m.status))
	}

	// Show details for the selected key
	if len(m.displayedKeys) > 0 && m.cursor < len(m.displayedKeys) {
		key := m.displayedKeys[m.cursor]
		detailsItems = append(detailsItems, "", helpStyle.Render(i18n.T("public_keys.detail_comment", key.Comment)))
		detailsItems = append(detailsItems, helpStyle.Render(i18n.T("public_keys.detail_algorithm", key.Algorithm)))
		detailsItems = append(detailsItems, helpStyle.Render(i18n.T("public_keys.detail_global", boolToYesNo(key.IsGlobal))))

		// Calculate and display the fingerprint on the fly.
		parsedKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(key.String()))
		if err == nil {
			fingerprint := internalkey.FingerprintSHA256(parsedKey)
			detailsItems = append(detailsItems, helpStyle.Render(i18n.T("public_keys.detail_fingerprint", fingerprint)))
		}
	}

	// Only show filter status if filtering
	if m.isFiltering {
		detailsItems = append(detailsItems, "", helpStyle.Render(fmt.Sprintf(i18n.T("public_keys.filtering"), m.filter)))
	}

	rightPane := paneStyle.Width(detailWidth).MarginLeft(2).Render(lipgloss.JoinVertical(lipgloss.Left, detailsItems...))

	mainArea := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	// Help/footer line always at the bottom
	footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Background(lipgloss.Color("236")).Padding(0, 1).Italic(true)
	var filterStatus string
	if m.isFiltering {
		filterStatus = fmt.Sprintf(i18n.T("public_keys.filtering"), m.filter)
	} else if m.filter != "" {
		filterStatus = fmt.Sprintf(i18n.T("public_keys.filter_active"), m.filter)
	} else {
		filterStatus = i18n.T("public_keys.filter_hint")
	}
	helpLine := footerStyle.Render(fmt.Sprintf("%s  %s", i18n.T("public_keys.footer"), filterStatus))

	return lipgloss.JoinVertical(lipgloss.Left, header, "\n", mainArea, "\n", helpLine)

}

// boolToYesNo is a helper function to convert a boolean to a localized "Yes" or "No".
func boolToYesNo(val bool) string {
	if val {
		return i18n.T("public_keys.yes")
	}
	return i18n.T("public_keys.no")
}

// viewUsageReport renders the view that shows which accounts a key is assigned to.
func (m publicKeysModel) viewUsageReport() string {
	var b strings.Builder
	title := fmt.Sprintf("📜 Key Usage Report for: %s", m.usageReportKey.Comment)
	b.WriteString(titleStyle.Render(title))

	if len(m.usageReportAccts) == 0 {
		b.WriteString(helpStyle.Render("This key is not assigned to any accounts."))
	} else {
		b.WriteString("This key is assigned to the following accounts:\n\n")
		for _, acc := range m.usageReportAccts {
			b.WriteString(itemStyle.Render("- "+acc.String()) + "\n")
		}
	}

	b.WriteString(helpStyle.Render("\n(esc or q to go back)"))
	return b.String()
}
