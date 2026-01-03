// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// package tui provides the terminal user interface for Keymaster.
// This file contains the logic for the key assignment view, which allows
// users to manage the many-to-many relationship between accounts and public keys.
package tui // import "github.com/toeirei/keymaster/internal/tui"

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/i18n"
	"github.com/toeirei/keymaster/internal/model"
)

// filterStyle was removed as it was unused; styles are created inline where needed.

// assignState represents the current focus of the assignment view.
type assignState int

const (
	// assignStateSelectAccount means the user is selecting an account from the left pane.
	assignStateSelectAccount assignState = iota
	// assignStateSelectKeys means the user is selecting keys for the chosen account in the right pane.
	assignStateSelectKeys
)

// assignKeysModel holds the state for the key assignment view.
// It manages the two-pane layout for selecting an account and then assigning keys to it.
type assignKeysModel struct {
	state           assignState
	accounts        []model.Account
	accountViewport viewport.Model
	keyViewport     viewport.Model
	keys            []model.PublicKey
	accountCursor   int
	keyCursor       int
	selectedAccount model.Account
	assignedKeys    map[int]struct{} // Set of key IDs assigned to the selected account
	status          string
	err             error
	accountFilter   string
	isFilteringAcct bool
	keyFilter       string
	isFilteringKey  bool
	width, height   int
}

// newAssignKeysModel creates a new model for the key assignment view, pre-loading accounts and keys.
func newAssignKeysModel() *assignKeysModel {
	m := &assignKeysModel{
		state:           assignStateSelectAccount,
		assignedKeys:    make(map[int]struct{}),
		accountViewport: viewport.New(0, 0),
		keyViewport:     viewport.New(0, 0),
	}

	var err error
	// Only show active accounts for assignment.
	m.accounts, err = db.GetAllActiveAccounts()
	if err != nil {
		m.err = err
		return m
	}
	// We also fetch all keys now, so we don't have to do it later.
	m.keys, err = db.GetAllPublicKeys()
	if err != nil {
		m.err = err
	}
	m.accountViewport.SetContent(m.accountListViewContent())
	return m
}

// Init initializes the model.
func (m *assignKeysModel) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model's state.
func (m *assignKeysModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	if sizeMsg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = sizeMsg.Width
		m.height = sizeMsg.Height
		headerHeight := lipgloss.Height(m.headerView())
		footerHeight := lipgloss.Height(m.footerView())
		mainAreaHeight := m.height - headerHeight - footerHeight - 2

		m.accountViewport.Height = mainAreaHeight - 6 // Pane borders, title, padding, filter bar
		m.accountViewport.Width = 40
		m.keyViewport.Height = mainAreaHeight - 6
		m.keyViewport.Width = m.width - m.accountViewport.Width - 10
	}

	// Route the message to the correct sub-updater based on the current state.
	var updatedModel tea.Model
	switch m.state {
	case assignStateSelectAccount:
		updatedModel, cmd = m.updateAccountSelection(msg)
		cmds = append(cmds, cmd)
		m.accountViewport, cmd = m.accountViewport.Update(msg)
		cmds = append(cmds, cmd)
	case assignStateSelectKeys:
		updatedModel, cmd = m.updateKeySelection(msg)
		cmds = append(cmds, cmd)
		m.keyViewport, cmd = m.keyViewport.Update(msg)
		cmds = append(cmds, cmd)
	default:
		return m, nil // Should be unreachable
	}

	return updatedModel, tea.Batch(cmds...)
}

func (m *assignKeysModel) updateAccountSelection(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Filtering mode for accounts
		if m.isFilteringAcct {
			switch msg.Type {
			case tea.KeyEsc:
				m.isFilteringAcct = false
			case tea.KeyEnter:
				m.isFilteringAcct = false
			case tea.KeyBackspace:
				if len(m.accountFilter) > 0 {
					m.accountFilter = m.accountFilter[:len(m.accountFilter)-1]
				}
			case tea.KeyRunes:
				m.accountFilter += string(msg.Runes)
			}
			// Reset cursor after filter change
			m.accountCursor = 0
			m.accountViewport.SetContent(m.accountListViewContent())
			return m, nil
		}
		switch msg.String() {
		case "/":
			m.isFilteringAcct = true
			return m, nil
		case "q", "esc":
			return m, func() tea.Msg { return backToMenuMsg{} }
		case "up", "k":
			if m.accountCursor > 0 {
				m.accountCursor--
				m.accountViewport.SetContent(m.accountListViewContent())
				m.ensureAccountCursorInView()
			}
		case "down", "j":
			if m.accountCursor < len(m.filteredAccounts())-1 {
				m.accountCursor++
				m.accountViewport.SetContent(m.accountListViewContent())
				m.ensureAccountCursorInView()
			}
		case "enter":
			filteredAccounts := m.filteredAccounts()
			if len(filteredAccounts) == 0 {
				return m, nil
			}
			// Clamp cursor just in case
			if m.accountCursor >= len(filteredAccounts) {
				m.accountCursor = len(filteredAccounts) - 1
			}
			if m.accountCursor < 0 {
				return m, nil // Should not happen
			}
			m.selectedAccount = filteredAccounts[m.accountCursor]
			m.state = assignStateSelectKeys
			m.keyCursor = 0
			m.status = ""

			// Refresh the key list to ensure we have the latest data
			keys, err := db.GetAllPublicKeys()
			if err != nil {
				m.err = fmt.Errorf("error refreshing key list: %v", err)
				return m, nil
			}
			m.keys = keys

			// Get currently assigned keys
			assigned, err := db.GetKeysForAccount(m.selectedAccount.ID)
			if err != nil {
				m.err = err
				return m, nil
			}
			m.assignedKeys = make(map[int]struct{})
			for _, key := range assigned {
				m.assignedKeys[key.ID] = struct{}{}
			}
			m.status = i18n.T("assign_keys.status.selected_account", m.selectedAccount.String(), len(assigned))
			m.keyViewport.SetContent(m.keyListViewContent())
			return m, nil
		}
	}
	return m, nil
}

// filteredAccounts returns a slice of accounts that match the current filter text.
func (m *assignKeysModel) filteredAccounts() []model.Account {
	if m.accountFilter == "" {
		return m.accounts
	}
	var filtered []model.Account
	for _, acc := range m.accounts {
		if containsIgnoreCase(acc.String(), m.accountFilter) {
			filtered = append(filtered, acc)
		}
	}
	return filtered
}

// filteredKeys returns a slice of public keys that match the current filter text.
func (m *assignKeysModel) filteredKeys() []model.PublicKey {
	if m.keyFilter == "" {
		return m.keys
	}
	var filtered []model.PublicKey
	for _, key := range m.keys {
		if containsIgnoreCase(key.Comment, m.keyFilter) || containsIgnoreCase(key.Algorithm, m.keyFilter) {
			filtered = append(filtered, key)
		}
	}
	return filtered
}

// containsIgnoreCase is a helper function for case-insensitive string searching.
func containsIgnoreCase(s, substr string) bool {
	if substr == "" {
		return true
	}
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

func (m *assignKeysModel) updateKeySelection(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Filtering mode for keys
		if m.isFilteringKey {
			switch msg.Type {
			case tea.KeyEsc:
				m.isFilteringKey = false
			case tea.KeyEnter:
				m.isFilteringKey = false
			case tea.KeyBackspace:
				if len(m.keyFilter) > 0 {
					m.keyFilter = m.keyFilter[:len(m.keyFilter)-1]
				}
			case tea.KeyRunes:
				m.keyFilter += string(msg.Runes)
			}
			// After a filter change, we should reset the cursor to avoid out-of-bounds.
			m.keyCursor = 0
			m.keyViewport.SetContent(m.keyListViewContent())
			return m, nil
		}
		switch msg.String() {
		case "/":
			m.isFilteringKey = true
			return m, nil
		case "q", "esc":
			m.state = assignStateSelectAccount
			m.status = ""
			m.err = nil
			m.keyFilter = ""
			m.isFilteringKey = false
			return m, nil
		case "up", "k":
			if m.keyCursor > 0 {
				m.keyCursor--
				m.keyViewport.SetContent(m.keyListViewContent())
				m.ensureKeyCursorInView()
			}
		case "down", "j":
			// Use filtered list for bounds checking
			if m.keyCursor < len(m.filteredKeys())-1 {
				m.keyCursor++
				m.keyViewport.SetContent(m.keyListViewContent())
				m.ensureKeyCursorInView()
			}
		case " ": // Only use space for toggling assignment
			filteredKeys := m.filteredKeys()
			if len(filteredKeys) == 0 || m.keyCursor >= len(filteredKeys) {
				return m, nil
			}
			selectedKey := filteredKeys[m.keyCursor]
			if _, assigned := m.assignedKeys[selectedKey.ID]; assigned {
				// Unassign
				m.status = i18n.T("assign_keys.status.unassign_attempt", selectedKey.Comment)
				if err := db.UnassignKeyFromAccount(selectedKey.ID, m.selectedAccount.ID); err != nil {
					m.err = err
					m.status = i18n.T("assign_keys.status.unassign_error", err)
				} else {
					delete(m.assignedKeys, selectedKey.ID)
					m.status = i18n.T("assign_keys.status.unassign_success", selectedKey.Comment)
				}
			} else {
				// Assign
				m.status = i18n.T("assign_keys.status.assign_attempt", selectedKey.Comment)
				// Verify key still exists
				exists := false
				for _, k := range m.keys {
					if k.ID == selectedKey.ID {
						exists = true
						break
					}
				}
				if !exists {
					m.err = fmt.Errorf("key ID %d no longer exists in memory", selectedKey.ID)
					m.status = i18n.T("assign_keys.status.assign_error_deleted", m.err)
					return m, nil
				}
				if err := db.AssignKeyToAccount(selectedKey.ID, m.selectedAccount.ID); err != nil {
					m.err = err
					m.status = i18n.T("assign_keys.status.assign_error", err)
				} else {
					m.assignedKeys[selectedKey.ID] = struct{}{}
					m.status = i18n.T("assign_keys.status.assign_success", selectedKey.Comment)
				}
			}
			m.keyViewport.SetContent(m.keyListViewContent())
			return m, nil
		}
	}
	return m, nil
}

func (m *assignKeysModel) accountListViewContent() string {
	var listItems []string
	accounts := m.filteredAccounts()
	if len(accounts) == 0 {
		listItems = append(listItems, helpStyle.Render(i18n.T("assign_keys.no_accounts")))
	} else {
		for i, acc := range accounts {
			line := acc.String()
			if m.accountCursor == i {
				line = "▸ " + line
				listItems = append(listItems, selectedItemStyle.Render(line))
			} else {
				listItems = append(listItems, itemStyle.Render("  "+line))
			}
		}
	}
	return lipgloss.JoinVertical(lipgloss.Left, listItems...)
}

func (m *assignKeysModel) keyListViewContent() string {
	var listItems []string
	keys := m.filteredKeys()

	if len(keys) == 0 {
		listItems = append(listItems, helpStyle.Render(i18n.T("assign_keys.no_keys")))
	} else {
		for i, key := range keys {
			checked := i18n.T("assign_keys.checkmark_unchecked")
			if _, ok := m.assignedKeys[key.ID]; ok {
				checked = i18n.T("assign_keys.checkmark_checked")
			}
			globalMark := ""
			if key.IsGlobal {
				globalMark = i18n.T("assign_keys.global_marker")
			}
			cursor := "  "
			if m.keyCursor == i {
				cursor = "▸ "
			}
			item := i18n.T("assign_keys.key_item_format", cursor, checked, globalMark, key.Comment, key.Algorithm)
			if key.IsGlobal {
				listItems = append(listItems, inactiveItemStyle.Render(item)) // Render global keys as inactive
			} else if _, ok := m.assignedKeys[key.ID]; ok {
				listItems = append(listItems, selectedItemStyle.Render(item))
			} else {
				listItems = append(listItems, itemStyle.Render(item))
			}
		}
	}
	return lipgloss.JoinVertical(lipgloss.Left, listItems...)
}

func (m *assignKeysModel) ensureAccountCursorInView() {
	top := m.accountViewport.YOffset
	bottom := top + m.accountViewport.Height - 1
	if m.accountCursor < top {
		m.accountViewport.YOffset = m.accountCursor
	} else if m.accountCursor > bottom {
		m.accountViewport.YOffset = m.accountCursor - m.accountViewport.Height + 1
	}
}

func (m *assignKeysModel) ensureKeyCursorInView() {
	top := m.keyViewport.YOffset
	bottom := top + m.keyViewport.Height - 1
	if m.keyCursor < top {
		m.keyViewport.YOffset = m.keyCursor
	} else if m.keyCursor > bottom {
		m.keyViewport.YOffset = m.keyCursor - m.keyViewport.Height + 1
	}
}

func (m *assignKeysModel) headerView() string {
	return mainTitleStyle.Render("🔑 " + i18n.T("menu.assign_keys"))
}

func (m *assignKeysModel) footerView() string {
	footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Background(lipgloss.Color("236")).Padding(0, 1).Italic(true)
	var filterStatus string
	var helpText string
	if m.state == assignStateSelectKeys {
		if m.isFilteringKey {
			filterStatus = i18n.T("assign_keys.filtering", m.keyFilter)
		} else if m.keyFilter != "" {
			filterStatus = i18n.T("assign_keys.filter_active", m.keyFilter)
		} else {
			filterStatus = i18n.T("assign_keys.search_hint")
		}
		helpText = fmt.Sprintf("%s  %s", i18n.T("assign_keys.help_bar_keys"), filterStatus)
		if m.status != "" {
			return statusMessageStyle.Render(m.status)
		}
	} else {
		// This logic is duplicated in View() for the filter bar, but that's okay.
		// It's clearer than passing it around.
		if m.isFilteringAcct {
			filterStatus = i18n.T("assign_keys.filtering", m.accountFilter)
		} else if m.accountFilter != "" {
			filterStatus = i18n.T("assign_keys.filter_active", m.accountFilter)
		} else {
			filterStatus = i18n.T("assign_keys.search_hint")
		}
		helpText = fmt.Sprintf("%s  %s", i18n.T("assign_keys.help_bar_accounts"), filterStatus)
	}
	return footerStyle.Render(helpText)
}

func (m *assignKeysModel) View() string {
	header := m.headerView()
	footer := m.footerView()

	paneStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorSubtle).
		Padding(1, 2)

	// Left Pane
	accountListTitle := lipgloss.NewStyle().Bold(true).Render(i18n.T("assign_keys.accounts_title"))
	var accountFilterBar string
	if m.isFilteringAcct {
		accountFilterBar = i18n.T("assign_keys.filtering", m.accountFilter)
	} else if m.accountFilter != "" {
		accountFilterBar = i18n.T("assign_keys.filter_active", m.accountFilter)
	} else {
		accountFilterBar = i18n.T("assign_keys.search_hint")
	}
	leftPaneContent := lipgloss.JoinVertical(lipgloss.Left, accountListTitle, "", m.accountViewport.View(), "", accountFilterBar)
	paneHeight := m.accountViewport.Height + 6
	leftPane := paneStyle.Width(m.accountViewport.Width + 4).Height(paneHeight).Render(leftPaneContent)

	// Right Pane
	var rightPane string
	if m.state == assignStateSelectKeys {
		keyPaneTitle := lipgloss.NewStyle().Bold(true).Render(i18n.T("assign_keys.keys_title", m.selectedAccount.String()))
		var keyFilterBar string
		if m.isFilteringKey {
			keyFilterBar = i18n.T("assign_keys.filtering", m.keyFilter)
		} else if m.keyFilter != "" {
			keyFilterBar = i18n.T("assign_keys.filter_active", m.keyFilter)
		} else {
			keyFilterBar = i18n.T("assign_keys.search_hint")
		}
		rightPaneContent := lipgloss.JoinVertical(lipgloss.Left, keyPaneTitle, "", m.keyViewport.View(), "", keyFilterBar)
		rightPane = paneStyle.Width(m.keyViewport.Width + 4).Height(paneHeight).Render(rightPaneContent)
	} else {
		// Render an empty placeholder pane
		placeholderTitle := lipgloss.NewStyle().Bold(true).Render(i18n.T("assign_keys.keys_title_short"))
		placeholderContent := lipgloss.JoinVertical(lipgloss.Left, placeholderTitle, "", helpStyle.Render(i18n.T("assign_keys.select_account_prompt")))
		rightPane = paneStyle.Width(m.keyViewport.Width + 4).Height(paneHeight).Render(placeholderContent)
	}

	mainArea := lipgloss.JoinHorizontal(lipgloss.Left, leftPane, rightPane)

	return lipgloss.JoinVertical(lipgloss.Top, header, mainArea, footer)
}
