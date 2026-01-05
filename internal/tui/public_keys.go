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
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	internalkey "github.com/toeirei/keymaster/internal/crypto/ssh"
	"github.com/toeirei/keymaster/internal/i18n"
	"github.com/toeirei/keymaster/internal/model"
	"github.com/toeirei/keymaster/internal/ui"
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
	viewport         viewport.Model
	displayedKeys    []model.PublicKey // Filtered list
	cursor           int
	status           string
	err              error
	usageReportKey   model.PublicKey
	usageReportAccts []model.Account
	filter           string
	isFiltering      bool
	searcher         ui.KeySearcher
	// For delete confirmation
	isConfirmingDelete bool
	keyToDelete        model.PublicKey
	confirmCursor      int // 0 for No, 1 for Yes
	// For setting expiration on an existing key
	isSettingExpiry bool
	keyToExpire     model.PublicKey
	expireInput     textinput.Model
	width, height   int
}

// newPublicKeysModel creates a new model for the public key view, pre-loading keys from the database.
func newPublicKeysModel() publicKeysModel {
	return newPublicKeysModelWithSearcher(ui.DefaultKeySearcher())
}

// newPublicKeysModelWithSearcher creates a publicKeysModel that will use the
// provided KeySearcher for server-side searches. Pass nil to force local filtering.
func newPublicKeysModelWithSearcher(s ui.KeySearcher) publicKeysModel {
	m := publicKeysModel{
		viewport: viewport.New(0, 0),
		searcher: s,
	}
	// expiration input for the inline expiry modal
	ei := textinput.New()
	ei.Placeholder = "YYYY-MM-DD or RFC3339"
	ei.CharLimit = 64
	ei.Width = 30
	ei.Prompt = "Expires: "
	m.expireInput = ei
	var err error
	km := ui.DefaultKeyManager()
	if km == nil {
		m.err = fmt.Errorf("no key manager available")
		return m
	}
	m.keys, err = km.GetAllPublicKeys()
	if err != nil {
		m.err = err
	}
	m.rebuildDisplayedKeys()
	m.viewport.SetContent(m.listContentView())
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
		// Prefer server-side searcher when provided. If it returns non-empty
		// results, use them; otherwise fall back to local filtering.
		// Build localResults first so we can fall back if server search is
		// unavailable or returns no results. This avoids repeated ToLower in
		// the hot loop by creating a single lowercased representation per key.
		localResults := []model.PublicKey{}
		for _, key := range m.keys {
			combined := key.Comment + " " + key.Algorithm + " " + key.KeyData
			if ui.ContainsIgnoreCase(combined, m.filter) {
				localResults = append(localResults, key)
			}
		}

		// Prefer injected KeySearcher when present, otherwise use UI default.
		var searcher ui.KeySearcher
		if m.searcher != nil {
			searcher = m.searcher
		} else {
			searcher = ui.DefaultKeySearcher()
		}

		if searcher != nil {
			if res, err := searcher.SearchPublicKeys(m.filter); err == nil && len(res) > 0 {
				m.displayedKeys = res
			} else {
				// On error or empty server result, fall back to local filtering.
				m.displayedKeys = localResults
			}
		} else {
			m.displayedKeys = localResults
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
func (m *publicKeysModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Handle window size messages first, as they affect layout.
	if sizeMsg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = sizeMsg.Width
		m.height = sizeMsg.Height

		header := lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(m.headerView())
		footer := m.footerView()

		// The space for the panes is the total height minus header, footer, and the newlines around the main area.
		mainAreaHeight := m.height - lipgloss.Height(header) - lipgloss.Height(footer) - 2

		// The viewport's height is the available area minus chrome for the pane itself (borders, padding, title).
		m.viewport.Height = mainAreaHeight - 6
		m.viewport.Width = m.width/2 - 4 // Approximate width for left pane
	}

	if m.state == publicKeysFormView {
		if _, ok := msg.(publicKeyCreatedMsg); ok {
			m.state = publicKeysListView
			m.status = i18n.T("public_keys.status.add_success")
			km := ui.DefaultKeyManager()
			if km == nil {
				m.err = fmt.Errorf("no key manager available")
			} else {
				m.keys, m.err = km.GetAllPublicKeys()
			}
			m.rebuildDisplayedKeys()
			m.viewport.SetContent(m.listContentView())
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
		// Handle expiration modal first
		if m.isSettingExpiry {
			switch msg.String() {
			case "esc", "q":
				m.isSettingExpiry = false
				m.status = i18n.T("public_keys.status.expire_cancelled")
				return m, nil
			case "enter":
				// submit expiration
				val := strings.TrimSpace(m.expireInput.Value())
				var expiresAt time.Time
				if val != "" {
					if t, err := time.Parse(time.RFC3339, val); err == nil {
						expiresAt = t
					} else if t2, err2 := time.Parse("2006-01-02", val); err2 == nil {
						expiresAt = time.Date(t2.Year(), t2.Month(), t2.Day(), 0, 0, 0, 0, time.UTC)
					} else {
						m.status = i18n.T("public_keys.status.expire_invalid_format")
						return m, nil
					}
				}
				km := ui.DefaultKeyManager()
				if km == nil {
					m.err = fmt.Errorf("no key manager available")
					return m, nil
				}
				if err := km.SetPublicKeyExpiry(m.keyToExpire.ID, expiresAt); err != nil {
					m.err = err
				} else {
					m.status = i18n.T("public_keys.status.expire_success", m.keyToExpire.Comment)
					m.keys, m.err = km.GetAllPublicKeys()
					m.rebuildDisplayedKeys()
					m.viewport.SetContent(m.listContentView())
				}
				m.isSettingExpiry = false
				return m, nil
			default:
				var tcmd tea.Cmd
				m.expireInput, tcmd = m.expireInput.Update(msg)
				return m, tcmd
			}
		}
		// Handle delete confirmation
		if m.isConfirmingDelete {
			switch msg.String() {
			case "y":
				// Fallthrough to confirm
			case "n", "q", "esc":
				m.isConfirmingDelete = false
				m.status = i18n.T("public_keys.status.delete_cancelled")
				return m, nil
			case "right", "tab", "l":
				m.confirmCursor = 1 // Yes
				return m, nil
			case "left", "shift+tab", "h":
				m.confirmCursor = 0 // No
				return m, nil
			case "enter":
				if m.confirmCursor == 1 { // Yes is selected
					km := ui.DefaultKeyManager()
					if km == nil {
						m.err = fmt.Errorf("no key manager available")
					} else if err := km.DeletePublicKey(m.keyToDelete.ID); err != nil {
						m.err = err
					} else {
						m.status = i18n.T("public_keys.status.delete_success", m.keyToDelete.Comment)
						m.keys, m.err = km.GetAllPublicKeys()
						m.rebuildDisplayedKeys()
						m.viewport.SetContent(m.listContentView())
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
				m.viewport.SetContent(m.listContentView())
			case tea.KeyEnter:
				m.isFiltering = false
			case tea.KeyBackspace:
				if len(m.filter) > 0 {
					m.filter = m.filter[:len(m.filter)-1]
					m.rebuildDisplayedKeys()
					m.viewport.SetContent(m.listContentView())
				}
			case tea.KeyRunes:
				m.filter += string(msg.Runes)
				m.rebuildDisplayedKeys()
				m.viewport.SetContent(m.listContentView())
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
				m.viewport.SetContent(m.listContentView())
				return m, nil
			}
			return m, func() tea.Msg { return backToMenuMsg{} }
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.viewport.SetContent(m.listContentView())
				m.ensureCursorInView()
			}
		case "down", "j":
			if m.cursor < len(m.displayedKeys)-1 {
				m.cursor++
				m.viewport.SetContent(m.listContentView())
				m.ensureCursorInView()
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
				km := ui.DefaultKeyManager()
				if km == nil {
					m.err = fmt.Errorf("no key manager available")
				} else if err := km.TogglePublicKeyGlobal(keyToToggle.ID); err != nil {
					m.err = err
				} else {
					m.status = i18n.T("public_keys.status.toggle_success", keyToToggle.Comment)
					// Refresh the list to show the new status
					m.keys, m.err = km.GetAllPublicKeys()
					m.rebuildDisplayedKeys()
					m.viewport.SetContent(m.listContentView())
				}
			}
			return m, nil
		case "e": // Deactivate/reactivate via epoch-0 toggle
			if len(m.displayedKeys) > 0 {
				keyToToggle := m.displayedKeys[m.cursor]
				km := ui.DefaultKeyManager()
				if km == nil {
					m.err = fmt.Errorf("no key manager available")
				} else {
					now := time.Now().UTC()
					inactive := !keyToToggle.ExpiresAt.IsZero() && keyToToggle.ExpiresAt.Before(now)
					var newExp time.Time
					if inactive {
						// reactivate by clearing expiration
						newExp = time.Time{}
					} else {
						// deactivate by setting to unix epoch 0
						newExp = time.Unix(0, 0).UTC()
					}
					if err := km.SetPublicKeyExpiry(keyToToggle.ID, newExp); err != nil {
						m.err = err
					} else {
						if inactive {
							m.status = i18n.T("public_keys.status.reactivate_success", keyToToggle.Comment)
						} else {
							m.status = i18n.T("public_keys.status.deactivate_success", keyToToggle.Comment)
						}
						m.keys, m.err = km.GetAllPublicKeys()
						m.rebuildDisplayedKeys()
						m.viewport.SetContent(m.listContentView())
					}
				}
			}
			return m, nil
		case "x": // Set expiration for selected key
			if len(m.displayedKeys) > 0 {
				m.keyToExpire = m.displayedKeys[m.cursor]
				// prepopulate input with existing expiration if set
				if !m.keyToExpire.ExpiresAt.IsZero() {
					m.expireInput.SetValue(m.keyToExpire.ExpiresAt.UTC().Format(time.RFC3339))
				} else {
					m.expireInput.SetValue("")
				}
				m.expireInput.Focus()
				m.isSettingExpiry = true
			}
			return m, nil
		case "u": // Usage report
			if len(m.displayedKeys) > 0 {
				m.usageReportKey = m.displayedKeys[m.cursor]
				km := ui.DefaultKeyManager()
				if km == nil {
					m.err = fmt.Errorf("no key manager available")
					return m, nil
				}
				accounts, err := km.GetAccountsForKey(m.usageReportKey.ID)
				if err != nil {
					m.err = err
					return m, nil
				}
				m.usageReportAccts = accounts
				m.state = publicKeysUsageView
				m.status = ""
			}
			return m, nil
		case "c": // Copy to clipboard
			if len(m.displayedKeys) > 0 {
				keyToCopy := m.displayedKeys[m.cursor]
				keyContent := fmt.Sprintf("%s %s %s", keyToCopy.Algorithm, keyToCopy.KeyData, keyToCopy.Comment)
				err := clipboard.WriteAll(keyContent)
				if err != nil {
					m.status = i18n.T("public_keys.status.copy_failed", err.Error())
				} else {
					m.status = i18n.T("public_keys.status.copy_success", keyToCopy.Comment)
				}
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
func (m *publicKeysModel) ensureCursorInView() {
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
func (m *publicKeysModel) headerView() string {
	return mainTitleStyle.Render("🔑 " + i18n.T("public_keys.title"))
}

// View renders the public key management UI based on the current model state.
func (m *publicKeysModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	if m.isSettingExpiry {
		return m.viewExpiryModal()
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
func (m *publicKeysModel) viewConfirmation() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(i18n.T("public_keys.delete_confirm.title")))

	question := i18n.T("public_keys.delete_confirm.question", m.keyToDelete.Comment)
	b.WriteString(question)
	b.WriteString("\n\n" + i18n.T("public_keys.delete_confirm.warning"))
	b.WriteString("\n\n")

	var yesButton, noButton string
	if m.confirmCursor == 1 { // Yes
		yesButton = activeButtonStyle.Render(i18n.T("public_keys.delete_confirm.yes"))
		noButton = buttonStyle.Render(i18n.T("public_keys.delete_confirm.no"))
	} else { // No
		yesButton = buttonStyle.Render(i18n.T("public_keys.delete_confirm.yes"))
		noButton = activeButtonStyle.Render(i18n.T("public_keys.delete_confirm.no"))
	}

	buttons := lipgloss.JoinHorizontal(lipgloss.Top, noButton, "  ", yesButton)
	b.WriteString(buttons)

	b.WriteString("\n" + helpStyle.Render("\n"+i18n.T("public_keys.delete_confirm.help")))

	// Center the whole dialog
	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		dialogBoxStyle.Render(b.String()),
	)
}

// viewExpiryModal renders the modal dialog for setting an expiration on a key.
func (m *publicKeysModel) viewExpiryModal() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(i18n.T("public_keys.expire.title")))

	prompt := i18n.T("public_keys.expire.prompt", m.keyToExpire.Comment)
	b.WriteString(prompt + "\n\n")

	// expiration input
	b.WriteString(m.expireInput.View() + "\n\n")
	b.WriteString(helpStyle.Render(i18n.T("public_keys.expire.help")))
	b.WriteString("\n\n")
	// explanatory three-state help (Always Active / Still Active / Inactive)
	b.WriteString(helpStyle.Render(i18n.T("public_keys.expire.help_states")))

	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		dialogBoxStyle.Render(b.String()),
	)
}

// listContentView builds the string content for the list viewport.
func (m *publicKeysModel) listContentView() string {
	var b strings.Builder
	for i, key := range m.displayedKeys {
		var globalMarker string
		if key.IsGlobal {
			globalMarker = "🌐 "
		}
		var expiryMarker string
		if !key.ExpiresAt.IsZero() {
			now := time.Now().UTC()
			if key.ExpiresAt.After(now) {
				expiryMarker = "⏳ "
			} else {
				expiryMarker = "❌ "
			}
		}
		line := fmt.Sprintf("  %s%s%s (%s)", globalMarker, expiryMarker, key.Comment, key.Algorithm)
		if m.cursor == i {
			line = fmt.Sprintf("▸ %s%s%s (%s)", globalMarker, expiryMarker, key.Comment, key.Algorithm)
			b.WriteString(selectedItemStyle.Render(line) + "\n")
		} else {
			b.WriteString(itemStyle.Render(line) + "\n")
		}
	}
	return b.String()
}

// viewKeyList renders the main two-pane view with the key list and details.
func (m *publicKeysModel) viewKeyList() string {
	header := m.headerView()

	// List pane (left)
	var listContent string
	if len(m.displayedKeys) == 0 {
		if m.filter == "" {
			listContent = helpStyle.Render(i18n.T("public_keys.empty"))
		} else {
			listContent = helpStyle.Render(i18n.T("public_keys.empty_filtered"))
		}
	} else {
		listContent = m.viewport.View()
	}

	listPaneTitle := lipgloss.NewStyle().Bold(true).Render(i18n.T("public_keys.list_title"))
	listPane := lipgloss.JoinVertical(lipgloss.Left, listPaneTitle, "", listContent)

	paneStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorSubtle).
		Padding(1, 2)

	menuWidth := 48
	paneHeight := m.viewport.Height + 6

	leftPane := paneStyle.Width(menuWidth).Height(paneHeight).Render(listPane)
	detailWidth := m.width - menuWidth - 8

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

		if key.ExpiresAt.IsZero() {
			detailsItems = append(detailsItems, helpStyle.Render(i18n.T("public_keys.detail_expires", i18n.T("public_keys.never"))))
		} else {
			detailsItems = append(detailsItems, helpStyle.Render(i18n.T("public_keys.detail_expires_at", key.ExpiresAt.UTC().Format(time.RFC3339))))
		}

		// Calculate and display the fingerprint on the fly.
		parsedKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(key.String()))
		if err == nil {
			fingerprint := internalkey.FingerprintSHA256(parsedKey)
			detailsItems = append(detailsItems, helpStyle.Render(i18n.T("public_keys.detail_fingerprint", fingerprint)))
		}
	}

	// Only show filter status if filtering
	if m.isFiltering {
		detailsItems = append(detailsItems, "", helpStyle.Render(i18n.T("public_keys.filtering", m.filter)))
	}

	rightPane := paneStyle.Width(detailWidth).Height(paneHeight).Render(lipgloss.JoinVertical(lipgloss.Left, detailsItems...))

	mainArea := lipgloss.JoinHorizontal(lipgloss.Left, leftPane, rightPane)

	// Footer
	footer := m.footerView()

	return lipgloss.JoinVertical(lipgloss.Top, header, mainArea, footer)
}

// footerView renders the help text at the bottom of the page.
func (m *publicKeysModel) footerView() string {
	footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Background(lipgloss.Color("236")).Padding(0, 1).Italic(true)
	var filterStatus string
	if m.isFiltering {
		filterStatus = i18n.T("public_keys.filtering", m.filter)
	} else if m.filter != "" {
		filterStatus = i18n.T("public_keys.filter_active", m.filter)
	} else {
		filterStatus = i18n.T("public_keys.filter_hint")
	}
	return footerStyle.Render(fmt.Sprintf("%s  %s", i18n.T("public_keys.footer"), filterStatus))
}

// boolToYesNo is a helper function to convert a boolean to a localized "Yes" or "No".
func boolToYesNo(val bool) string {
	if val {
		return i18n.T("public_keys.yes")
	}
	return i18n.T("public_keys.no")
}

// viewUsageReport renders the view that shows which accounts a key is assigned to.
func (m *publicKeysModel) viewUsageReport() string {
	var b strings.Builder
	title := i18n.T("public_keys.usage_report.title", m.usageReportKey.Comment)
	b.WriteString(titleStyle.Render(title))

	if len(m.usageReportAccts) == 0 {
		b.WriteString("\n\n" + helpStyle.Render(i18n.T("public_keys.usage_report.not_assigned")))
	} else {
		b.WriteString("\n\n" + i18n.T("public_keys.usage_report.assigned_to") + "\n\n")
		for _, acc := range m.usageReportAccts {
			b.WriteString(itemStyle.Render("- "+acc.String()) + "\n")
		}
	}

	b.WriteString(helpStyle.Render("\n" + i18n.T("public_keys.usage_report.help")))
	return b.String()
}
