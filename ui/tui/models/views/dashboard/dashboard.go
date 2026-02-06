// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package dashboard

import (
	"fmt"

	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/core"
	"github.com/toeirei/keymaster/core/model"
	"github.com/toeirei/keymaster/ui/tui/util"
	"github.com/toeirei/keymaster/util/slicest"
)

type Model struct {
	data core.DashboardData
	err  error
	size util.Size
}

func New() *Model {
	return &Model{}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	if m.size.Update(msg) {
		return nil
	}
	// wont work until db can be constructed... mock for now
	m.data = core.DashboardData{
		AccountCount:       69,
		ActiveAccountCount: 420,
		HostsUpToDate:      67,
		HostsOutdated:      0,
		SystemKeySerial:    9001,
		RecentLogs: []model.AuditLogEntry{
			{
				ID:        1,
				Timestamp: "2026.01.01",
				Username:  "rei",
				Action:    "eat",
				Details:   "nothing",
			},
		},
	}
	m.err = nil
	return nil
}

func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf(
			"Error Loading Dashboard data:\n%s",
			lipgloss.NewStyle().Italic(true).Render(m.err.Error()),
		)
	} else {
		return fmt.Sprintf(
			"AccountCount: %v\nActiveAccountCount: %v\nHostsUpToDate: %v\nHostsOutdated: %v\nSystemKeySerial: %v\nRecentLogs:\n%s\n",
			m.data.AccountCount,
			m.data.ActiveAccountCount,
			m.data.HostsUpToDate,
			m.data.HostsOutdated,
			m.data.SystemKeySerial,
			lipgloss.JoinVertical(
				lipgloss.Left,
				slicest.Map(
					m.data.RecentLogs,
					func(al model.AuditLogEntry) string { return fmt.Sprintf("- %+#v", al) },
				)...,
			),
		)
	}
}

// no focus needed, as it will only be used as a basic background for the apps content
func (m *Model) Focus(baseKeyMap help.KeyMap) tea.Cmd { return nil }
func (m *Model) Blur()                                {}

// *Model implements util.Model
var _ util.Model = (*Model)(nil)
