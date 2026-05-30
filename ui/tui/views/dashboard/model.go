// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package dashboard

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/client"
	"github.com/toeirei/keymaster/i18n"
	"github.com/toeirei/keymaster/ui/tui/helpers/tablecontroll"
	"github.com/toeirei/keymaster/ui/tui/util"
	"github.com/toeirei/keymaster/util/slicest"
)

// copied from keymaster core for now
type Data = struct {
	AccountCount       int
	ActiveAccountCount int
	PublicKeyCount     int
	GlobalKeyCount     int
	AlgoCounts         map[string]int
	HostsUpToDate      int
	HostsOutdated      int
	SystemKeySerial    int
	AuditLogs          []AuditLogEntry
}

type AuditLogEntry = client.AuditLog

type recentActivityRow struct {
	Timestamp string
	Action    string
	Details   string
}

type Model struct {
	data   Data
	err    error
	client client.Client
	size   util.Size
}

func New(c client.Client) *Model {
	return &Model{
		client: c,
	}
}

func (m Model) Init() tea.Cmd {
	// TODO setup table model and table controll
	return m.reload()
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	if m.size.UpdateFromMsg(msg) {
		// TODO rerender table (only here)
		return nil
	}

	switch msg := msg.(type) {
	case msgReloadResult:
		m.data = msg.data
		m.err = msg.err
		// TODO rerender table (only here)
	}

	return nil
}

func (m Model) View() string {
	// TODO should not be needed! this being needed is a side effect of another problem.
	width := m.size.Width
	if width <= 0 {
		width = 80
	}

	contentWidth := util.Clamp(36, width-4, 76)
	sectionTitleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)
	bodyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	warnValueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true)

	if m.err != nil {
		errTitle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("1")).
			Bold(true).
			Render(i18n.T("dashboard.error_title"))
		errBody := lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			Width(contentWidth).
			Render(m.err.Error())
		return lipgloss.JoinVertical(lipgloss.Left, errTitle, "", errBody)
	}

	height := m.size.Height
	if height <= 0 {
		height = 24
	}

	recentActivityRows := recentActivityTableRows(m.data.AuditLogs)
	accountsLine := fmt.Sprintf(i18n.T("dashboard.accounts"), m.data.ActiveAccountCount, m.data.AccountCount)
	publicKeysLine := fmt.Sprintf(i18n.T("dashboard.public_keys"), m.data.PublicKeyCount, m.data.GlobalKeyCount)
	currentLine := fmt.Sprintf(i18n.T("dashboard.hosts_current_key"), m.data.HostsUpToDate)
	dirtyLine := fmt.Sprintf(i18n.T("dashboard.hosts_past_keys"), m.data.HostsOutdated)
	accountsRendered, publicKeysRendered := renderAlignedPair(accountsLine, publicKeysLine, valueStyle, valueStyle, false)
	currentRendered, dirtyRendered := renderAlignedPair(currentLine, dirtyLine, valueStyle, warnValueStyle, true)

	lines := []string{
		sectionTitleStyle.Render(i18n.T("dashboard.system_status")),
		"",
		accountsRendered,
		publicKeysRendered,
		"",
		sectionTitleStyle.Render(i18n.T("dashboard.deployment_status")),
		"",
		currentRendered,
		dirtyRendered,
		"",
		sectionTitleStyle.Render(i18n.T("dashboard.security_posture")),
		"",
		bodyStyle.Render(fmt.Sprintf(i18n.T("dashboard.key_type_spread"), formatAlgoSpread(m.data.AlgoCounts, warnValueStyle))),
		"",
		sectionTitleStyle.Render(i18n.T("dashboard.recent_activity")),
		"",
	}

	if len(recentActivityRows) == 0 {
		lines = append(lines, bodyStyle.Italic(true).Render(i18n.T("dashboard.no_recent_activity")))
	} else {
		maxLogRows := util.Clamp(3, height-14, 10)
		if len(recentActivityRows) > maxLogRows {
			recentActivityRows = recentActivityRows[:maxLogRows]
		}

		recentActivityControll := tablecontroll.New(tablecontroll.Columns[recentActivityRow]{
			{Title: func() string { return i18n.T("dashboard.log_col_time") }, View: func(row recentActivityRow) string { return row.Timestamp }, MaxWidth: 0.18},
			{Title: func() string { return i18n.T("dashboard.log_col_action") }, View: func(row recentActivityRow) string { return row.Action }, MaxWidth: 0.24},
			{Title: func() string { return i18n.T("dashboard.log_col_details") }, View: func(row recentActivityRow) string { return row.Details }, MaxWidth: 0.58},
		})

		tableWidth := recentActivityControll.PreferredWidth(recentActivityRows, contentWidth)
		columns, rows := recentActivityControll.RenderBubblesTable(recentActivityRows, tableWidth)

		tableModel := table.New()
		tableModel.SetColumns(columns)
		tableModel.SetRows(rows)
		tableModel.SetCursor(-1)
		tableModel.SetWidth(tableWidth)
		tableModel.SetHeight(len(rows) + 1)

		lines = append(lines, strings.Split(tableModel.View(), "\n")...)
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func formatAlgoSpread(algoCounts map[string]int, style lipgloss.Style) string {
	if len(algoCounts) == 0 {
		return "-"
	}

	algorithms := make([]string, 0, len(algoCounts))
	for algorithm := range algoCounts {
		algorithms = append(algorithms, algorithm)
	}
	slices.Sort(algorithms)

	parts := make([]string, 0, len(algorithms))
	for _, algorithm := range algorithms {
		parts = append(parts, style.Render(fmt.Sprintf("%s: %d", algorithm, algoCounts[algorithm])))
	}
	return strings.Join(parts, ", ")
}

func recentActivityTableRows(logs []AuditLogEntry) []recentActivityRow {
	return slicest.Map(logs, func(al AuditLogEntry) recentActivityRow {
		return recentActivityRow{
			Timestamp: al.Timestamp.Format("Jan 02 15:04"),
			Action:    titleFromUnderscore(strings.TrimSpace(al.Action)),
			Details:   strings.TrimSpace(strings.ReplaceAll(al.Details, "\n", " ")),
		}
	})
}

func renderAlignedPair(line1, line2 string, style1, style2 lipgloss.Style, alignValueRight bool) (string, string) {
	label1, value1 := splitLabelValue(line1)
	label2, value2 := splitLabelValue(line2)
	labelWidth := max(lipgloss.Width(label1), lipgloss.Width(label2))
	labelRenderer := lipgloss.NewStyle().Width(labelWidth)

	valueRenderer := lipgloss.NewStyle()
	if alignValueRight {
		valueWidth := max(lipgloss.Width(value1), lipgloss.Width(value2))
		valueRenderer = valueRenderer.Width(valueWidth).Align(lipgloss.Right)
	}

	render := func(label, value string, style lipgloss.Style) string {
		if value == "" {
			return style.Render(label)
		}
		return style.Render(labelRenderer.Render(label) + " " + valueRenderer.Render(value))
	}

	return render(label1, value1, style1), render(label2, value2, style2)
}

func splitLabelValue(line string) (string, string) {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) < 2 {
		return line, ""
	}
	return parts[0] + ":", strings.TrimSpace(parts[1])
}

// TODO decide if this function handles date, time or datetime
func parseTimestamp(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return i18n.T("dashboard.no_timestamp")
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

// TODO use lipgloss
func titleFromUnderscore(action string) string {
	if action == "" {
		return i18n.T("dashboard.unknown_action")
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
	// TODO add empty skeleton loading state

	return func() tea.Msg {
		ctx := context.Background()

		accounts, err := m.client.ListAccounts(ctx)
		if err != nil {
			return msgReloadResult{err: err}
		}

		publicKeys, err := m.client.ListPublicKeys(ctx, "")
		if err != nil {
			return msgReloadResult{err: err}
		}

		dirtyAccounts, err := m.client.ListAccountsDirty(ctx)
		if err != nil {
			return msgReloadResult{err: err}
		}

		auditLogs, err := m.client.ListAuditLogs(ctx, 25)
		if err != nil {
			return msgReloadResult{err: err}
		}

		return msgReloadResult{data: Data{
			AccountCount:       len(accounts),
			ActiveAccountCount: len(accounts), // TODO client API currently has no account activation state
			PublicKeyCount:     len(publicKeys),
			GlobalKeyCount:     0, // TODO client API currently has no global-key flag on PublicKey
			AlgoCounts: slicest.ToMap(
				slicest.Reduce(
					slicest.Map(
						publicKeys,
						func(publicKey client.PublicKey) string { return publicKey.Algorithm },
					),
					func(algo string, algos []string) []string {
						if !slices.Contains(algos, algo) {
							algos = append(algos, algo)
						}
						return algos
					},
				),
				func(algo string) (string, int) {
					return algo, len(slicest.Filter(publicKeys, func(publicKey client.PublicKey) bool { return publicKey.Algorithm == algo }))
				},
			),
			HostsUpToDate:   len(accounts) - len(dirtyAccounts),
			HostsOutdated:   len(dirtyAccounts),
			SystemKeySerial: 0,
			AuditLogs:       auditLogs,
		}}
	}
}
