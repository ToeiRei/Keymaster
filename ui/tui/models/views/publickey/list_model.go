// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package publickey

import (
	"context"
	"strings"

	"github.com/bobg/go-generics/v4/slices"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/client"
	"github.com/toeirei/keymaster/ui/tui/models/components/router"
	popupviews "github.com/toeirei/keymaster/ui/tui/models/views/popup"
	"github.com/toeirei/keymaster/ui/tui/util"
)

type ListModel struct {
	// state
	publicKeys   []client.PublicKey
	locked       *string
	loadingError error
	focussed     bool

	// util
	client client.Client
	rc     router.Controll
	size   util.Size

	// sub models
	table *table.Model
}

func NewList(client client.Client, rc router.Controll) *ListModel {
	return &ListModel{
		client: client,
		rc:     rc,
		table:  util.NewPointer(table.New()),
	}
}

// Init implements util.Model.
func (m *ListModel) Init() tea.Cmd {
	m.refreshTable()
	return m.reload()
}

// Update implements util.Model.
func (m *ListModel) Update(msg tea.Msg) tea.Cmd {
	// Handle resizing
	if m.size.UpdateFromMsg(msg) {
		m.table.SetWidth(m.size.Width)
		m.table.SetHeight(m.size.Height)
		m.refreshTable()
		return nil
	}

	// Handle messages
	switch msg := msg.(type) {
	case listMsgReloaded:
		m.locked = nil
		m.publicKeys = msg.publicKeys
		m.loadingError = msg.err
		m.refreshTable()
		return nil

	case listMsgDeleting:
		m.locked = util.NewPointer("Deleting Public Key...")

	case listMsgDeleteResult:
		m.locked = nil
		if msg.err != nil {
			// TODO show popup with error
			return nil
		}
		m.publicKeys = slices.DeleteFunc(m.publicKeys, func(pk client.PublicKey) bool { return pk.Id == msg.publicKey.Id })
		m.refreshTable()
		// TODO does not work for some reason
		return nil

	case EditMsgUpdated, CreateMsgCreated:
		// TODO optimize by only fetching the updated item inplace
		return m.reload()

	case tea.KeyMsg:
		if !m.focussed || m.locked != nil {
			return nil
		}
		switch {
		case key.Matches(msg, ListBaseKeyMap.Create):
			return m.rc.Push(util.ModelPointer(NewCreate(m.client, m.rc, nil)))

		case key.Matches(msg, ListBaseKeyMap.Edit):
			if m.table.Cursor() == -1 {
				return nil // TODO open popup with "please select a public key" text
			}
			return m.rc.Push(util.ModelPointer(NewEdit(
				m.client,
				m.rc,
				m.publicKeys[m.table.Cursor()].Id,
			)))

		case key.Matches(msg, ListBaseKeyMap.Duplicate):
			if m.table.Cursor() == -1 {
				return nil // TODO open popup with "please select a public key" text
			}
			publicKey := m.publicKeys[m.table.Cursor()]
			return m.rc.Push(util.ModelPointer(NewCreate(m.client, m.rc, &createFormData{publicKey.Data, publicKey.Algorithm, publicKey.Comment, tagsStringify(publicKey.Tags)})))

		case key.Matches(msg, ListBaseKeyMap.Delete):
			publicKey := m.publicKeys[m.table.Cursor()]
			return popupviews.OpenChoice(
				"Do you realy want to delete this PublicKey?",
				popupviews.Choices{
					{"Cancel", nil},
					{"Delete", tea.Sequence(
						func() tea.Msg { return listMsgDeleting{} },
						func() tea.Msg {
							return listMsgDeleteResult{
								publicKey: publicKey,
								err:       m.client.DeletePublicKeys(context.Background(), publicKey.Id),
							}
						},
					)},
				},
				40, 40,
			)

		case key.Matches(msg, ListBaseKeyMap.Exit):
			return m.rc.Pop(1)

		case key.Matches(
			msg,
			ListBaseKeyMap.LineUp,
			ListBaseKeyMap.LineDown,
			ListBaseKeyMap.PageUp,
			ListBaseKeyMap.PageDown,
			ListBaseKeyMap.HalfPageUp,
			ListBaseKeyMap.HalfPageDown,
			ListBaseKeyMap.GotoTop,
			ListBaseKeyMap.GotoBottom,
		):
			// pass key msg to table
			return util.UpdateTeaModelInplace(msg, m.table)
		}
	}

	return nil
}

// View implements util.Model.
func (m *ListModel) View() string {
	if m.locked != nil {
		return *m.locked
	}
	if m.loadingError != nil {
		return m.loadingError.Error()
	}
	return m.table.View()
}

// Focus implements util.Model.
func (m *ListModel) Focus(parentKeyMap help.KeyMap) tea.Cmd {
	m.focussed = true
	m.table.Focus()
	return util.AnnounceKeyMapCmd(parentKeyMap, ListBaseKeyMap)
}

// Blur implements util.Model.
func (m *ListModel) Blur() {
	m.focussed = false
	m.table.Blur()
}

// *[ListModel] implements [util.Model]
var _ util.Model = (*ListModel)(nil)

func (m *ListModel) reload() tea.Cmd {
	if m.locked != nil {
		return nil
	}

	m.locked = util.NewPointer("Loading Public Keys...")

	return func() tea.Msg {
		publicKeys, err := m.client.ListPublicKeys(context.Background(), "")
		return listMsgReloaded{publicKeys, err}
	}
}

func (m *ListModel) refreshTable() {
	// TODO this code is just a prove of concept and needs improvements like dynamic scaling!
	m.table.SetColumns([]table.Column{
		{Title: "Algorithm", Width: 10},
		{Title: "Comment", Width: 10},
		{Title: "Tags", Width: m.size.Width - 20 - 6},
	})

	m.table.SetRows(slices.Map(m.publicKeys, func(publicKey client.PublicKey) table.Row {
		return table.Row{
			// column: Algorithm
			publicKey.Algorithm,
			// column: Comment
			publicKey.Comment,
			// column: Tags
			strings.Join(publicKey.Tags, ", "),
		}
	}))
}
