package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/internal/db"
)

type auditLogModel struct {
	viewport viewport.Model
	ready    bool
	err      error
}

func newAuditLogModel() auditLogModel {
	m := auditLogModel{}
	entries, err := db.GetAllAuditLogEntries()
	if err != nil {
		m.err = err
		return m
	}

	var b strings.Builder
	for _, entry := range entries {
		// Format: 2024-09-22 10:30:00 | vbauer | ADD_ACCOUNT | account: root@localhost
		ts := entry.Timestamp
		if len(ts) > 19 {
			ts = ts[:19] // Truncate fractional seconds for cleaner display
		}
		line := fmt.Sprintf("%s | %-15s | %-20s | %s\n", ts, entry.Username, entry.Action, entry.Details)
		b.WriteString(line)
	}

	m.viewport = viewport.New(80, 20) // Placeholder size, will be updated on WindowSizeMsg
	m.viewport.SetContent(b.String())

	return m
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
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			return m, func() tea.Msg { return backToMenuMsg{} }
		}
	case tea.WindowSizeMsg:
		headerHeight := 4 // Title + newlines
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - headerHeight
		if !m.ready {
			m.ready = true
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m auditLogModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error loading audit log: %v", m.err)
	}
	if !m.ready {
		return "\n  Initializing..."
	}

	header := titleStyle.Render("📜 Audit Log")
	return fmt.Sprintf("%s\n%s", header, m.viewport.View())
}
