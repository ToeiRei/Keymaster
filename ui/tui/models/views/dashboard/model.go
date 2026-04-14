// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package dashboard

import (
	"context"
	"errors"
	"fmt"

	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/client"
	"github.com/toeirei/keymaster/core"
	"github.com/toeirei/keymaster/core/model"
	"github.com/toeirei/keymaster/ui/tui/util"
	"github.com/toeirei/keymaster/util/slicest"
)

type Data = core.DashboardData

type Model struct {
	data Data
	err  error

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
	// TODO make it look fancy
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
		// TODO implement dashboard data loader
		accounts, err1 := m.client.GetAccounts(context.Background())
		accountsDirty, err2 := m.client.GetDirtyAccounts(context.Background())
		_ = accountsDirty
		// _, _ := m.client.GetAccounts(context.Background())

		return msgReloadResult{
			data: Data{
				AccountCount: len(accounts),
			},
			err: errors.Join(err1, err2),
		}
	}
}
