// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package tui

import (
	"fmt"

	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/i18n"
	"github.com/toeirei/keymaster/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/internal/core"
	"github.com/toeirei/keymaster/internal/model"
)

type hostsViewModel struct {
	// Data
	sortedHosts    []string
	accountsByHost map[string][]model.Account

	// State
	lines       []interface{} // Can be a string (hostname) or model.Account
	expanded    map[string]bool
	cursor      int
	filter      string
	isFiltering bool
	err         error
	searcher    db.AccountSearcher
}

// newHostsViewModelWithSearcher constructs a hosts view model and allows an
// optional AccountSearcher to be provided for server-side operations.
func newHostsViewModelWithSearcher(s db.AccountSearcher) hostsViewModel {
	m := hostsViewModel{
		expanded: make(map[string]bool),
		searcher: s,
	}

	var accounts []model.Account
	var err error
	if s != nil {
		accounts, err = s.SearchAccounts("")
	} else if def := ui.DefaultAccountSearcher(); def != nil {
		accounts, err = def.SearchAccounts("")
	}
	if err != nil {
		m.err = err
		return m
	}

	// Use core helpers to build hostname lists and groupings.
	m.accountsByHost = core.BuildAccountsByHost(accounts)
	m.sortedHosts = core.UniqueHosts(accounts)
	m.rebuildLines()

	return m
}

// rebuildLines constructs the flattened list of items to display.
func (m *hostsViewModel) rebuildLines() {
	m.lines = []interface{}{}

	var hostsToDisplay []string
	if m.filter != "" {
		for _, host := range m.sortedHosts {
			if core.ContainsIgnoreCase(host, m.filter) {
				hostsToDisplay = append(hostsToDisplay, host)
			}
		}
	} else {
		hostsToDisplay = m.sortedHosts
	}

	for _, host := range hostsToDisplay {
		m.lines = append(m.lines, host)
		if m.expanded[host] {
			// Add accounts for this host
			for _, acc := range m.accountsByHost[host] {
				m.lines = append(m.lines, acc)
			}
		}
	}

	// Reset cursor if it's out of bounds after filtering
	if m.cursor >= len(m.lines) {
		m.cursor = 0
	}
}

func (m hostsViewModel) Init() tea.Cmd {
	return nil
}

func (m hostsViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
				if host, ok := lineItem.(string); ok {
					// It's a hostname, toggle it
					m.expanded[host] = !m.expanded[host]
					m.rebuildLines()
				}
			}
		}
	}
	return m, nil
}

func (m hostsViewModel) View() string {
	if m.err != nil {
		return errorStyle.Render(fmt.Sprintf("Error: %v\n", m.err))
	}

	title := mainTitleStyle.Render("🖥️  " + i18n.T("hosts_view.title"))
	var listItems []string
	if len(m.lines) == 0 {
		listItems = append(listItems, helpStyle.Render(i18n.T("hosts_view.empty")))
	} else {
		for i, lineItem := range m.lines {
			var lineStr string
			if host, ok := lineItem.(string); ok {
				marker := "▶"
				if m.expanded[host] || m.filter != "" {
					marker = "▼"
				}
				// Get a representative label for this host (use first account's label)
				accounts := m.accountsByHost[host]
				hostDisplay := host
				if len(accounts) > 0 && accounts[0].Label != "" {
					hostDisplay = fmt.Sprintf("%s - %s", accounts[0].Label, host)
				}
				lineStr = fmt.Sprintf("%s %s (%d)", marker, hostDisplay, len(accounts))
			} else if acc, ok := lineItem.(model.Account); ok {
				// Show just username@hostname for account items
				lineStr = fmt.Sprintf("   • %s@%s", acc.Username, acc.Hostname)
			}
			if m.cursor == i {
				listItems = append(listItems, selectedItemStyle.Render("▸ "+lineStr))
			} else {
				listItems = append(listItems, itemStyle.Render("  "+lineStr))
			}
		}
	}
	paneStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorSubtle).Padding(1, 2)
	listPane := paneStyle.Width(60).Render(lipgloss.JoinVertical(lipgloss.Left, listItems...))

	// Help/footer line always at the bottom
	footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Background(lipgloss.Color("236")).Padding(0, 1).Italic(true)
	var filterStatus string
	if m.isFiltering {
		filterStatus = i18n.T("hosts_view.filtering", m.filter)
	} else if m.filter != "" {
		filterStatus = i18n.T("hosts_view.filter_active", m.filter)
	} else {
		filterStatus = i18n.T("hosts_view.filter_hint")
	}
	left := i18n.T("hosts_view.footer")
	// hosts view does not track full terminal width; use a reasonable default
	helpLine := footerStyle.Render(AlignFooter(left, filterStatus, 80))

	return lipgloss.JoinVertical(lipgloss.Left, title, "", listPane, "", helpLine)
}
