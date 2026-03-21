// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package publickey

import (
	"context"
	"fmt"
	"strings"

	"github.com/bobg/go-generics/v4/slices"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/client"
	"github.com/toeirei/keymaster/ui/tui/models/components/router"
	"github.com/toeirei/keymaster/ui/tui/util"
	"github.com/toeirei/keymaster/util/slicest"
)

type ListModel struct {
	// data
	publicKeys   []client.PublicKey
	loading      bool
	loadingError error

	// util
	client client.Client
	rc     router.Controll
	size   util.Size

	// sub models
	table *table.Model
}

func NewList(client client.Client, rc router.Controll) *ListModel {
	table := table.New()
	return &ListModel{
		client:       client,
		rc:           rc,
		publicKeys:   nil,
		loading:      false,
		loadingError: nil,
		table:        &table,
	}
}

// Init implements util.Model.
func (m *ListModel) Init() tea.Cmd {
	m.updateColumns()
	return m.reload()
}

// Update implements util.Model.
func (m *ListModel) Update(msg tea.Msg) tea.Cmd {
	// Handle resizing
	if m.size.Update(msg) {
		m.table.SetWidth(m.size.Width)
		m.table.SetHeight(m.size.Height)
		m.updateColumns()
		return nil
	}

	// Handle messages
	switch msg := msg.(type) {
	case listMsgReload:
		m.loading = false
		m.publicKeys = msg.publicKeys
		m.loadingError = msg.err
		m.table.SetRows(slices.Map(msg.publicKeys, func(publicKey client.PublicKey) table.Row {
			return table.Row{
				fmt.Sprint(publicKey.Id),
				strings.Join(publicKey.Tags, ", "),
			}
		}))
		return nil
	case tea.KeyMsg:
		return util.UpdateTeaModelInplace(msg, m.table)
	}

	return nil
}

// View implements util.Model.
func (m *ListModel) View() string {
	return m.table.View()
}

// Focus implements util.Model.
func (m *ListModel) Focus(baseKeyMap help.KeyMap) tea.Cmd {
	m.table.Focus()
	return util.AnnounceKeyMapCmd(baseKeyMap, m.table.KeyMap)
}

// Blur implements util.Model.
func (m *ListModel) Blur() {
	m.table.Blur()
}

// *Model implements util.Model
var _ util.Model = (*ListModel)(nil)

func (m *ListModel) reload() tea.Cmd {
	if m.loading {
		return nil
	}

	m.loading = true

	return func() tea.Msg {
		publicKeys, err := m.client.ListPublicKeys(context.Background(), "")
		return listMsgReload{publicKeys, err}
	}
}

func (m *ListModel) updateColumns() {
	// i dont know wich one is better >_>
	// Ids := slicest.Map(m.publicKeys, func(publicKey client.PublicKey) int {
	// 	return len(fmt.Sprint(publicKey.Id))
	// })
	// Ids = append(Ids, 2)
	// _ = slices.Max(Ids)
	minIdLength := slicest.ReduceD(m.publicKeys, 2, func(publicKey client.PublicKey, prev int) int {
		return max(len(fmt.Sprint(publicKey.Id)), prev)
	})

	// TODO this code is just a prove of concept and needs improvements!
	m.table.SetColumns([]table.Column{
		{Title: "ID", Width: minIdLength},
		{Title: "Tags", Width: m.size.Width - minIdLength - 2},
	})
}
