package tui

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/model"
)

type tagsViewModel struct {
	// Data
	sortedTags    []string
	accountsByTag map[string][]model.Account

	// State
	lines    []interface{} // Can be a string (tag) or model.Account
	expanded map[string]bool
	cursor   int
	err      error
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
	for _, tag := range m.sortedTags {
		m.lines = append(m.lines, tag)
		if m.expanded[tag] {
			// Add accounts for this tag
			for _, acc := range m.accountsByTag[tag] {
				m.lines = append(m.lines, acc)
			}
		}
	}
}

func (m tagsViewModel) Init() tea.Cmd {
	return nil
}

func (m tagsViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
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
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("🏷️  View Accounts by Tag"))
	b.WriteString("\n\n")

	if len(m.lines) == 0 {
		b.WriteString(helpStyle.Render("No accounts found."))
	}

	for i, lineItem := range m.lines {
		var lineStr string
		if tag, ok := lineItem.(string); ok {
			marker := "▶" // Collapsed
			if m.expanded[tag] {
				marker = "▼" // Expanded
			}
			lineStr = fmt.Sprintf("%s %s (%d accounts)", marker, tag, len(m.accountsByTag[tag]))
		} else if acc, ok := lineItem.(model.Account); ok {
			lineStr = "  • " + acc.String()
		}

		if m.cursor == i {
			b.WriteString(selectedItemStyle.Render("» " + lineStr))
		} else {
			b.WriteString(itemStyle.Render(lineStr))
		}
		b.WriteString("\n")
	}

	b.WriteString(helpStyle.Render("\n(enter to expand/collapse, q to quit)"))
	return b.String()
}
