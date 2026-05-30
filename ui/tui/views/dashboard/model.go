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
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/client"
	"github.com/toeirei/keymaster/core"
	"github.com/toeirei/keymaster/core/model"
	"github.com/toeirei/keymaster/i18n"
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

	// TODO use util.Clamp
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
			Render(i18n.T("dashboard.error_title"))
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
		titleStyle.Render(i18n.T("dashboard.system_metrics")),
		"",
		formatKeyValue(i18n.T("dashboard.label.accounts"), fmt.Sprintf(i18n.T("dashboard.accounts_summary"), m.data.AccountCount, m.data.ActiveAccountCount), labelStyle, valueStyle),
		formatKeyValue(i18n.T("dashboard.label.hosts"), formatHostStatus(m.data.HostsUpToDate, m.data.HostsOutdated), labelStyle, chooseOutdatedStyle(m.data.HostsOutdated, valueStyle, warnValueStyle)),
		formatKeyValue(i18n.T("dashboard.label.system_key"), formatSystemKeySerial(m.data.SystemKeySerial), labelStyle, valueStyle),
	))

	// TODO use bubbles table (like everywhere else)
	logsTitle := titleStyle.Render(i18n.T("dashboard.recent_audit_logs"))
	logsBody := ""
	if len(m.data.RecentLogs) == 0 {
		logsBody = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Italic(true).Render(i18n.T("dashboard.no_recent_entries"))
	} else {
		height := m.size.Height
		if height <= 0 {
			height = 24
		}
		maxLogRows := clampInt(height-14, 3, 10)
		logLines := make([]string, 0, minInt(len(m.data.RecentLogs), maxLogRows+1))
		entryWidth := contentWidth - 4
		logTimeCol := padRight(i18n.T("dashboard.log_col_time"), 12)
		logActionCol := padRight(i18n.T("dashboard.log_col_action"), 20)
		logDetailsCol := i18n.T("dashboard.log_col_details")
		head := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Bold(true).Render(
			fmt.Sprintf("%s | %s | %s", logTimeCol, logActionCol, logDetailsCol),
		)
		logLines = append(logLines, head)
		for i, al := range m.data.RecentLogs {
			if i >= maxLogRows {
				break
			}
			logLines = append(logLines, formatLogEntry(al, entryWidth))
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
		return fmt.Sprintf(i18n.T("dashboard.hosts_status_mixed"), upToDate, outdated)
	}
	return fmt.Sprintf(i18n.T("dashboard.hosts_status_clean"), upToDate)
}

func formatSystemKeySerial(serial int) string {
	if serial <= 0 {
		return i18n.T("dashboard.system_key.not_generated")
	}
	return fmt.Sprintf(i18n.T("dashboard.system_key_serial"), serial)
}

func chooseOutdatedStyle(outdated int, normal, warn lipgloss.Style) lipgloss.Style {
	if outdated > 0 {
		return warn
	}
	return normal
}

func formatLogEntry(al AuditLogEntry, width int) string {
	ts := parseTimestamp(al.Timestamp)
	action := titleFromUnderscore(strings.TrimSpace(al.Action))
	details := strings.TrimSpace(strings.ReplaceAll(al.Details, "\n", " "))
	// TODO use go builtin min()
	if width < 38 {
		width = 38
	}
	actionWidth := 20
	// TODO use bubbles table (like everywhere else)
	base := fmt.Sprintf("%s | %s | ", ts, action)
	maxDetails := width - lipgloss.Width(base)
	// TODO use go builtin min()
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

// TODO use go builtin min()
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TODO keymaster util package
func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
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

// TODO use lipgloss
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
