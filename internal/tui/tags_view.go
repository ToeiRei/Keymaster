// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/toeirei/keymaster/internal/i18n"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/model"
)

type tagsViewModel struct {
	// Data
	sortedTags    []string
	accountsByTag map[string][]model.Account

	// State
	lines       []interface{} // Can be a string (tag) or model.Account
	expanded    map[string]bool
	cursor      int
	filter      string
	isFiltering bool
	err         error
}

func newTagsViewModel() tagsViewModel {
	m := tagsViewModel{
		expanded: make(map[string]bool),
	}

	accounts, err := db.GetAllAccounts()
	if err != nil {
		m.err = err
		return m
	}

	accountsByTag := make(map[string][]model.Account)
	tagSet := make(map[string]struct{})

	// Special tag for untagged accounts
	untagged := "(no tags)"
	hasUntagged := false

	for _, acc := range accounts {
		if acc.Tags == "" {
			accountsByTag[untagged] = append(accountsByTag[untagged], acc)
			hasUntagged = true
			continue
		}
		tags := strings.Split(acc.Tags, ",")
		for _, tag := range tags {
			trimmedTag := strings.TrimSpace(tag)
			if trimmedTag == "" {
				continue
			}
			accountsByTag[trimmedTag] = append(accountsByTag[trimmedTag], acc)
			tagSet[trimmedTag] = struct{}{}
		}
	}

	sortedTags := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		sortedTags = append(sortedTags, tag)
	}
	sort.Strings(sortedTags)

	// Add untagged to the end if it exists
	if hasUntagged {
		sortedTags = append(sortedTags, untagged)
	}

	m.accountsByTag = accountsByTag
	m.sortedTags = sortedTags
	m.rebuildLines()

	return m
}

// rebuildLines constructs the flattened list of items to display.
func (m *tagsViewModel) rebuildLines() {
	m.lines = []interface{}{}

	var tagsToDisplay []string
	if m.filter != "" {
		for _, tag := range m.sortedTags {
			if strings.Contains(strings.ToLower(tag), strings.ToLower(m.filter)) {
				tagsToDisplay = append(tagsToDisplay, tag)
			}
		}
	} else {
		tagsToDisplay = m.sortedTags
	}

	for _, tag := range tagsToDisplay {
		m.lines = append(m.lines, tag)
		if m.expanded[tag] {
			// Add accounts for this tag
			for _, acc := range m.accountsByTag[tag] {
				m.lines = append(m.lines, acc)
			}
		}
	}

	// Reset cursor if it's out of bounds after filtering
	if m.cursor >= len(m.lines) {
		m.cursor = 0
	}
}

func (m tagsViewModel) Init() tea.Cmd {
	return nil
}

func (m tagsViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
				if tag, ok := lineItem.(string); ok {
					// It's a tag, toggle it
					m.expanded[tag] = !m.expanded[tag]
					m.rebuildLines()
				}
			}
		}
	}
	return m, nil
}

func (m tagsViewModel) View() string {
	if m.err != nil {
		return errorStyle.Render(fmt.Sprintf("Error: %v\n", m.err))
	}

	title := mainTitleStyle.Render("🏷️  " + i18n.T("tags_view.title"))
	var listItems []string
	if len(m.lines) == 0 {
		listItems = append(listItems, helpStyle.Render(i18n.T("tags_view.empty")))
	} else {
		for i, lineItem := range m.lines {
			var lineStr string
			if tag, ok := lineItem.(string); ok {
				marker := "▶"
				if m.expanded[tag] || m.filter != "" {
					marker = "▼"
				}
				lineStr = fmt.Sprintf("%s %s (%d)", marker, tag, len(m.accountsByTag[tag]))
			} else if acc, ok := lineItem.(model.Account); ok {
				lineStr = "   • " + acc.String()
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
		filterStatus = fmt.Sprintf(i18n.T("tags_view.filtering"), m.filter)
	} else if m.filter != "" {
		filterStatus = fmt.Sprintf(i18n.T("tags_view.filter_active"), m.filter)
	} else {
		filterStatus = i18n.T("tags_view.filter_hint")
	}
	helpLine := footerStyle.Render(fmt.Sprintf("%s  %s", i18n.T("tags_view.footer"), filterStatus))

	return lipgloss.JoinVertical(lipgloss.Left, title, "", listPane, "", helpLine)
}
