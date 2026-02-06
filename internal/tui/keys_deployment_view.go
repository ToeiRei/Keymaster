// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package tui

import (
	"fmt"

	"github.com/toeirei/keymaster/internal/core"
	"github.com/toeirei/keymaster/internal/core/model"
	"github.com/toeirei/keymaster/internal/i18n"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type keysDeploymentViewModel struct {
	// Data
	deployments   []core.KeyDeploymentInfo
	sortedKeys    []string // key comments
	accountsByKey map[string][]model.Account
	keysByComment map[string]*model.PublicKey

	// State
	lines       []interface{} // Can be a string (key comment) or model.Account
	expanded    map[string]bool
	cursor      int
	filter      string
	isFiltering bool
	err         error
}

// newKeysDeploymentViewModel constructs a keys deployment view model.
func newKeysDeploymentViewModel() keysDeploymentViewModel {
	m := keysDeploymentViewModel{
		expanded: make(map[string]bool),
	}

	deployments, err := core.GetKeyDeployments()
	if err != nil {
		m.err = err
		return m
	}

	m.deployments = deployments
	m.accountsByKey = core.BuildAccountsByKey(deployments)
	m.sortedKeys = core.GetKeysWithAccounts(deployments)

	// Build keysByComment map for quick lookup
	m.keysByComment = make(map[string]*model.PublicKey)
	for _, dep := range deployments {
		m.keysByComment[dep.Key.Comment] = &dep.Key
	}

	m.rebuildLines()

	return m
}

// rebuildLines constructs the flattened list of items to display.
func (m *keysDeploymentViewModel) rebuildLines() {
	m.lines = []interface{}{}

	var keysToDisplay []string
	if m.filter != "" {
		for _, keyComment := range m.sortedKeys {
			if core.ContainsIgnoreCase(keyComment, m.filter) {
				keysToDisplay = append(keysToDisplay, keyComment)
			}
		}
	} else {
		keysToDisplay = m.sortedKeys
	}

	for _, keyComment := range keysToDisplay {
		m.lines = append(m.lines, keyComment)
		if m.expanded[keyComment] {
			// Add accounts for this key
			for _, acc := range m.accountsByKey[keyComment] {
				m.lines = append(m.lines, acc)
			}
		}
	}

	// Reset cursor if it's out of bounds after filtering
	if m.cursor >= len(m.lines) {
		m.cursor = 0
	}
}

func (m keysDeploymentViewModel) Init() tea.Cmd {
	return nil
}

func (m keysDeploymentViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// If we are in filtering mode, capture all input for the filter.
		if m.isFiltering {
			switch msg.Type {
			case tea.KeyEsc:
				m.isFiltering = false
				m.filter = ""
				m.rebuildLines()
			case tea.KeyEnter:
				m.isFiltering = false
			case tea.KeyBackspace:
				if len(m.filter) > 0 {
					m.filter = m.filter[:len(m.filter)-1]
					m.rebuildLines()
				}
			case tea.KeyRunes:
				m.filter += string(msg.Runes)
				m.rebuildLines()
			}
			return m, nil
		}

		// Not in filtering mode, handle commands.
		switch msg.String() {
		case "/":
			m.isFiltering = true
			m.filter = "" // Start with a fresh filter
			m.rebuildLines()
			return m, nil
		case "q", "esc":
			if m.filter != "" {
				m.filter = ""
				m.isFiltering = false
				m.rebuildLines()
				return m, nil
			}
			return m, func() tea.Msg { return backToMenuMsg{} }
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.lines)-1 {
				m.cursor++
			}
		case "enter":
			if m.cursor >= 0 && m.cursor < len(m.lines) {
				lineItem := m.lines[m.cursor]
				if keyComment, ok := lineItem.(string); ok {
					// It's a key comment, toggle it
					m.expanded[keyComment] = !m.expanded[keyComment]
					m.rebuildLines()
				}
			}
		}
	}
	return m, nil
}

func (m keysDeploymentViewModel) View() string {
	if m.err != nil {
		return errorStyle.Render(fmt.Sprintf("Error: %v\n", m.err))
	}

	title := mainTitleStyle.Render("🔑 " + i18n.T("keys_deployment_view.title"))
	var listItems []string
	if len(m.lines) == 0 {
		listItems = append(listItems, helpStyle.Render(i18n.T("keys_deployment_view.empty")))
	} else {
		for i, lineItem := range m.lines {
			var lineStr string
			if keyComment, ok := lineItem.(string); ok {
				marker := "▶"
				if m.expanded[keyComment] || m.filter != "" {
					marker = "▼"
				}
				// Get the key to show algorithm and global status
				key := m.keysByComment[keyComment]
				keyDisplay := keyComment
				if key != nil {
					globalIndicator := ""
					if key.IsGlobal {
						globalIndicator = " 🌍 GLOBAL"
					}
					keyDisplay = fmt.Sprintf("%s (%s)%s", keyComment, key.Algorithm, globalIndicator)
				}
				accountCount := len(m.accountsByKey[keyComment])
				lineStr = fmt.Sprintf("%s %s - %d accounts", marker, keyDisplay, accountCount)
			} else if acc, ok := lineItem.(model.Account); ok {
				// Show account with label if available
				if acc.Label != "" {
					lineStr = fmt.Sprintf("   • %s - %s@%s", acc.Label, acc.Username, acc.Hostname)
				} else {
					lineStr = fmt.Sprintf("   • %s@%s", acc.Username, acc.Hostname)
				}
			}
			if m.cursor == i {
				listItems = append(listItems, selectedItemStyle.Render("▸ "+lineStr))
			} else {
				listItems = append(listItems, itemStyle.Render("  "+lineStr))
			}
		}
	}
	paneStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorSubtle).Padding(1, 2)
	listPane := paneStyle.Width(90).Render(lipgloss.JoinVertical(lipgloss.Left, listItems...))

	// Help/footer line always at the bottom
	footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Background(lipgloss.Color("236")).Padding(0, 1).Italic(true)
	var filterStatus string
	if m.isFiltering {
		filterStatus = i18n.T("keys_deployment_view.filtering", m.filter)
	} else if m.filter != "" {
		filterStatus = i18n.T("keys_deployment_view.filter_active", m.filter)
	} else {
		filterStatus = i18n.T("keys_deployment_view.filter_hint")
	}
	left := i18n.T("keys_deployment_view.footer")
	// keys deployment view does not track full terminal width; use a reasonable default
	helpLine := footerStyle.Render(AlignFooter(left, filterStatus, 100))

	return lipgloss.JoinVertical(lipgloss.Left, title, "", listPane, "", helpLine)
}
