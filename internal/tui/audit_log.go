// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// package tui provides the terminal user interface for Keymaster.
// This file contains the logic for the audit log view, which displays a
// filterable, color-coded table of all actions taken within the application.
package tui // import "github.com/toeirei/keymaster/internal/tui"

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/internal/core"
	"github.com/toeirei/keymaster/internal/i18n"
	"github.com/toeirei/keymaster/internal/model"
	"github.com/toeirei/keymaster/internal/ui"
)

// Risk-based color styles for audit log actions.
var (
	auditHighRiskStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196")) // Bright Red
	auditMediumRiskStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("220")) // Amber/Yellow
	auditLowRiskStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("40"))  // Green
	auditInfoStyle       = lipgloss.NewStyle().Foreground(colorSubtle)           // Gray
)

// auditActionStyle returns a specific color-coded style for an audit log action
// based on its perceived risk level (e.g., destructive actions are red).
func auditActionStyle(action string) lipgloss.Style {
	switch core.AuditActionRisk(action) {
	case "high":
		return auditHighRiskStyle
	case "medium":
		return auditMediumRiskStyle
	case "low":
		return auditLowRiskStyle
	default:
		return auditInfoStyle
	}
}

// auditLogModel holds the state for the audit log view.
// It manages the table, filtering, and custom rendering logic.
type auditLogModel struct {
	table       table.Model
	styles      table.Styles
	allEntries  []model.AuditLogEntry // Master list of all log entries
	filter      string
	filterCol   int // 0=all, 1=timestamp, 2=user, 3=action, 4=details
	isFiltering bool
	err         error
	searcher    ui.AuditSearcher
}

// newAuditLogModelWithSearcher creates a new model for the audit log view, loading entries from the provided searcher.
func newAuditLogModelWithSearcher(searcher ui.AuditSearcher) *auditLogModel {
	m := &auditLogModel{searcher: searcher}
	var entries []model.AuditLogEntry
	var err error
	if searcher != nil {
		entries, err = searcher.GetAllAuditLogEntries()
	} else {
		// Fallback to the direct DB helper when no searcher is provided.
		entries, err = ui.GetAllAuditLogEntries()
	}
	if err != nil {
		m.err = err
		return m
	}
	m.allEntries = entries

	columns := []table.Column{
		{Title: i18n.T("audit_log.header.timestamp"), Width: 20},
		{Title: i18n.T("audit_log.header.user"), Width: 15},
		{Title: i18n.T("audit_log.header.action"), Width: 25},
		{Title: i18n.T("audit_log.header.details"), Width: 60},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(15), // Placeholder height
	)

	// --- Styles ---
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(colorSubtle).
		BorderBottom(true).
		Bold(true)
	// Harden Selected style: set foreground, background, reset, and clear all attributes
	s.Selected = lipgloss.NewStyle().
		Foreground(colorWhite).
		Background(colorHighlight).
		Bold(false).
		Faint(false).
		Italic(false).
		Underline(false).
		Strikethrough(false).
		Blink(false).
		Reverse(false)
	t.SetStyles(s)
	m.styles = s

	m.table = t
	m.rebuildTableRows()
	return m
}

// rebuildTableRows constructs the table rows from the master list of entries,
// applying the current filter.
func (m *auditLogModel) rebuildTableRows() {
	var rows []table.Row
	for _, entry := range m.allEntries {
		match := false
		// Build combined strings and use ui.ContainsIgnoreCase to avoid
		// repeated strings.ToLower calls in the hot loop.
		switch m.filterCol {
		case 0:
			combined := entry.Timestamp + " " + entry.Username + " " + entry.Action + " " + entry.Details
			match = core.ContainsIgnoreCase(combined, m.filter)
		case 1:
			match = core.ContainsIgnoreCase(entry.Timestamp, m.filter)
		case 2:
			match = core.ContainsIgnoreCase(entry.Username, m.filter)
		case 3:
			match = core.ContainsIgnoreCase(entry.Action, m.filter)
		case 4:
			match = core.ContainsIgnoreCase(entry.Details, m.filter)
		}
		if m.filter != "" && !match {
			continue
		}
		ts := entry.Timestamp
		if len(ts) > 19 {
			ts = ts[:19]
		}
		// Store only plain text in the row
		rows = append(rows, table.Row{ts, entry.Username, entry.Action, entry.Details})
	}
	m.table.SetRows(rows)
}

// padCell is a helper to pad a string to a certain width for table layout.
func padCell(s string, width int) string {
	if len([]rune(s)) >= width {
		return string([]rune(s)[:width])
	}
	return s + strings.Repeat(" ", width-len([]rune(s)))
}

// Init initializes the model.
func (m *auditLogModel) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model's state.
func (m *auditLogModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.table.SetHeight(msg.Height - 6)
		m.table.SetWidth(msg.Width - 4)
	case tea.KeyMsg:
		if m.isFiltering {
			switch msg.Type {
			case tea.KeyEsc:
				m.isFiltering = false
				m.filter = ""
				m.rebuildTableRows()
			case tea.KeyEnter:
				m.isFiltering = false
			case tea.KeyBackspace:
				if len(m.filter) > 0 {
					m.filter = m.filter[:len(m.filter)-1]
					m.table.GotoTop()
					m.rebuildTableRows()
				}
			case tea.KeyRunes:
				m.filter += string(msg.Runes)
				m.table.GotoTop()
				m.rebuildTableRows()
			case tea.KeyTab:
				m.filterCol = (m.filterCol + 1) % 5
				m.rebuildTableRows()
			case tea.KeyShiftTab:
				m.filterCol = (m.filterCol + 4) % 5
				m.rebuildTableRows()
			}
			return m, nil
		}
		switch msg.String() {
		case "/":
			m.isFiltering = true
			m.filter = ""
			m.rebuildTableRows()
			return m, nil
		case "q", "esc":
			if m.filter != "" {
				m.filter = ""
				m.isFiltering = false
				m.rebuildTableRows()
				return m, nil
			}
			return m, func() tea.Msg { return backToMenuMsg{} }
		}
	}
	m.table, cmd = m.table.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

// View renders the audit log UI.
func (m *auditLogModel) View() string {
	if m.err != nil {
		return errorStyle.Render(fmt.Sprintf("Error loading audit log: %v", m.err))
	}
	var b strings.Builder
	b.WriteString(titleStyle.Render("📜 "+i18n.T("audit_log.title")) + "\n\n")
	if len(m.table.Rows()) == 0 {
		b.WriteString(helpStyle.Render(i18n.T("audit_log.empty")))
	} else {
		b.WriteString(m.renderAuditLogTable())
	}
	// Always show the styled help line/footer at the bottom, single line, styled
	footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Background(lipgloss.Color("236")).Padding(0, 1).Italic(true)
	b.WriteString("\n")
	b.WriteString(footerStyle.Render(m.footerLine()))
	return b.String()
}

// footerLine generates the dynamic help/status line for the footer.
func (m *auditLogModel) footerLine() string {
	var filterStatus string
	colNames := []string{
		i18n.T("all"),
		i18n.T("audit_log.header.timestamp"),
		i18n.T("audit_log.header.user"),
		i18n.T("audit_log.header.action"),
		i18n.T("audit_log.header.details"),
	}
	if m.isFiltering {
		filterStatus = i18n.T("audit_log.filtering", colNames[m.filterCol], m.filter)
	} else if m.filter != "" {
		filterStatus = i18n.T("audit_log.filter_active", colNames[m.filterCol], m.filter)
	} else {
		filterStatus = i18n.T("audit_log.filter_hint")
	}
	// Single line: help and filter status
	return fmt.Sprintf("%s  %s", i18n.T("audit_log.footer"), filterStatus)
}

// renderAuditLogTable provides a custom rendering implementation for the table.
// This is necessary to apply custom row-level styling (e.g., color-coding actions)
// which the standard bubbletea table does not support directly.
func (m *auditLogModel) renderAuditLogTable() string {
	var out strings.Builder
	rows := m.table.Rows()
	cursor := m.table.Cursor()
	height := m.table.Height()
	// Render header
	out.WriteString(m.styles.Header.Render(
		padCell(m.table.Columns()[0].Title, m.table.Columns()[0].Width)+
			padCell(m.table.Columns()[1].Title, m.table.Columns()[1].Width)+
			padCell(m.table.Columns()[2].Title, m.table.Columns()[2].Width)+
			padCell(m.table.Columns()[3].Title, m.table.Columns()[3].Width),
	) + "\n")

	// Calculate visible window for scrolling
	start := 0
	if len(rows) > height {
		// Try to keep the selected row in the middle of the viewport
		mid := height / 2
		if cursor > mid {
			start = cursor - mid
			if start+height > len(rows) {
				start = len(rows) - height
			}
		}
		if start < 0 {
			start = 0
		}
	}
	end := start + height
	if end > len(rows) {
		end = len(rows)
	}
	// Render visible rows
	for i := start; i < end; i++ {
		row := rows[i]
		rendered := lipgloss.JoinHorizontal(lipgloss.Top,
			padCell(row[0], m.table.Columns()[0].Width),
			padCell(row[1], m.table.Columns()[1].Width),
			padCell(row[2], m.table.Columns()[2].Width),
			padCell(row[3], m.table.Columns()[3].Width),
		)
		if i == cursor {
			// Selected row: use table's Selected style for the whole row
			out.WriteString(m.styles.Selected.Render(rendered) + "\n")
		} else {
			// Non-selected: color the whole row based on action type
			style := auditActionStyle(row[2])
			out.WriteString(style.Render(rendered) + "\n")
		}
	}
	return out.String()
}
