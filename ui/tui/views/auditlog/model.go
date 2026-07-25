// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package auditlog

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/client"
	"github.com/toeirei/keymaster/ui/i18n"
	"github.com/toeirei/keymaster/ui/tui/components/router"
	"github.com/toeirei/keymaster/ui/tui/helpers/tablecontroll"
	windowtitle "github.com/toeirei/keymaster/ui/tui/helpers/title"
	"github.com/toeirei/keymaster/ui/tui/util"
)

const (
	// pageSize used before the terminal size is known.
	defaultPageSize = 10
	// smallest page we ever render, so tiny terminals stay usable.
	minPageSize = 3
	// rows reserved from the content height: 1 table header + 1 footer line.
	reservedRows = 2
	// totalUnknown marks that we have not yet discovered the end of the log.
	totalUnknown = -1
)

// auditRow is the flattened, display-ready form of a single audit-log entry.
type auditRow struct {
	Time   string
	User   string
	Action string
	Detail string
}

type Model struct {
	client         client.Client
	routerControll router.Controll

	size     util.Size
	table    *table.Model
	controll tablecontroll.Controll[auditRow]

	// cache maps a global index (0 == newest, matching timestamp DESC) to its
	// entry. Keyed by global index so it survives page-size changes on resize.
	cache map[int]client.AuditLog
	// selectedGlobal is the global index of the highlighted row; page and
	// cursor are derived from it, so all navigation just moves this value.
	selectedGlobal int
	// total is the number of entries once the end is discovered, else totalUnknown.
	total int
	// pageSize is the number of rows per page, derived from the content height.
	pageSize int
	// loadGen is bumped on every dispatched fetch so stale results are dropped.
	loadGen int

	focussed bool
	err      error
}

func New(c client.Client, routerControll router.Controll) *Model {
	return &Model{
		client:         c,
		routerControll: routerControll,
		table:          util.NewPointer(table.New()),
		cache:          map[int]client.AuditLog{},
		total:          totalUnknown,
		pageSize:       defaultPageSize,
		controll: tablecontroll.New(tablecontroll.Columns[auditRow]{
			{Title: i18n.Text("auditlog.col_time"), View: func(r auditRow) string { return r.Time }},
			{Title: i18n.Text("auditlog.col_user"), View: func(r auditRow) string { return r.User }},
			{Title: i18n.Text("auditlog.col_action"), View: func(r auditRow) string { return i18n.TAuditAction(r.Action) }},
			{Title: i18n.Text("auditlog.col_details"), View: func(r auditRow) string { return r.Detail }, EvictionOrder: -1},
		}),
	}
}

// Init implements util.Model.
func (m *Model) Init() tea.Cmd {
	return m.ensureLoaded()
}

// Update implements util.Model.
func (m *Model) Update(msg tea.Msg) tea.Cmd {
	// Handle resizing: recompute page size (keeping the cache) and reconcile.
	if m.size.UpdateFromMsg(msg) {
		m.pageSize = m.computePageSize()
		return m.ensureLoaded()
	}

	switch msg := msg.(type) {
	case msgPageLoaded:
		return m.handleLoaded(msg)

	case tea.KeyMsg:
		if !m.focussed {
			return nil
		}
		switch {
		case key.Matches(msg, BaseKeyMap.Up):
			return m.moveSelection(-1)
		case key.Matches(msg, BaseKeyMap.Down):
			return m.moveSelection(1)
		case key.Matches(msg, BaseKeyMap.PrevPage):
			return m.movePage(-1)
		case key.Matches(msg, BaseKeyMap.NextPage):
			return m.movePage(1)
		case key.Matches(msg, BaseKeyMap.Exit):
			return m.routerControll.Pop(1)
		}
	}

	return nil
}

// View implements util.Model.
func (m *Model) View() string {
	if m.err != nil {
		title := lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true).Render(i18n.T("auditlog.error_title"))
		body := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Width(m.size.Width).Render(m.err.Error())
		return lipgloss.JoinVertical(lipgloss.Left, title, "", body)
	}

	if m.total == 0 {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Italic(true).Render(i18n.T("auditlog.empty"))
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.table.View(),
		lipgloss.PlaceHorizontal(m.size.Width, lipgloss.Center, m.footer()),
	)
}

// Focus implements util.Model.
func (m *Model) Focus(parentKeyMap help.KeyMap) tea.Cmd {
	m.focussed = true
	m.table.Focus()
	return tea.Batch(
		windowtitle.Announce(i18n.T("auditlog.title")),
		util.AnnounceKeyMapCmd(parentKeyMap, BaseKeyMap),
		m.ensureLoaded(),
	)
}

// Blur implements util.Model.
func (m *Model) Blur() {
	m.focussed = false
	m.table.Blur()
}

// *[Model] implements [util.Model]
var _ util.Model = (*Model)(nil)

// moveSelection shifts the selected global index by delta (±1). Because the page
// and cursor are derived from selectedGlobal, crossing a page border here flips
// the page automatically.
func (m *Model) moveSelection(delta int) tea.Cmd {
	next := m.selectedGlobal + delta
	if next < 0 {
		return nil
	}
	if m.total != totalUnknown && next >= m.total {
		return nil
	}
	m.selectedGlobal = next
	return m.ensureLoaded()
}

// movePage jumps a whole page in the given direction, landing on the first row
// of the target page.
func (m *Model) movePage(dir int) tea.Cmd {
	page := m.page() + dir
	if page < 0 {
		return nil
	}
	next := page * m.pageSize
	if m.total != totalUnknown && next >= m.total {
		return nil
	}
	m.selectedGlobal = next
	return m.ensureLoaded()
}

// ensureLoaded clamps the selection, redraws the current page from cache, and
// returns a fetch command when any row in the visible window is missing.
func (m *Model) ensureLoaded() tea.Cmd {
	if m.pageSize <= 0 {
		m.pageSize = m.computePageSize()
	}
	m.clampSelection()

	start, end := m.window()
	m.refreshTable(start, end)

	for i := start; i < end; i++ {
		if _, ok := m.cache[i]; !ok {
			return m.loadWindow(start, end-start)
		}
	}
	return nil
}

func (m *Model) handleLoaded(msg msgPageLoaded) tea.Cmd {
	if msg.gen != m.loadGen {
		return nil // stale result for a window we have navigated away from
	}
	if msg.err != nil {
		m.err = msg.err
		return nil
	}
	m.err = nil

	for i, entry := range msg.entries {
		m.cache[msg.offset+i] = entry
	}

	// A short read reveals the end of the log.
	if len(msg.entries) < msg.limit {
		m.total = msg.offset + len(msg.entries)
		m.clampSelection()
	}

	start, end := m.window()
	m.refreshTable(start, end)
	return nil
}

// loadWindow dispatches an async fetch of [offset, offset+limit) and tags it
// with a fresh generation so earlier in-flight fetches are ignored.
func (m *Model) loadWindow(offset, limit int) tea.Cmd {
	m.loadGen++
	gen := m.loadGen
	client := m.client
	return func() tea.Msg {
		entries, err := client.ListAuditLogs(context.Background(), offset, limit)
		return msgPageLoaded{gen: gen, offset: offset, limit: limit, entries: entries, err: err}
	}
}

// refreshTable rebuilds the bubbles table from whatever is cached for the given
// window; missing rows show a placeholder until their fetch lands.
func (m *Model) refreshTable(start, end int) {
	records := make([]auditRow, 0, end-start)
	for i := start; i < end; i++ {
		if entry, ok := m.cache[i]; ok {
			records = append(records, toRow(entry))
		} else {
			records = append(records, auditRow{Time: "…"})
		}
	}

	columns, rows := m.controll.RenderBubblesTable(records, m.size.Width)
	m.table.SetColumns(columns)
	m.table.SetRows(rows)
	m.table.SetWidth(m.size.Width)
	m.table.SetHeight(len(rows) + 1)
	m.table.SetCursor(m.selectedGlobal - start)
}

func (m *Model) footer() string {
	start, end := m.window()
	page := m.page()
	if m.total == totalUnknown {
		return fmt.Sprintf(i18n.T("auditlog.page_info"), page+1, start+1, end)
	}

	totalPages := (m.total + m.pageSize - 1) / m.pageSize
	info := fmt.Sprintf(i18n.T("auditlog.page_info_total"), page+1, totalPages, start+1, end, m.total)
	if end >= m.total {
		info += " " + i18n.T("auditlog.end_marker")
	}
	return info
}

// window returns the [start, end) global-index range of the current page,
// capped by total once the end is known.
func (m *Model) window() (int, int) {
	start := m.page() * m.pageSize
	end := start + m.pageSize
	if m.total != totalUnknown && end > m.total {
		end = m.total
	}
	if start > end {
		start = end
	}
	return start, end
}

func (m *Model) page() int {
	if m.pageSize <= 0 {
		return 0
	}
	return m.selectedGlobal / m.pageSize
}

func (m *Model) clampSelection() {
	if m.selectedGlobal < 0 {
		m.selectedGlobal = 0
	}
	if m.total != totalUnknown {
		if m.total <= 0 {
			m.selectedGlobal = 0
		} else if m.selectedGlobal >= m.total {
			m.selectedGlobal = m.total - 1
		}
		return
	}
	// While the end is unknown, never let the selection run past the loaded
	// frontier. This keeps rapid Down/Right input from skipping unfetched
	// entries, which would make the short-read end detection overestimate total.
	if frontier := m.contiguousLoaded(); m.selectedGlobal > frontier {
		m.selectedGlobal = frontier
	}
}

// contiguousLoaded returns the count of entries cached contiguously from index 0
// (i.e. the first index that is not yet loaded).
func (m *Model) contiguousLoaded() int {
	n := 0
	for {
		if _, ok := m.cache[n]; !ok {
			return n
		}
		n++
	}
}

func (m *Model) computePageSize() int {
	if m.size.Height <= 0 {
		return defaultPageSize
	}
	return max(m.size.Height-reservedRows, minPageSize)
}

func toRow(entry client.AuditLog) auditRow {
	user := entry.Metadata.Hostuser
	host := entry.Metadata.Hostname
	if user == "" {
		user = "-"
	}
	if host == "" {
		host = "-"
	}

	return auditRow{
		util.StringifyTime(entry.Timestamp),
		user + " @ " + host,
		entry.Action,
		strings.TrimSpace(strings.ReplaceAll(entry.Details.String(), "\n", " ")),
	}
}
