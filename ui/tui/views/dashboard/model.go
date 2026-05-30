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
	"github.com/toeirei/keymaster/core"
	"github.com/toeirei/keymaster/core/model"
	"github.com/toeirei/keymaster/i18n"
	"github.com/toeirei/keymaster/ui/tui/helpers/tablecontroll"
	"github.com/toeirei/keymaster/ui/tui/util"
	"github.com/toeirei/keymaster/uiadapters"
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
	RecentLogs         []AuditLogEntry
}

// copied from keymaster core for now
type AuditLogEntry = model.AuditLogEntry

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

	recentActivityRows := recentActivityTableRows(m.data.RecentLogs)

	lines := []string{
		sectionTitleStyle.Render(i18n.T("dashboard.system_status")),
		"",
		valueStyle.Render(fmt.Sprintf(i18n.T("dashboard.accounts"), m.data.ActiveAccountCount, m.data.AccountCount)),
		valueStyle.Render(fmt.Sprintf(i18n.T("dashboard.public_keys"), m.data.PublicKeyCount, m.data.GlobalKeyCount)),
		"",
		sectionTitleStyle.Render(i18n.T("dashboard.deployment_status")),
		"",
		valueStyle.Render(fmt.Sprintf(i18n.T("dashboard.hosts_current_key"), m.data.HostsUpToDate)),
		warnValueStyle.Render(fmt.Sprintf(i18n.T("dashboard.hosts_past_keys"), m.data.HostsOutdated)),
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
			Timestamp: parseTimestamp(al.Timestamp),
			Action:    titleFromUnderscore(strings.TrimSpace(al.Action)),
			Details:   strings.TrimSpace(strings.ReplaceAll(al.Details, "\n", " ")),
		}
	})
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
		// Prefer canonical dashboard aggregation from core to avoid placeholder data
		// and keep dashboard behavior aligned across UIs.
		coreData, err := core.BuildDashboardData(uiadapters.NewStoreAdapter())
		if err == nil {
			return msgReloadResult{data: Data{
				AccountCount:       coreData.AccountCount,
				ActiveAccountCount: coreData.ActiveAccountCount,
				HostsUpToDate:      coreData.HostsUpToDate,
				HostsOutdated:      coreData.HostsOutdated,
				SystemKeySerial:    coreData.SystemKeySerial,
				RecentLogs:         coreData.RecentLogs,
			}}
		}

		// Fallback to the client-based approximation for environments that do not
		// have core DB services initialized (for example, isolated TUI test clients).
		accounts, err := m.client.ListAccounts(context.Background())
		if err != nil {
			return msgReloadResult{err: err}
		}

		publicKeys, err := m.client.ListPublicKeys(context.Background(), "")
		if err != nil {
			return msgReloadResult{err: err}
		}

		dirtyAccounts, err := slicest.FilterX(accounts, func(account client.Account) (bool, error) {
			return m.client.IsAccountDirty(context.Background(), account)
		})
		if err != nil {
			return msgReloadResult{err: err}
		}

		return msgReloadResult{data: Data{
			len(accounts),
			len(slicest.Filter(accounts, func(account client.Account) bool { return true /* cant't deactivate accounts */ })),
			len(publicKeys),
			0, // TODO does not exist in client
			slicest.ToMap(
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
			len(accounts) - len(dirtyAccounts),
			len(dirtyAccounts),
			0,   // TODO does not exist in client
			nil, // TODO does not exist in client yet (needs to be added)
		}}
	}
}
