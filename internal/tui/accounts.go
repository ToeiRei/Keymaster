// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
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
	viewport               viewport.Model
	displayedAccounts      []model.Account // The filtered list for display
	cursor                 int
	status                 string // For showing status messages like "Deleted..."
	err                    error
	pendingImportAccountID int
	pendingImportKeys      []model.PublicKey
	filter                 string
	isFiltering            bool
	// For delete confirmation
	isConfirmingDelete       bool
	accountToDelete          model.Account
	confirmCursor            int  // 0 for No, 1 for Yes, 2 for checkbox
	withDecommission         bool // Checkbox state - whether to decommission (default true)
	isConfirmingKeySelection bool // True when showing key selection dialog
	availableKeys            []model.PublicKey
	selectedKeysToKeep       map[int]bool // Keys selected to keep
	keySelectionCursor       int          // Cursor position in key list
	keySelectionButtonCursor int          // 0 for Cancel, 1 for Continue
	keySelectionInButtonMode bool         // True when navigating buttons instead of keys
	width, height            int
	searcher                 db.AccountSearcher
}

func newAccountsModel() accountsModel {
	return newAccountsModelWithSearcher(db.DefaultAccountSearcher())
}

// newAccountsModelWithSearcher creates an accountsModel that will use the
// provided AccountSearcher for server-side searches. Pass nil to rely on the
// package default searcher.
func newAccountsModelWithSearcher(s db.AccountSearcher) accountsModel {
	m := accountsModel{
		viewport: viewport.New(0, 0),
		searcher: s,
	}
	var err error
	m.accounts, err = db.GetAllAccounts()
	if err != nil {
		m.err = err
	}
	m.rebuildDisplayedAccounts()
	m.viewport.SetContent(m.listContentView())
	return m
}

func (m accountsModel) Init() tea.Cmd {
	return nil
}

func (m *accountsModel) rebuildDisplayedAccounts() {
	if m.filter == "" {
		m.displayedAccounts = m.accounts
	} else {
		// Prefer server-side search via AccountSearcher when available. This
		// decouples UI filtering from DB implementation and allows DB engines
		// to optimize searches. However, some tests expect local in-memory
		// filtering semantics; prefer server-side results only when they are
		// successful and return non-empty results, otherwise fall back to
		// local in-memory filtering to preserve test expectations.
		// Build local results first so we can compare.
		localResults := []model.Account{}
		lowerFilter := strings.ToLower(m.filter)
		for _, acc := range m.accounts {
			// Build a single lowercased representation per account to avoid
			// repeated calls to strings.ToLower in the hot loop.
			lowerAcc := strings.ToLower(acc.Username + " " + acc.Hostname + " " + acc.Label + " " + acc.Tags)
			if strings.Contains(lowerAcc, lowerFilter) {
				localResults = append(localResults, acc)
			}
		}

		// Prefer the injected searcher if present, otherwise use the package default.
		var searcher db.AccountSearcher
		if m.searcher != nil {
			searcher = m.searcher
		} else {
			searcher = db.DefaultAccountSearcher()
		}

		if searcher != nil {
			if res, err := searcher.SearchAccounts(m.filter); err == nil && len(res) > 0 {
				m.displayedAccounts = res
			} else {
				// On error or empty server result, fall back to local filtering.
				m.displayedAccounts = localResults
			}
		} else {
			m.displayedAccounts = localResults
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

func (m *accountsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Handle window size messages first, as they affect layout.
	if sizeMsg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = sizeMsg.Width
		m.height = sizeMsg.Height

		headerHeight := lipgloss.Height(lipgloss.NewStyle().Align(lipgloss.Center).Render(m.headerView()))
		footerHeight := lipgloss.Height(m.footerView())
		// The space for the panes is the total height minus header, footer, and the newlines around the main area.
		mainAreaHeight := m.height - headerHeight - footerHeight - 2 // -2 for newlines around mainArea

		// The viewport's height is the available area minus chrome for the pane itself (borders, padding, title).
		m.viewport.Height = mainAreaHeight - 6
		m.viewport.Width = m.width/2 - 4 // Approximate width for left pane
	}

	// Delegate updates to the form if it's active.
	if m.state == accountsFormView {
		// If the form signals an account was created, switch back to the list and refresh.
		if am, ok := msg.(accountModifiedMsg); ok {
			m.state = accountsListView
			m.status = i18n.T("accounts.status.modified_success")
			m.accounts, m.err = db.GetAllAccounts()
			m.rebuildDisplayedAccounts()
			m.viewport.SetContent(m.listContentView()) // Update viewport content

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
				km := db.DefaultKeyManager()
				if km == nil {
					// No key manager available; skip assignment but record status
					m.status = i18n.T("accounts.status.import_skipped_assign", len(m.pendingImportKeys))
				} else {
					for _, key := range m.pendingImportKeys {
						_ = km.AssignKeyToAccount(key.ID, m.pendingImportAccountID)
					}
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

	// Handle key selection dialog
	if m.isConfirmingKeySelection {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "q", "esc":
				m.isConfirmingKeySelection = false
				m.isConfirmingDelete = false // Cancel everything, go to account overview
				m.keySelectionInButtonMode = false
				m.status = i18n.T("accounts.status.delete_cancelled")
				return m, nil
			case "up", "k":
				if m.keySelectionInButtonMode {
					// Switch from buttons to keys list
					m.keySelectionInButtonMode = false
					m.keySelectionCursor = len(m.availableKeys) - 1
				} else if m.keySelectionCursor > 0 {
					m.keySelectionCursor--
				}
				return m, nil
			case "down", "j":
				if !m.keySelectionInButtonMode && m.keySelectionCursor < len(m.availableKeys)-1 {
					m.keySelectionCursor++
				} else if !m.keySelectionInButtonMode && m.keySelectionCursor == len(m.availableKeys)-1 {
					// At last key, move to buttons
					m.keySelectionInButtonMode = true
					m.keySelectionButtonCursor = 0
				}
				return m, nil
			case "tab":
				if !m.keySelectionInButtonMode {
					// Tab from keys to buttons
					m.keySelectionInButtonMode = true
					m.keySelectionButtonCursor = 0
				} else {
					// Tab between buttons
					m.keySelectionButtonCursor = (m.keySelectionButtonCursor + 1) % 2
				}
				return m, nil
			case "shift+tab":
				if m.keySelectionInButtonMode && m.keySelectionButtonCursor > 0 {
					m.keySelectionButtonCursor--
				} else if m.keySelectionInButtonMode && m.keySelectionButtonCursor == 0 {
					// Go back to keys list
					m.keySelectionInButtonMode = false
					if len(m.availableKeys) > 0 {
						m.keySelectionCursor = len(m.availableKeys) - 1
					}
				}
				return m, nil
			case "left":
				if m.keySelectionInButtonMode {
					m.keySelectionButtonCursor = 0 // Cancel
				}
				return m, nil
			case "right":
				if m.keySelectionInButtonMode {
					m.keySelectionButtonCursor = 1 // Continue
				}
				return m, nil
			case " ": // Toggle selection (only for keys, not buttons)
				if !m.keySelectionInButtonMode && m.keySelectionCursor < len(m.availableKeys) {
					keyID := m.availableKeys[m.keySelectionCursor].ID
					m.selectedKeysToKeep[keyID] = !m.selectedKeysToKeep[keyID]
				}
				return m, nil
			case "enter":
				if m.keySelectionInButtonMode {
					// Button action
					if m.keySelectionButtonCursor == 0 { // Cancel
						m.isConfirmingKeySelection = false
						m.isConfirmingDelete = false // Cancel everything, go to account overview
						m.keySelectionInButtonMode = false
						m.status = i18n.T("accounts.status.delete_cancelled")
						return m, nil
					} else { // Continue
						m.isConfirmingKeySelection = false
						m.keySelectionInButtonMode = false
						return m, m.performDecommissionWithKeys()
					}
				} else {
					// Toggle key selection
					if m.keySelectionCursor < len(m.availableKeys) {
						keyID := m.availableKeys[m.keySelectionCursor].ID
						m.selectedKeysToKeep[keyID] = !m.selectedKeysToKeep[keyID]
					}
				}
				return m, nil
			case "a": // Select all (keep all)
				for _, key := range m.availableKeys {
					m.selectedKeysToKeep[key.ID] = true
				}
				return m, nil
			case "n": // Select none (keep none)
				for _, key := range m.availableKeys {
					m.selectedKeysToKeep[key.ID] = false
				}
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
			case "n", "q", "esc":
				m.isConfirmingDelete = false
				m.status = i18n.T("accounts.status.delete_cancelled")
				return m, nil
			case "right", "tab", "down":
				m.confirmCursor = (m.confirmCursor + 1) % 3 // Navigate through No/Yes/Checkbox
				return m, nil
			case "left", "shift+tab", "up":
				m.confirmCursor = (m.confirmCursor + 2) % 3 // Navigate backwards through No/Yes/Checkbox
				return m, nil
			case " ": // Toggle decommission checkbox when focused
				if m.confirmCursor == 2 {
					m.withDecommission = !m.withDecommission
				}
				return m, nil
			case "enter":
				switch m.confirmCursor {
				case 1: // Yes is selected
					if m.withDecommission {
						// Show key selection dialog - don't close delete dialog yet
						return m, m.loadKeysForSelection()
					} else {
						// Simple delete without decommission. Prefer AccountManager when available.
						mgr := db.DefaultAccountManager()
						if mgr == nil {
							m.err = fmt.Errorf("no account manager configured")
						} else if err := mgr.DeleteAccount(m.accountToDelete.ID); err != nil {
							m.err = err
						} else {
							m.status = i18n.T("accounts.status.delete_success", m.accountToDelete.String())
							m.accounts, m.err = db.GetAllAccounts()
							m.rebuildDisplayedAccounts()
							m.viewport.SetContent(m.listContentView())
						}
						m.isConfirmingDelete = false
						return m, nil
					}
				case 2: // Checkbox is selected
					m.withDecommission = !m.withDecommission
					return m, nil
				case 0: // No is selected
					m.isConfirmingDelete = false
					m.status = i18n.T("accounts.status.delete_cancelled")
					return m, nil
				}
				return m, nil
			}
		}
		// Don't return here - allow other message types (like keySelectionLoadedMsg) to pass through
	}

	// Handle async messages for the list view
	switch msg := msg.(type) {
	case error:
		// Handle errors from async operations
		m.err = msg
		m.isConfirmingDelete = false
		m.isConfirmingKeySelection = false
		return m, nil
	case keySelectionLoadedMsg:
		// Keys loaded, initialize selection state
		m.availableKeys = msg.keys
		m.selectedKeysToKeep = make(map[int]bool)
		// Default: keep all keys (user can deselect what they want to remove)
		for _, key := range m.availableKeys {
			m.selectedKeysToKeep[key.ID] = true
		}
		m.keySelectionCursor = 0
		m.isConfirmingKeySelection = true
		m.isConfirmingDelete = false
		return m, nil
	case decommissionCompletedMsg:
		// Decommission completed, show result
		result := msg.result
		if result.DatabaseDeleteError != nil {
			m.err = result.DatabaseDeleteError
			m.status = i18n.T("accounts.status.decommission_failed", result.DatabaseDeleteError)
		} else if result.RemoteCleanupError != nil {
			m.status = i18n.T("accounts.status.decommission_partial", result.AccountString, result.RemoteCleanupError)
			m.accounts, m.err = db.GetAllAccounts()
			m.rebuildDisplayedAccounts()
			m.viewport.SetContent(m.listContentView())
		} else {
			m.status = i18n.T("accounts.status.decommission_success", result.AccountString)
			m.accounts, m.err = db.GetAllAccounts()
			m.rebuildDisplayedAccounts()
			m.viewport.SetContent(m.listContentView())
		}
		return m, nil
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
					m.viewport.SetContent(m.listContentView())
				}
			case tea.KeyRunes:
				m.filter += string(msg.Runes)
				m.rebuildDisplayedAccounts()
				m.viewport.SetContent(m.listContentView())
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
				m.viewport.SetContent(m.listContentView())
				return m, nil
			}
			return m, func() tea.Msg { return backToMenuMsg{} }

		// Navigate up.
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				// SetContent must be called to redraw the cursor.
				// This resets the viewport's YOffset, so ensureCursorInView must be called *after*.
				m.viewport.SetContent(m.listContentView())
				m.ensureCursorInView()
			}

		// Navigate down.
		case "down", "j":
			if m.cursor < len(m.displayedAccounts)-1 {
				m.cursor++
				m.viewport.SetContent(m.listContentView())
				m.ensureCursorInView()
			}

		// Delete an account.
		case "d", "delete":
			if len(m.displayedAccounts) > 0 {
				m.accountToDelete = m.displayedAccounts[m.cursor]
				m.isConfirmingDelete = true
				m.withDecommission = true // Default to decommission
				m.confirmCursor = 0       // Default to No
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
					m.viewport.SetContent(m.listContentView())
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

	// Pass messages to the viewport at the end
	var vpCmd tea.Cmd
	m.viewport, vpCmd = m.viewport.Update(msg)
	return m, tea.Batch(cmd, vpCmd)
}

// ensureCursorInView adjusts the viewport's Y offset to ensure the cursor is visible.
// It implements "edge scrolling," where the list only scrolls when the cursor
// hits the top or bottom of the visible area.
func (m *accountsModel) ensureCursorInView() {
	top := m.viewport.YOffset
	bottom := top + m.viewport.Height - 1

	if m.cursor < top {
		// Cursor is above the viewport, so scroll up to bring it into view.
		m.viewport.YOffset = m.cursor
	} else if m.cursor > bottom {
		// Cursor is below the viewport, so scroll down.
		m.viewport.YOffset = m.cursor - m.viewport.Height + 1
	}
}

// headerView renders the main title of the page.
func (m *accountsModel) headerView() string {
	// --- Styled, dashboard-like layout ---
	return mainTitleStyle.Render("🔑 " + i18n.T("accounts.title"))
}

// listContentView builds the string content for the list viewport.
func (m *accountsModel) listContentView() string {
	var b strings.Builder
	for i, acc := range m.displayedAccounts {
		var styledLine string
		line := "  " + acc.String()
		if m.cursor == i {
			line = "▸ " + acc.String() // This line was missing
			if !acc.IsActive {
				styledLine = selectedItemStyle.Strikethrough(true).Render(line)
			} else {
				styledLine = selectedItemStyle.Render(line)
			}
		} else if !acc.IsActive {
			styledLine = inactiveItemStyle.Render(line)
		} else {
			styledLine = itemStyle.Render(line)
		}
		b.WriteString(styledLine + "\n")
	}
	return b.String()
}

// detailContentView builds the string content for the detail pane.
func (m *accountsModel) detailContentView() string {
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
			detailsItems = append(detailsItems, "", helpStyle.Render(i18n.T("accounts.tags", acc.Tags)))
		}
	}
	// Only show filter status if filtering
	if m.isFiltering {
		detailsItems = append(detailsItems, "", helpStyle.Render(i18n.T("accounts.filtering", m.filter)))
	}
	return lipgloss.JoinVertical(lipgloss.Left, detailsItems...)
}

// footerView renders the help text at the bottom of the page.
func (m *accountsModel) footerView() string {
	footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Background(lipgloss.Color("236")).Padding(0, 1).Italic(true)
	var filterStatus string
	if m.isFiltering {
		filterStatus = i18n.T("accounts.filtering", m.filter)
	} else if m.filter != "" {
		filterStatus = i18n.T("accounts.filter_active", m.filter)
	} else {
		filterStatus = i18n.T("accounts.filter_hint")
	}
	return footerStyle.Render(fmt.Sprintf("%s  %s", i18n.T("accounts.footer"), filterStatus))
}

func (m *accountsModel) viewConfirmation() string {
	if m.isConfirmingKeySelection {
		return m.viewKeySelection()
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render(i18n.T("accounts.delete_confirm.title")))
	b.WriteString("\n\n")
	question := i18n.T("accounts.delete_confirm.question", m.accountToDelete.String())
	b.WriteString(question)
	b.WriteString("\n\n")

	// Decommission checkbox
	var checkbox string
	if m.withDecommission {
		checkbox = "☑ " + i18n.T("accounts.delete_confirm.decommission")
	} else {
		checkbox = "☐ " + i18n.T("accounts.delete_confirm.decommission")
	}

	if m.confirmCursor == 2 { // Checkbox focused
		checkbox = formSelectedItemStyle.Render(checkbox)
	} else {
		checkbox = formItemStyle.Render(checkbox)
	}
	b.WriteString(checkbox + "\n\n")

	// Yes/No buttons
	var yesButton, noButton string
	if m.confirmCursor == 1 { // Yes focused
		yesButton = activeButtonStyle.Render(i18n.T("accounts.delete_confirm.yes"))
		noButton = buttonStyle.Render(i18n.T("accounts.delete_confirm.no"))
	} else if m.confirmCursor == 0 { // No focused
		yesButton = buttonStyle.Render(i18n.T("accounts.delete_confirm.yes"))
		noButton = activeButtonStyle.Render(i18n.T("accounts.delete_confirm.no"))
	} else { // Checkbox focused (2)
		yesButton = buttonStyle.Render(i18n.T("accounts.delete_confirm.yes"))
		noButton = buttonStyle.Render(i18n.T("accounts.delete_confirm.no"))
	}
	buttons := lipgloss.JoinHorizontal(lipgloss.Top, noButton, "  ", yesButton)
	b.WriteString(buttons)
	b.WriteString("\n" + helpStyle.Render("\n"+i18n.T("accounts.delete_confirm.help_checkbox")))

	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		dialogBoxStyle.Render(b.String()),
	)
}

func (m *accountsModel) viewKeySelection() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(i18n.T("accounts.key_selection.title")))
	b.WriteString("\n\n")
	b.WriteString(i18n.T("accounts.key_selection.question", m.accountToDelete.String()))
	b.WriteString("\n\n")

	// Show keys list
	if len(m.availableKeys) == 0 {
		b.WriteString(helpStyle.Render(i18n.T("accounts.key_selection.no_keys")))
	} else {
		for i, key := range m.availableKeys {
			var line string
			// Show cursor only when in key selection mode
			if !m.keySelectionInButtonMode && i == m.keySelectionCursor {
				line = "▸ "
			} else {
				line = "  "
			}

			// Checkbox
			if m.selectedKeysToKeep[key.ID] {
				line += "☑ "
			} else {
				line += "☐ "
			}

			// Key info
			line += fmt.Sprintf("%s (...%s)", key.Comment, key.KeyData[len(key.KeyData)-12:])

			if !m.keySelectionInButtonMode && i == m.keySelectionCursor {
				b.WriteString(selectedItemStyle.Render(line))
			} else {
				b.WriteString(itemStyle.Render(line))
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")

	// Buttons
	var cancelButton, continueButton string
	if m.keySelectionInButtonMode {
		if m.keySelectionButtonCursor == 0 {
			cancelButton = activeButtonStyle.Render(i18n.T("accounts.key_selection.cancel"))
			continueButton = buttonStyle.Render(i18n.T("accounts.key_selection.continue"))
		} else {
			cancelButton = buttonStyle.Render(i18n.T("accounts.key_selection.cancel"))
			continueButton = activeButtonStyle.Render(i18n.T("accounts.key_selection.continue"))
		}
	} else {
		cancelButton = buttonStyle.Render(i18n.T("accounts.key_selection.cancel"))
		continueButton = buttonStyle.Render(i18n.T("accounts.key_selection.continue"))
	}
	buttons := lipgloss.JoinHorizontal(lipgloss.Top, cancelButton, "  ", continueButton)
	b.WriteString(buttons)

	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render(i18n.T("accounts.key_selection.help")))

	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		dialogBoxStyle.Render(b.String()),
	)
}

func (m *accountsModel) View() string {
	// Handle full-screen views first.
	switch m.state {
	case accountsFormView:
		return m.form.View()
	case accountsImportConfirmView:
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

	if m.isConfirmingDelete || m.isConfirmingKeySelection {
		return m.viewConfirmation()
	}

	// If we've reached here, we are in the main list view.
	header := m.headerView()

	// --- List Pane (Left) ---
	listPaneTitle := lipgloss.NewStyle().Bold(true).Render(i18n.T("accounts.list_title"))
	var listContent string
	if len(m.displayedAccounts) == 0 {
		if m.filter == "" {
			listContent = helpStyle.Render(i18n.T("accounts.empty"))
		} else {
			listContent = helpStyle.Render(i18n.T("accounts.empty_filtered"))
		}
	} else {
		listContent = m.viewport.View()
	}
	listPaneBody := lipgloss.JoinVertical(lipgloss.Left, listPaneTitle, "", listContent)

	// --- Detail Pane (Right) ---
	// We set the content here, but the height is driven by the left pane's viewport.
	detailContent := m.detailContentView()

	// --- Layout ---
	paneStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorSubtle).
		Padding(1, 2)

	// Use the viewport's calculated height to drive the pane height.
	// The pane height is the viewport height plus the vertical padding, borders, and title space.
	paneHeight := m.viewport.Height + 6
	menuWidth := m.width/2 - 4
	detailWidth := m.width - menuWidth - 8 // Correctly calculate the remaining width

	leftPane := paneStyle.Width(menuWidth).Height(paneHeight).Render(listPaneBody)
	rightPane := paneStyle.Width(detailWidth).Height(paneHeight).Render(detailContent)
	mainArea := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	return lipgloss.JoinVertical(lipgloss.Top, header, mainArea, m.footerView())
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

// loadKeysForSelection loads available keys for the account and shows selection dialog
func (m *accountsModel) loadKeysForSelection() tea.Cmd {
	return func() tea.Msg {
		// Get global keys
		km := db.DefaultKeyManager()
		if km == nil {
			return fmt.Errorf("no key manager available")
		}
		globalKeys, err := km.GetGlobalPublicKeys()
		if err != nil {
			return fmt.Errorf("failed to get global keys: %w", err)
		}

		// Get account-specific keys
		accountKeys, err := km.GetKeysForAccount(m.accountToDelete.ID)
		if err != nil {
			return fmt.Errorf("failed to get account keys: %w", err)
		}

		// Combine keys and deduplicate
		keyMap := make(map[int]model.PublicKey)
		for _, key := range globalKeys {
			keyMap[key.ID] = key
		}
		for _, key := range accountKeys {
			keyMap[key.ID] = key
		}

		// Convert to slice
		var allKeys []model.PublicKey
		for _, key := range keyMap {
			allKeys = append(allKeys, key)
		}

		return keySelectionLoadedMsg{
			keys: allKeys,
		}
	}
}

// performDecommissionWithKeys performs decommission with selected keys to remove
func (m *accountsModel) performDecommissionWithKeys() tea.Cmd {
	return func() tea.Msg {
		// Get active system key
		systemKey, err := db.GetActiveSystemKey()
		if err != nil || systemKey == nil {
			return fmt.Errorf("no active system key found")
		}

		// Build list of key IDs to remove (inverse of keys to keep)
		var keysToRemove []int
		for keyID, shouldKeep := range m.selectedKeysToKeep {
			if !shouldKeep {
				keysToRemove = append(keysToRemove, keyID)
			}
		}

		// Perform selective decommission
		options := deploy.DecommissionOptions{
			SkipRemoteCleanup: false,
			KeepFile:          true, // Keep file, remove selected keys
			Force:             false,
			DryRun:            false,
			SelectiveKeys:     keysToRemove,
		}
		result := deploy.DecommissionAccount(m.accountToDelete, systemKey.PrivateKey, options)

		return decommissionCompletedMsg{
			result: result,
		}
	}
}

// Message types for async operations
type keySelectionLoadedMsg struct {
	keys []model.PublicKey
}

type decommissionCompletedMsg struct {
	result deploy.DecommissionResult
}
