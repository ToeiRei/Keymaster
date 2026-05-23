// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package dashboard

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/core"
	"github.com/toeirei/keymaster/core/model"
	"github.com/toeirei/keymaster/ui/tui/util"
)

type Data = core.DashboardData

type Model struct {
	data  Data
	err   error
	store interface{}
	size  util.Size
}

func New(storeParam interface{}) *Model {
	return &Model{
		store: storeParam,
	}
}

func (m Model) Init() tea.Cmd {
	return m.reload()
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	if m.size.UpdateFromMsg(msg) {
		return nil
	}

	switch msg := msg.(type) {
	case msgReloadResult:
		m.data = msg.data
		m.err = msg.err
	}

	return nil
}

func (m Model) View() string {
	width := m.size.Width
	if width <= 0 {
		width = 80
	}

	contentWidth := width - 4
	if contentWidth < 36 {
		contentWidth = 36
	}
	if contentWidth > 76 {
		contentWidth = 76
	}

	borderStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("6")).
		Padding(0, 1).
		Width(contentWidth)

	if m.err != nil {
		errTitle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("1")).
			Bold(true).
			Render("Dashboard Error")
		errBody := lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			Width(contentWidth - 2).
			Render(m.err.Error())
		return borderStyle.Render(lipgloss.JoinVertical(lipgloss.Left, errTitle, "", errBody))
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("6")).
		Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	warnValueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true)

	metricsPanel := borderStyle.Render(lipgloss.JoinVertical(
		lipgloss.Left,
		titleStyle.Render("System Metrics"),
		"",
		formatKeyValue("Accounts", fmt.Sprintf("%d total / %d active", m.data.AccountCount, m.data.ActiveAccountCount), labelStyle, valueStyle),
		formatKeyValue("Hosts", formatHostStatus(m.data.HostsUpToDate, m.data.HostsOutdated), labelStyle, chooseOutdatedStyle(m.data.HostsOutdated, valueStyle, warnValueStyle)),
		formatKeyValue("System Key", fmt.Sprintf("serial #%d", m.data.SystemKeySerial), labelStyle, valueStyle),
	))

	logsTitle := titleStyle.Render("Recent Audit Logs")
	logsBody := ""
	if len(m.data.RecentLogs) == 0 {
		logsBody = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Italic(true).Render("No recent entries")
	} else {
		height := m.size.Height
		if height <= 0 {
			height = 24
		}
		maxLogRows := clampInt(height-14, 3, 10)
		logLines := make([]string, 0, minInt(len(m.data.RecentLogs), maxLogRows+1))
		entryWidth := contentWidth - 4
		head := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Bold(true).Render("Time         | Action               | Details")
		logLines = append(logLines, head)
		for i, al := range m.data.RecentLogs {
			if i >= maxLogRows {
				break
			}
			logLines = append(logLines, formatLogEntry(al, entryWidth))
		}
		if len(m.data.RecentLogs) > maxLogRows {
			remaining := len(m.data.RecentLogs) - maxLogRows
			more := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render(fmt.Sprintf("+%d more entries", remaining))
			logLines = append(logLines, more)
		}
		logsBody = lipgloss.JoinVertical(lipgloss.Left, logLines...)
	}

	logsPanel := borderStyle.Render(lipgloss.JoinVertical(
		lipgloss.Left,
		logsTitle,
		"",
		logsBody,
	))

	return lipgloss.JoinVertical(lipgloss.Left, metricsPanel, "", logsPanel)
}

func formatKeyValue(label, value string, labelStyle, valueStyle lipgloss.Style) string {
	return labelStyle.Render(label+":") + " " + valueStyle.Render(value)
}

func formatHostStatus(upToDate, outdated int) string {
	if outdated > 0 {
		return fmt.Sprintf("%d up-to-date, %d outdated", upToDate, outdated)
	}
	return fmt.Sprintf("%d up-to-date, all clean", upToDate)
}

func chooseOutdatedStyle(outdated int, normal, warn lipgloss.Style) lipgloss.Style {
	if outdated > 0 {
		return warn
	}
	return normal
}

func formatLogEntry(al model.AuditLogEntry, width int) string {
	ts := parseTimestamp(al.Timestamp)
	action := titleFromUnderscore(strings.TrimSpace(al.Action))
	details := strings.TrimSpace(strings.ReplaceAll(al.Details, "\n", " "))
	if width < 38 {
		width = 38
	}
	actionWidth := 20
	base := fmt.Sprintf("%s | %s | ", ts, action)
	maxDetails := width - lipgloss.Width(base)
	if maxDetails < 8 {
		maxDetails = 8
	}
	if lipgloss.Width(details) > maxDetails {
		details = truncateRight(details, maxDetails)
	}

	timestampStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	actionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)
	detailStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("7"))

	actionPadded := padRight(action, actionWidth)
	return timestampStyle.Render(padRight(ts, 12)) + " | " + actionStyle.Render(actionPadded) + " | " + detailStyle.Render(details)
}

func parseTimestamp(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "--:--"
	}
	layouts := []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02T15:04:05Z07:00"}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.Format("Jan 02 15:04")
		}
	}
	if len(raw) > 12 {
		return raw[:12]
	}
	return raw
}

func truncateRight(s string, max int) string {
	if max <= 1 || lipgloss.Width(s) <= max {
		return s
	}
	r := []rune(s)
	if len(r) >= max {
		return string(r[:max-3]) + "..."
	}
	return s
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func titleFromUnderscore(action string) string {
	if action == "" {
		return "Unknown"
	}
	parts := strings.Split(strings.ToLower(strings.ReplaceAll(action, "_", " ")), " ")
	for i, p := range parts {
		if p == "" {
			continue
		}
		r := []rune(p)
		r[0] = []rune(strings.ToUpper(string(r[0])))[0]
		parts[i] = string(r)
	}
	return strings.Join(parts, " ")
}

func padRight(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

func (m *Model) Focus(parentKeyMap help.KeyMap) tea.Cmd {
	return tea.Batch(
		m.reload(),
		util.AnnounceKeyMapCmd(parentKeyMap),
	)
}

func (m *Model) Blur() {}

// *[Model] implements [util.Model]
var _ util.Model = (*Model)(nil)

func (m *Model) reload() tea.Cmd {
	return func() tea.Msg {
		// Type assert store to core.DashboardReader and call core.BuildDashboardData
		reader, ok := m.store.(core.DashboardReader)
		if !ok {
			return msgReloadResult{
				data: Data{},
				err:  fmt.Errorf("store does not implement DashboardReader"),
			}
		}

		data, err := core.BuildDashboardData(reader)
		return msgReloadResult{
			data: data,
			err:  err,
		}
	}
}
