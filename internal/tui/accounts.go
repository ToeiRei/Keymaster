// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/deploy"
	"github.com/toeirei/keymaster/internal/i18n"
	"github.com/toeirei/keymaster/internal/model"
	"github.com/toeirei/keymaster/internal/sshkey"
	"golang.org/x/crypto/ssh"
)

// A message to signal that we should go back to the main menu.
type backToMenuMsg struct{}

// A message to signal that we should go back to the list from the form.
type backToListMsg struct{}

// A message to signal that a host key has been verified.
type hostKeyVerifiedMsg struct {
	hostname string
	warning  string
	err      error
}

// A message to signal that a remote key import has finished.
type remoteKeysImportedMsg struct {
	accountID    int
	importedKeys []model.PublicKey
	skippedCount int
	warning      string
	err          error
}

type accountsViewState int

const (
	accountsListView accountsViewState = iota
	accountsFormView
	accountsImportConfirmView
)

// accountsModel is the model for the account management view.
type accountsModel struct {
	state                  accountsViewState
	form                   accountFormModel
	accounts               []model.Account // The master list
	displayedAccounts      []model.Account // The filtered list for display
	cursor                 int
	status                 string // For showing status messages like "Deleted..."
	err                    error
	pendingImportAccountID int
	pendingImportKeys      []model.PublicKey
	filter                 string
	isFiltering            bool
	// For delete confirmation
	isConfirmingDelete bool
	accountToDelete    model.Account
	confirmCursor      int // 0 for No, 1 for Yes
	width, height      int
}

func newAccountsModel() accountsModel {
	m := accountsModel{}
	var err error
	m.accounts, err = db.GetAllAccounts()
	if err != nil {
		m.err = err
	}
	m.rebuildDisplayedAccounts()
	return m
}

func (m accountsModel) Init() tea.Cmd {
	return nil
}

func (m *accountsModel) rebuildDisplayedAccounts() {
	if m.filter == "" {
		m.displayedAccounts = m.accounts
	} else {
		m.displayedAccounts = []model.Account{}
		lowerFilter := strings.ToLower(m.filter)
		for _, acc := range m.accounts {
			// Check against username, hostname, label, and tags
			if strings.Contains(strings.ToLower(acc.Username), lowerFilter) ||
				strings.Contains(strings.ToLower(acc.Hostname), lowerFilter) ||
				strings.Contains(strings.ToLower(acc.Label), lowerFilter) ||
				strings.Contains(strings.ToLower(acc.Tags), lowerFilter) {
				m.displayedAccounts = append(m.displayedAccounts, acc)
			}
		}
	}

	// Reset cursor if it's out of bounds
	if m.cursor >= len(m.displayedAccounts) {
		if len(m.displayedAccounts) > 0 {
			m.cursor = len(m.displayedAccounts) - 1
		} else {
			m.cursor = 0
		}
	}
}

func (m accountsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Handle window size messages first, as they affect layout.
	if sizeMsg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = sizeMsg.Width
		m.height = sizeMsg.Height
	}

	// Delegate updates to the form if it's active.
	if m.state == accountsFormView {
		// If the form signals an account was created, switch back to the list and refresh.
		if am, ok := msg.(accountModifiedMsg); ok {
			m.state = accountsListView
			m.status = i18n.T("accounts.status.modified_success")
			m.accounts, m.err = db.GetAllAccounts()
			m.rebuildDisplayedAccounts()

			// Find the new/edited account in the list and set the cursor
			for i, acc := range m.displayedAccounts {
				if acc.ID == am.accountID {
					m.cursor = i
					break
				}
			}

			// If it was a new account, automatically try to trust the host.
			if am.isNew && am.hostname != "" {
				m.status += "\n" + i18n.T("accounts.status.trust_attempt", am.hostname)
				return m, verifyHostKeyCmd(am.hostname)
			}
			return m, nil // For edits, just return to the list.
		}
		// If the form signals to go back, just switch the view.
		if _, ok := msg.(backToListMsg); ok {
			m.state = accountsListView
			m.status = "" // Clear any status
			return m, nil
		}

		var newFormModel tea.Model
		newFormModel, cmd = m.form.Update(msg)
		m.form = newFormModel.(accountFormModel)
		return m, cmd
	}

	// Handle updates for the import confirmation view
	if m.state == accountsImportConfirmView {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "y":
				// Assign the keys
				for _, key := range m.pendingImportKeys {
					_ = db.AssignKeyToAccount(m.pendingImportAccountID, key.ID)
				}
				m.status = i18n.T("accounts.status.import_assigned", len(m.pendingImportKeys))
				m.state = accountsListView
				m.pendingImportAccountID = 0
				m.pendingImportKeys = nil
				return m, nil
			case "n", "q", "esc":
				// Don't assign, just go back
				m.status = i18n.T("accounts.status.import_skipped_assign", len(m.pendingImportKeys))
				m.state = accountsListView
				m.pendingImportAccountID = 0
				m.pendingImportKeys = nil
				return m, nil
			}
		}
		return m, nil
	}

	// Handle delete confirmation
	if m.isConfirmingDelete {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "y":
				// Fallthrough to confirm
			case "n", "q", "esc":
				m.isConfirmingDelete = false
				m.status = i18n.T("accounts.status.delete_cancelled")
				return m, nil
			case "right", "tab", "l":
				m.confirmCursor = 1 // Yes
				return m, nil
			case "left", "shift+tab", "h":
				m.confirmCursor = 0 // No
				return m, nil
			case "enter":
				if m.confirmCursor == 1 { // Yes is selected
					if err := db.DeleteAccount(m.accountToDelete.ID); err != nil {
						m.err = err
					} else {
						m.status = i18n.T("accounts.status.delete_success", m.accountToDelete.String())
						m.accounts, m.err = db.GetAllAccounts()
						m.rebuildDisplayedAccounts()
					}
				}
				m.isConfirmingDelete = false
				return m, nil
			}
		}
		return m, nil
	}

	// Handle async messages for the list view
	switch msg := msg.(type) {
	case hostKeyVerifiedMsg:
		if msg.err != nil {
			m.status = i18n.T("accounts.status.verify_fail", msg.hostname, msg.err)
		} else {
			m.status = i18n.T("accounts.status.verify_success", msg.hostname)
			if msg.warning != "" {
				m.status += fmt.Sprintf("\n%s", msg.warning)
			}
		}
		return m, nil
	case remoteKeysImportedMsg:
		if msg.err != nil {
			m.status = i18n.T("accounts.status.import_fail", msg.err)
			if msg.warning != "" {
				m.status += "\n" + msg.warning
			}
			return m, nil
		}

		// Handle success case
		if len(msg.importedKeys) == 0 {
			m.status = i18n.T("accounts.status.import_no_new", msg.skippedCount)
		} else {
			// We have keys, move to confirmation state
			m.state = accountsImportConfirmView
			m.pendingImportAccountID = msg.accountID
			m.pendingImportKeys = msg.importedKeys
			m.status = i18n.T("accounts.import_confirm.question", len(m.pendingImportKeys))
		}

		if msg.warning != "" {
			m.status += "\n" + msg.warning
		}
		return m, nil
	}

	// --- This is the list view update logic ---
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// If we are in filtering mode, capture all input for the filter.
		if m.isFiltering {
			switch msg.Type {
			case tea.KeyEsc:
				m.isFiltering = false // Exit filter mode, but keep filter
			case tea.KeyEnter:
				m.isFiltering = false
			case tea.KeyBackspace:
				if len(m.filter) > 0 {
					m.filter = m.filter[:len(m.filter)-1]
					m.rebuildDisplayedAccounts()
				}
			case tea.KeyRunes:
				m.filter += string(msg.Runes)
				m.rebuildDisplayedAccounts()
			}
			return m, nil
		}

		// Not in filtering mode, handle commands.
		switch msg.String() {
		case "/":
			m.isFiltering = true
			// Keep existing filter if present, don't clear
			m.rebuildDisplayedAccounts()
			return m, nil

			// Go back to the main menu.
		case "q":
			if m.filter != "" && !m.isFiltering {
				m.filter = ""
				m.rebuildDisplayedAccounts()
				return m, nil
			}
			return m, func() tea.Msg { return backToMenuMsg{} }
		case "esc":
			if m.isFiltering {
				m.isFiltering = false // Just exit filter mode, keep filter
				return m, nil
			} else if m.filter != "" {
				m.filter = ""
				m.rebuildDisplayedAccounts()
				return m, nil
			}
			return m, func() tea.Msg { return backToMenuMsg{} }

		// Navigate up.
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		// Navigate down.
		case "down", "j":
			if m.cursor < len(m.displayedAccounts)-1 {
				m.cursor++
			}

		// Delete an account.
		case "d", "delete":
			if len(m.displayedAccounts) > 0 {
				m.accountToDelete = m.displayedAccounts[m.cursor]
				m.isConfirmingDelete = true
				m.confirmCursor = 0 // Default to No
			}
			return m, nil

		// Edit an account's label.
		case "e":
			if len(m.displayedAccounts) > 0 {
				accToEdit := m.displayedAccounts[m.cursor]
				m.state = accountsFormView
				m.form = newAccountFormModel(&accToEdit)
				m.status = ""
				return m, m.form.Init()
			}
			return m, nil

		// Toggle active status.
		case "t":
			if len(m.displayedAccounts) > 0 {
				accToToggle := m.displayedAccounts[m.cursor]
				if err := db.ToggleAccountStatus(accToToggle.ID); err != nil {
					m.err = err
				} else {
					// Refresh the list after toggling.
					m.status = i18n.T("accounts.status.toggle_success", accToToggle.String())
					m.accounts, m.err = db.GetAllAccounts()
					m.rebuildDisplayedAccounts()
				}
			}
			return m, nil

		// Verify/Trust host key.
		case "v":
			if len(m.displayedAccounts) > 0 {
				accToTrust := m.displayedAccounts[m.cursor]
				m.status = i18n.T("accounts.status.verify_start", accToTrust.Hostname)
				return m, verifyHostKeyCmd(accToTrust.Hostname)
			}
			return m, nil

		// Switch to the form view to add a new account.
		case "a":
			m.state = accountsFormView
			m.form = newAccountFormModel(nil)
			m.status = "" // Clear status before showing form
			return m, m.form.Init()

		// Import keys from remote host.
		case "i":
			if len(m.displayedAccounts) > 0 {
				accToImportFrom := m.displayedAccounts[m.cursor]
				m.status = i18n.T("accounts.status.import_start", accToImportFrom.String())
				return m, importRemoteKeysCmd(accToImportFrom)
			}
			return m, nil
		}
	}
	return m, nil
}

func (m accountsModel) View() string {
	// If we're in the form view, render that instead.
	if m.state == accountsFormView {
		return m.form.View()
	}

	// If we're in the import confirmation view, just show the status.
	if m.state == accountsImportConfirmView {
		var viewItems []string
		viewItems = append(viewItems, titleStyle.Render(i18n.T("accounts.import_confirm.title")))
		viewItems = append(viewItems, i18n.T("accounts.import_confirm.found_keys", len(m.pendingImportKeys)), "")

		for _, key := range m.pendingImportKeys {
			line := fmt.Sprintf("- %s (%s)", key.Comment, key.Algorithm)
			viewItems = append(viewItems, itemStyle.Render(line))
		}

		viewItems = append(viewItems, "", helpStyle.Render(m.status))
		return lipgloss.JoinVertical(lipgloss.Left, viewItems...)
	}

	// If we are confirming a delete, render the modal instead of the list.
	if m.isConfirmingDelete {
		var b strings.Builder
		b.WriteString(titleStyle.Render(i18n.T("accounts.delete_confirm.title")))

		question := i18n.T("accounts.delete_confirm.question", m.accountToDelete.String())
		b.WriteString(question)
		b.WriteString("\n\n")

		var yesButton, noButton string
		if m.confirmCursor == 1 { // Yes
			yesButton = activeButtonStyle.Render(i18n.T("accounts.delete_confirm.yes"))
			noButton = buttonStyle.Render(i18n.T("accounts.delete_confirm.no"))
		} else { // No
			yesButton = buttonStyle.Render(i18n.T("accounts.delete_confirm.yes"))
			noButton = activeButtonStyle.Render(i18n.T("accounts.delete_confirm.no"))
		}

		buttons := lipgloss.JoinHorizontal(lipgloss.Top, noButton, "  ", yesButton)
		b.WriteString(buttons)

		b.WriteString("\n" + helpStyle.Render("\n"+i18n.T("accounts.delete_confirm.help")))

		// Center the whole dialog
		return lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			dialogBoxStyle.Render(b.String()),
		)
	}

	// --- Styled, dashboard-like layout ---
	title := mainTitleStyle.Render("🔑 " + i18n.T("accounts.title"))
	header := lipgloss.NewStyle().Align(lipgloss.Center).Render(title)

	// List pane (left)
	var listItems []string
	for i, acc := range m.displayedAccounts {
		line := acc.String()
		if m.cursor == i {
			line = "▸ " + line
			if acc.IsActive {
				listItems = append(listItems, selectedItemStyle.Render(line))
			} else {
				listItems = append(listItems, selectedItemStyle.Copy().Strikethrough(true).Render(line))
			}
		} else {
			if acc.IsActive {
				listItems = append(listItems, itemStyle.Render("  "+line))
			} else {
				listItems = append(listItems, inactiveItemStyle.Render("  "+line))
			}
		}
	}

	if len(m.displayedAccounts) == 0 && m.filter == "" {
		listItems = append(listItems, helpStyle.Render(i18n.T("accounts.empty")))
	} else if len(m.displayedAccounts) == 0 && m.filter != "" {
		listItems = append(listItems, helpStyle.Render(i18n.T("accounts.empty_filtered")))
	}

	listPaneTitle := lipgloss.NewStyle().Bold(true).Render(i18n.T("accounts.list_title"))
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
		detailsItems = append(detailsItems, helpStyle.Render(fmt.Sprintf(i18n.T("accounts.error"), m.err)))
	} else if m.status != "" {
		detailsItems = append(detailsItems, statusMessageStyle.Render(m.status))
	}

	// Show tags for the selected account in the detail pane
	if len(m.displayedAccounts) > 0 && m.cursor < len(m.displayedAccounts) {
		acc := m.displayedAccounts[m.cursor]
		if acc.Tags != "" {
			detailsItems = append(detailsItems, "", helpStyle.Render(fmt.Sprintf(i18n.T("accounts.tags"), acc.Tags)))
		}
	}

	// Only show filter status if filtering
	if m.isFiltering {
		detailsItems = append(detailsItems, "", helpStyle.Render(fmt.Sprintf(i18n.T("accounts.filtering"), m.filter)))
	}

	rightPane := paneStyle.Width(detailWidth).MarginLeft(2).Render(lipgloss.JoinVertical(lipgloss.Left, detailsItems...))

	mainArea := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	// Help/footer line always at the bottom
	footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Background(lipgloss.Color("236")).Padding(0, 1).Italic(true)
	// Compose help and filter status on one line
	var filterStatus string
	if m.isFiltering {
		filterStatus = fmt.Sprintf(i18n.T("accounts.filtering"), m.filter)
	} else if m.filter != "" {
		filterStatus = fmt.Sprintf(i18n.T("accounts.filter_active"), m.filter)
	} else {
		filterStatus = i18n.T("accounts.filter_hint")
	}
	helpLine := footerStyle.Render(fmt.Sprintf("%s  %s", i18n.T("accounts.footer"), filterStatus))

	return lipgloss.JoinVertical(lipgloss.Left, header, "\n", mainArea, "\n", helpLine)
}

// verifyHostKeyCmd is a tea.Cmd that fetches a host's public key and saves it.
func verifyHostKeyCmd(hostname string) tea.Cmd {
	return func() tea.Msg {
		key, err := deploy.GetRemoteHostKey(hostname)
		if err != nil {
			return hostKeyVerifiedMsg{hostname: hostname, err: err, warning: ""}
		}

		// Check for weak algorithms.
		warning := sshkey.CheckHostKeyAlgorithm(key)

		// Convert to string format for storage.
		keyStr := string(ssh.MarshalAuthorizedKey(key))

		// Store in DB.
		err = db.AddKnownHostKey(hostname, keyStr)
		if err != nil {
			return hostKeyVerifiedMsg{hostname: hostname, err: fmt.Errorf("failed to save key to database: %w", err), warning: warning}
		}

		return hostKeyVerifiedMsg{hostname: hostname, err: nil, warning: warning}
	}
}

// importRemoteKeysCmd is a tea.Cmd that fetches keys from a remote host and imports them.
func importRemoteKeysCmd(account model.Account) tea.Cmd {
	return func() tea.Msg {
		imported, skipped, warning, err := deploy.ImportRemoteKeys(account)
		return remoteKeysImportedMsg{accountID: account.ID, importedKeys: imported, skippedCount: skipped, warning: warning, err: err}
	}
}
