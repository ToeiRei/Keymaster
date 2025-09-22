package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/model"
)

type auditLogModel struct {
	table       table.Model
	allEntries  []model.AuditLogEntry // Master list of all log entries
	filter      string
	isFiltering bool
	err         error
}

func newAuditLogModel() auditLogModel {
	m := auditLogModel{}
	entries, err := db.GetAllAuditLogEntries()
	if err != nil {
		m.err = err
		return m
	}
	m.allEntries = entries

	columns := []table.Column{
		{Title: "Timestamp", Width: 20},
		{Title: "User", Width: 15},
		{Title: "Action", Width: 25},
		{Title: "Details", Width: 60},
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
	s.Selected = s.Selected.
		Foreground(colorWhite).
		Background(colorHighlight).
		Bold(false)
	t.SetStyles(s)

	m.table = t
	m.rebuildTableRows()
	return m
}

// rebuildTableRows filters the master list of entries and populates the table.
func (m *auditLogModel) rebuildTableRows() {
	var rows []table.Row
	lowerFilter := strings.ToLower(m.filter)

	for _, entry := range m.allEntries {
		// If filtering, check if any field contains the filter text.
		if m.filter != "" {
			if !strings.Contains(strings.ToLower(entry.Timestamp), lowerFilter) &&
				!strings.Contains(strings.ToLower(entry.Username), lowerFilter) &&
				!strings.Contains(strings.ToLower(entry.Action), lowerFilter) &&
				!strings.Contains(strings.ToLower(entry.Details), lowerFilter) {
				continue // Skip this row if it doesn't match
			}
		}

		// --- Color Code the Action ---
		actionCell := entry.Action
		switch {
		case strings.HasPrefix(entry.Action, "ADD"),
			strings.HasPrefix(entry.Action, "CREATE"),
			strings.HasPrefix(entry.Action, "TRUST"),
			strings.HasPrefix(entry.Action, "ASSIGN"):
			actionCell = successStyle.Render(entry.Action)
		case strings.HasPrefix(entry.Action, "DELETE_"),
			strings.HasPrefix(entry.Action, "ROTATE_"),
			strings.HasPrefix(entry.Action, "UNASSIGN"):
			actionCell = specialStyle.Render(entry.Action)
		case strings.HasPrefix(entry.Action, "TOGGLE_"),
			strings.HasPrefix(entry.Action, "UPDATE_"):
			actionCell = helpStyle.Render(entry.Action)
		}

		ts := entry.Timestamp
		if len(ts) > 19 {
			ts = ts[:19] // Truncate fractional seconds for cleaner display
		}

		rows = append(rows, table.Row{ts, entry.Username, actionCell, entry.Details})
	}
	m.table.SetRows(rows)

	// Go to the top of the table after filtering
	if m.isFiltering {
		m.table.GotoTop()
	}
}

func (m auditLogModel) Init() tea.Cmd {
	return nil
}

func (m auditLogModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Adjust table height based on window size.
		// header(3) + filter/help(3)
		m.table.SetHeight(msg.Height - 6)
		m.table.SetWidth(msg.Width - 4) // Account for docStyle margins

	case tea.KeyMsg:
		// If filtering, handle input.
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
					m.rebuildTableRows()
				}
			case tea.KeyRunes:
				m.filter += string(msg.Runes)
				m.rebuildTableRows()
			}
			return m, nil
		}

		// Not filtering, handle commands.
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

func (m auditLogModel) View() string {
	var b strings.Builder
	if m.err != nil {
		b.WriteString(fmt.Sprintf("Error loading audit log: %v", m.err))
		return b.String()
	}

	b.WriteString(titleStyle.Render("📜 Audit Log") + "\n")
	b.WriteString(m.table.View())
	b.WriteString(m.footerView())
	return b.String()
}

func (m auditLogModel) footerView() string {
	var filterStatus string
	if m.isFiltering {
		filterStatus = fmt.Sprintf("Filter: %s█", m.filter)
	} else if m.filter != "" {
		filterStatus = fmt.Sprintf("Filter: %s (press 'esc' to clear)", m.filter)
	} else {
		filterStatus = "Press / to filter..."
	}
	return helpStyle.Render(fmt.Sprintf("\n(↑/↓ to scroll, q to quit) %s", filterStatus))
}
