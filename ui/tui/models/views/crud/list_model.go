// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package crud

import (
	"github.com/bobg/go-generics/v4/slices"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/form"
	popupviews "github.com/toeirei/keymaster/ui/tui/models/views/popup"
	"github.com/toeirei/keymaster/ui/tui/util"
	"github.com/toeirei/keymaster/ui/tui/util/keys"
)

type ListModel[
	TRecord any,
	TRecordCreate comparable,
	TRecordEdit comparable,
	TId comparable,
	TFilter comparable,
] struct {
	// configuration
	crud *Crud[TRecord, TRecordCreate, TRecordEdit, TId, TFilter]

	// state
	records      []TRecord
	locked       *string
	loadingError error
	focussed     bool

	// util
	size util.Size

	// sub models
	table *table.Model
}

func NewList[
	TRecord any,
	TRecordCreate comparable,
	TRecordEdit comparable,
	TId comparable,
	TFilter comparable,
](crud *Crud[TRecord, TRecordCreate, TRecordEdit, TId, TFilter]) *ListModel[TRecord, TRecordCreate, TRecordEdit, TId, TFilter] {
	return &ListModel[TRecord, TRecordCreate, TRecordEdit, TId, TFilter]{
		crud:  crud,
		table: util.NewPointer(table.New()),
	}
}

// Init implements util.Model.
func (m *ListModel[TRecord, TRecordCreate, TRecordEdit, TId, TFilter]) Init() tea.Cmd {
	m.refreshTable()
	return m.reload()
}

// Update implements util.Model.
func (m *ListModel[TRecord, TRecordCreate, TRecordEdit, TId, TFilter]) Update(msg tea.Msg) tea.Cmd {
	// Handle resizing
	if m.size.UpdateFromMsg(msg) {
		m.table.SetWidth(m.size.Width)
		m.table.SetHeight(m.size.Height)
		m.refreshTable()
		return nil
	}

	// Intercept messages
	if cmd, done := Intercept(msg, struct{}{}, m.crud.listMsgInterceptors...); cmd != nil || done {
		return cmd
	}

	// Handle messages
	switch msg := msg.(type) {
	case listMsgReloaded[TRecord]:
		m.locked = nil
		m.records = msg.records
		m.loadingError = msg.err
		m.refreshTable()
		if m.table.Cursor() <= 0 && len(m.records) > 0 {
			m.table.MoveUp(1)
		}
		return nil

	case listMsgDeleting:
		m.locked = util.NewPointer("Deleting Record...")

	case listMsgDeleteResult[TRecord]:
		m.locked = nil
		if msg.err != nil {
			return popupviews.OpenMessage(popupviews.MessageError, "Error deleting Record:\n"+msg.err.Error(), nil)
		}
		m.records = slices.DeleteFunc(m.records, func(record TRecord) bool { return m.crud.getRecordId(record) == m.crud.getRecordId(msg.record) })
		m.refreshTable()
		return nil

	case editMsgUpdated[TId], createMsgCreated[TRecord]:
		// TODO optimize by only fetching the updated item inplace
		return m.reload()

	case tea.KeyMsg:
		if !m.focussed || m.locked != nil {
			return nil
		}
		switch {
		case key.Matches(msg, ListBaseKeyMap.Create):
			return m.crud.routerControll.Push(util.ModelPointer(NewCreate(m.crud, nil)))

		case key.Matches(msg, ListBaseKeyMap.Edit):
			if m.table.Cursor() == -1 {
				return popupviews.OpenMessage(popupviews.MessageInfo, "Please select a Record to edit.", nil)
			}
			return m.crud.routerControll.Push(util.ModelPointer(NewEdit(
				m.crud,
				m.records[m.table.Cursor()],
			)))

		case key.Matches(msg, ListBaseKeyMap.Delete):
			record := m.records[m.table.Cursor()]
			return popupviews.OpenChoice(
				"Do you realy want to delete this Record?",
				popupviews.Choices{
					{Name: "Cancel", Cmd: nil, KeyBindings: form.GlobalKeyMap{keys.Cancel()}},
					{Name: "Delete", Cmd: tea.Sequence(
						func() tea.Msg { return listMsgDeleting{} },
						func() tea.Msg {
							return listMsgDeleteResult[TRecord]{
								record: record,
								err:    m.crud.deleteRecord(m.crud.getRecordId(record)),
							}
						},
					)},
				},
				40, 40,
			)

		case key.Matches(msg, ListBaseKeyMap.Exit):
			return m.crud.routerControll.Pop(1)

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
func (m *ListModel[TRecord, TRecordCreate, TRecordEdit, TId, TFilter]) View() string {
	if m.locked != nil {
		return *m.locked
	}
	if m.loadingError != nil {
		return m.loadingError.Error()
	}
	return m.table.View()
}

// Focus implements util.Model.
func (m *ListModel[TRecord, TRecordCreate, TRecordEdit, TId, TFilter]) Focus(parentKeyMap help.KeyMap) tea.Cmd {
	m.focussed = true
	m.table.Focus()
	return util.AnnounceKeyMapCmd(parentKeyMap, ListBaseKeyMap)
}

// Blur implements util.Model.
func (m *ListModel[TRecord, TRecordCreate, TRecordEdit, TId, TFilter]) Blur() {
	m.focussed = false
	m.table.Blur()
}

// *[ListModel] implements [util.Model]
var _ util.Model = (*ListModel[any, any, any, any, any])(nil)

func (m *ListModel[TRecord, TRecordCreate, TRecordEdit, TId, TFilter]) reload() tea.Cmd {
	if m.locked != nil {
		return nil
	}

	m.locked = util.NewPointer("Loading Records...")

	return func() tea.Msg {
		var filter TFilter
		records, err := m.crud.getRecords(filter)
		return listMsgReloaded[TRecord]{records, err}
	}
}

func (m *ListModel[TRecord, TRecordCreate, TRecordEdit, TId, TFilter]) refreshTable() {
	columns, rows := m.crud.makeListTable(m.records, m.size.Width)

	m.table.SetColumns(columns)
	m.table.SetRows(rows)
}
