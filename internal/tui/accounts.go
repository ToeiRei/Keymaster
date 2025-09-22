package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/deploy"
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
			m.status = "Successfully modified account."
			m.accounts, m.err = db.GetAllAccounts()
			m.rebuildDisplayedAccounts()

			// Find the new/edited account in the list and set the cursor
			for i, acc := range m.displayedAccounts {
				if acc.Username == am.username && acc.Hostname == am.hostname {
					m.cursor = i
					break
				}
			}

			// If it was a new account, automatically try to trust the host.
			if am.isNew && am.hostname != "" {
				m.status += fmt.Sprintf("\nAttempting to trust host %s...", am.hostname)
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
				m.status = fmt.Sprintf("Assigned %d imported keys.", len(m.pendingImportKeys))
				m.state = accountsListView
				m.pendingImportAccountID = 0
				m.pendingImportKeys = nil
				return m, nil
			case "n", "q", "esc":
				// Don't assign, just go back
				m.status = fmt.Sprintf("Imported %d new keys without assigning.", len(m.pendingImportKeys))
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
					if err := db.DeleteAccount(m.accountToDelete.ID); err != nil {
						m.err = err
					} else {
						m.status = fmt.Sprintf("Deleted account: %s", m.accountToDelete.String())
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
			m.status = fmt.Sprintf("Failed to verify host %s: %v", msg.hostname, msg.err)
		} else {
			m.status = fmt.Sprintf("Successfully trusted host key for %s.", msg.hostname)
			if msg.warning != "" {
				m.status += fmt.Sprintf("\n%s", msg.warning)
			}
		}
		return m, nil
	case remoteKeysImportedMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Import failed: %v", msg.err)
			if msg.warning != "" {
				m.status += "\n" + msg.warning
			}
			return m, nil
		}

		// Handle success case
		if len(msg.importedKeys) == 0 {
			m.status = fmt.Sprintf("No new keys found to import. Skipped %d duplicates.", msg.skippedCount)
		} else {
			// We have keys, move to confirmation state
			m.state = accountsImportConfirmView
			m.pendingImportAccountID = msg.accountID
			m.pendingImportKeys = msg.importedKeys
			m.status = fmt.Sprintf("Assign these %d new keys to this account? (y/n)", len(m.pendingImportKeys))
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
				m.isFiltering = false
				m.filter = ""
				m.rebuildDisplayedAccounts()
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
			m.filter = "" // Start with a fresh filter
			m.rebuildDisplayedAccounts()
			return m, nil

		// Go back to the main menu.
		case "q", "esc":
			if m.filter != "" && !m.isFiltering {
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
					m.status = fmt.Sprintf("Toggled status for: %s", accToToggle.String())
					m.accounts, m.err = db.GetAllAccounts()
					m.rebuildDisplayedAccounts()
				}
			}
			return m, nil

		// Verify/Trust host key.
		case "v":
			if len(m.displayedAccounts) > 0 {
				accToTrust := m.displayedAccounts[m.cursor]
				m.status = fmt.Sprintf("Verifying host key for %s...", accToTrust.Hostname)
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
				m.status = fmt.Sprintf("Importing keys from %s...", accToImportFrom.String())
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
		viewItems = append(viewItems, titleStyle.Render("✨ Remote Import Confirmation"))
		viewItems = append(viewItems, fmt.Sprintf("Found %d new public keys on the remote host:", len(m.pendingImportKeys)), "")

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
		b.WriteString(titleStyle.Render("🗑️ Confirm Deletion"))

		question := fmt.Sprintf("Are you sure you want to delete the account\n\n%s?", m.accountToDelete.String())
		b.WriteString(question)
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

	// --- This is the list view rendering ---
	var viewItems []string
	viewItems = append(viewItems, titleStyle.Render("🔑 Manage Accounts"))

	if m.err != nil {
		viewItems = append(viewItems, fmt.Sprintf("Error: %v", m.err))
		return lipgloss.JoinVertical(lipgloss.Left, viewItems...)
	}

	var listItems []string
	for i, acc := range m.displayedAccounts {
		line := acc.String()
		if acc.Tags != "" {
			// Show tags in a muted color
			line += " " + helpStyle.Render(fmt.Sprintf("[%s]", acc.Tags))
		}

		if m.cursor == i {
			line = "▸ " + line
			if acc.IsActive {
				listItems = append(listItems, selectedItemStyle.Render(line))
			} else {
				// Keep strikethrough for inactive selected items
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
	viewItems = append(viewItems, lipgloss.JoinVertical(lipgloss.Left, listItems...))

	if len(m.displayedAccounts) == 0 && m.filter == "" {
		viewItems = append(viewItems, helpStyle.Render("No accounts found. Press 'a' to add one."))
	} else if len(m.displayedAccounts) == 0 && m.filter != "" {
		viewItems = append(viewItems, helpStyle.Render("No accounts match your filter."))
	}
	viewItems = append(viewItems, "") // Spacer

	var filterStatus string
	if m.isFiltering {
		filterStatus = fmt.Sprintf("Filter: %s█", m.filter)
	} else if m.filter != "" {
		filterStatus = fmt.Sprintf("Filter: %s (press 'esc' to clear)", m.filter)
	} else {
		filterStatus = "Press / to filter..."
	}

	viewItems = append(viewItems, helpStyle.Render(fmt.Sprintf("(a)dd (e)dit (d)elete (t)oggle (v)erify (i)mport (q)uit\n%s", filterStatus)))
	if m.status != "" {
		viewItems = append(viewItems, "", statusMessageStyle.Render(m.status))
	}

	return lipgloss.JoinVertical(lipgloss.Left, viewItems...)
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
