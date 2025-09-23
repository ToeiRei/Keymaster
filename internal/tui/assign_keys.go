// removed stray brace

package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/i18n"
	"github.com/toeirei/keymaster/internal/model"
)

var (
	filterStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Italic(true)
)

type assignState int

const (
	assignStateSelectAccount assignState = iota
	assignStateSelectKeys
)

type assignKeysModel struct {
	state           assignState
	accounts        []model.Account
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
}

func newAssignKeysModel() assignKeysModel {
	m := assignKeysModel{
		state:        assignStateSelectAccount,
		assignedKeys: make(map[int]struct{}),
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
	return m
}

func (m assignKeysModel) Init() tea.Cmd {
	return nil
}

func (m assignKeysModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.state {
	case assignStateSelectAccount:
		return m.updateAccountSelection(msg)
	case assignStateSelectKeys:
		return m.updateKeySelection(msg)
	}
	return m, nil
}

// updateAccountSelection handles input when the user is selecting an account.
func (m assignKeysModel) updateAccountSelection(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			}
		case "down", "j":
			if m.accountCursor < len(m.filteredAccounts())-1 {
				m.accountCursor++
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
			assigned, err := db.GetKeysForAccount(m.selectedAccount.ID)
			if err != nil {
				m.err = err
				return m, nil
			}
			m.assignedKeys = make(map[int]struct{})
			for _, key := range assigned {
				m.assignedKeys[key.ID] = struct{}{}
			}
			return m, nil
		}
	}
	return m, nil
}

// Filtered account list
func (m assignKeysModel) filteredAccounts() []model.Account {
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

// Filtered key list
func (m assignKeysModel) filteredKeys() []model.PublicKey {
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

func containsIgnoreCase(s, substr string) bool {
	if substr == "" {
		return true
	}
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// updateKeySelection handles input when the user is selecting keys to assign.
func (m assignKeysModel) updateKeySelection(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			}
		case "down", "j":
			// Use filtered list for bounds checking
			if m.keyCursor < len(m.filteredKeys())-1 {
				m.keyCursor++
			}
		case " ", "enter":
			filteredKeys := m.filteredKeys()
			if len(filteredKeys) == 0 || m.keyCursor >= len(filteredKeys) {
				return m, nil
			}
			selectedKey := filteredKeys[m.keyCursor]
			if _, assigned := m.assignedKeys[selectedKey.ID]; assigned {
				// Unassign
				if err := db.UnassignKeyFromAccount(m.selectedAccount.ID, selectedKey.ID); err != nil {
					m.err = err
				} else {
					delete(m.assignedKeys, selectedKey.ID)
					m.status = fmt.Sprintf("Unassigned key: %s", selectedKey.Comment)
				}
			} else {
				// Assign
				if err := db.AssignKeyToAccount(m.selectedAccount.ID, selectedKey.ID); err != nil {
					m.err = err
				} else {
					m.assignedKeys[selectedKey.ID] = struct{}{}
					m.status = fmt.Sprintf("Assigned key: %s", selectedKey.Comment)
				}
			}
			return m, nil
		}
	}
	return m, nil
}

func (m assignKeysModel) viewAccountList() string {
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
	filterBar := ""
	if m.isFilteringAcct {
		filterBar = filterStyle.Render("/" + m.accountFilter)
	} else {
		filterBar = filterStyle.Render(i18n.T("assign_keys.search_hint"))
	}
	listPaneTitle := lipgloss.NewStyle().Bold(true).Render(i18n.T("assign_keys.accounts_title"))
	listPane := lipgloss.JoinVertical(lipgloss.Left, listPaneTitle, "", lipgloss.JoinVertical(lipgloss.Left, listItems...), "", filterBar)
	paneStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorSubtle).Padding(1, 2)
	return paneStyle.Width(40).Render(listPane)
}

func (m assignKeysModel) viewKeySelection() string {
	var listItems []string
	keys := m.filteredKeys()
	if len(keys) == 0 {
		listItems = append(listItems, helpStyle.Render(i18n.T("assign_keys.no_keys")))
	} else {
		for i, key := range keys {
			checked := "○"
			if _, ok := m.assignedKeys[key.ID]; ok {
				checked = "✔"
			}
			globalMark := ""
			if key.IsGlobal {
				globalMark = "🌍 "
			}
			cursor := "  "
			if m.keyCursor == i {
				cursor = "▸ "
			}
			item := cursor + checked + " " + globalMark + key.Comment + " (" + key.Algorithm + ")"
			if key.IsGlobal {
				listItems = append(listItems, inactiveItemStyle.Render(item))
			} else if _, ok := m.assignedKeys[key.ID]; ok {
				listItems = append(listItems, selectedItemStyle.Render(item))
			} else {
				listItems = append(listItems, itemStyle.Render(item))
			}
		}
	}
	filterBar := ""
	if m.isFilteringKey {
		filterBar = filterStyle.Render("/" + m.keyFilter)
	} else {
		filterBar = filterStyle.Render(i18n.T("assign_keys.search_hint"))
	}
	status := ""
	if m.status != "" {
		status = statusMessageStyle.Render(m.status)
	}
	listPaneTitle := lipgloss.NewStyle().Bold(true).Render(i18n.T("assign_keys.keys_title_short"))
	listPane := lipgloss.JoinVertical(lipgloss.Left, listPaneTitle, "", lipgloss.JoinVertical(lipgloss.Left, listItems...), "", filterBar, "", status)
	paneStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorSubtle).Padding(1, 2)
	return paneStyle.Width(60).Render(listPane)
}

func (m assignKeysModel) View() string {
	left := m.viewAccountList()
	right := ""
	if m.state == assignStateSelectKeys {
		right = m.viewKeySelection()
	}
	mainArea := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	// Compose help and filter status on one line, matching accounts.go style
	footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Background(lipgloss.Color("236")).Padding(0, 1).Italic(true)
	var filterStatus string
	if m.isFilteringAcct {
		filterStatus = i18n.T("assign_keys.filtering") + m.accountFilter
	} else if m.accountFilter != "" {
		filterStatus = i18n.T("assign_keys.filter_active") + m.accountFilter
	} else {
		filterStatus = i18n.T("assign_keys.search_hint")
	}
	helpLine := footerStyle.Render(i18n.T("assign_keys.help_bar") + "  " + filterStatus)

	return lipgloss.JoinVertical(lipgloss.Left, mainArea, "", helpLine)
}
