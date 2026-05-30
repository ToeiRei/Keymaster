// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package crud

import (
	"context"

	"github.com/bobg/go-generics/v4/slices"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	windowtitle "github.com/toeirei/keymaster/ui/tui/helpers/title"
	"github.com/toeirei/keymaster/ui/tui/popups/choicepopup"
	"github.com/toeirei/keymaster/ui/tui/popups/messagepopup"
	"github.com/toeirei/keymaster/ui/tui/popups/progresspopup"
	"github.com/toeirei/keymaster/ui/tui/util"
	"github.com/toeirei/keymaster/ui/tui/util/keys"
)

type ListModel[
	TRecord any,
	TRecordCreate comparable,
	TRecordUpdate comparable,
	TRecordId comparable,
	TFilter comparable,
] struct {
	// configuration
	crud *Crud[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]

	// state
	records  []TRecord
	focussed bool

	// util
	size util.Size

	// sub models
	table *table.Model
}

func NewList[
	TRecord any,
	TRecordCreate comparable,
	TRecordUpdate comparable,
	TRecordId comparable,
	TFilter comparable,
](crud *Crud[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]) *ListModel[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter] {
	return &ListModel[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]{
		crud:  crud,
		table: util.NewPointer(table.New()),
	}
}

// Init implements util.Model.
func (m *ListModel[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]) Init() tea.Cmd {
	m.refreshTable()
	return m.reload()
}

// Update implements util.Model.
func (m *ListModel[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]) Update(msg tea.Msg) tea.Cmd {
	// Handle resizing
	if m.size.UpdateFromMsg(msg) {
		m.table.SetWidth(m.size.Width)
		m.table.SetHeight(m.size.Height)
		m.refreshTable()
		return nil
	}

	// Intercept messages
	selectedRecord := m.selectedRecord()
	if cmd, done := Intercept(
		msg,
		ListMsgInterceptorCtx[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]{m.crud, selectedRecord},
		m.crud.listMsgInterceptors...,
	); cmd != nil || done {
		return cmd
	}

	// Handle messages
	switch msg := msg.(type) {
	case listMsgReloaded[TRecord]:
		m.records = msg.records
		m.refreshTable()
		if msg.err != nil {
			return choicepopup.Open("Error loading "+m.crud.Texts.EntityNameMultiple()+":\n"+msg.err.Error(), choicepopup.Choices{
				choicepopup.Choice{Name: "Close", Cmd: m.crud.routerControll.Pop(1), KeyBindings: keys.KeyBindingList{keys.Close()}},
				choicepopup.Choice{Name: "Reload", Cmd: m.reload(), KeyBindings: nil},
			})
		}
		return nil

	case CreateMsgCreated[TRecord]:
		// full reload
		if m.crud.listReloadAfterChange {
			return m.reload()
		}
		// partial update
		m.records = append(m.records, msg.Record)
		m.refreshTable()
		return nil

	case UpdateMsgUpdated[TRecord]:
		// full reload
		if m.crud.listReloadAfterChange {
			return m.reload()
		}
		// partial update
		i := slices.IndexFunc(m.records, func(record TRecord) bool { return m.crud.getRecordId(record) == m.crud.getRecordId(msg.Record) })
		m.records[i] = msg.Record
		m.refreshTable()
		return nil

	case listMsgDeleteResult[TRecord]:
		if msg.err != nil {
			return messagepopup.Open(messagepopup.Error, "Error deleting "+m.crud.Texts.EntityNameSingular()+":\n"+msg.err.Error(), nil)
		}
		// full reload
		if m.crud.listReloadAfterChange {
			return m.reload()
		}
		// partial update
		m.records = slices.DeleteFunc(m.records, func(record TRecord) bool { return m.crud.getRecordId(record) == m.crud.getRecordId(msg.record) })
		m.refreshTable()
		return nil

	case tea.KeyMsg:
		if !m.focussed {
			return nil
		}
		switch {
		case key.Matches(msg, ListBaseKeyMap.Create):
			return m.crud.routerControll.Push(util.ModelPointer(NewCreate(m.crud, m.crud.createRecordPreset())))

		case key.Matches(msg, ListBaseKeyMap.Edit):
			selectedRecord := m.selectedRecord()
			if selectedRecord == nil {
				return messagepopup.Open(messagepopup.Info, "Please select a "+m.crud.Texts.EntityNameSingular()+" to edit.", nil)
			}
			return m.crud.routerControll.Push(util.ModelPointer(NewUpdate(
				m.crud,
				*selectedRecord,
			)))

		case key.Matches(msg, ListBaseKeyMap.Delete):
			selectedRecord := m.selectedRecord()
			if selectedRecord == nil {
				return messagepopup.Open(messagepopup.Info, "Please select a "+m.crud.Texts.EntityNameSingular()+" to delete.", nil)
			}
			return choicepopup.Open(
				"Do you realy want to delete this "+m.crud.Texts.EntityNameSingular()+"?",
				choicepopup.Choices{
					{Name: "Cancel", Cmd: nil, KeyBindings: keys.KeyBindingList{keys.Cancel()}},
					{Name: "Delete", Cmd: progresspopup.Open(
						progresspopup.Spinner,
						"Deleting "+m.crud.Texts.EntityNameSingular(),
						func(ctx context.Context, _ progresspopup.ProgressChan) tea.Cmd {
							err := m.crud.deleteRecord(ctx, m.crud.getRecordId(*selectedRecord))
							return func() tea.Msg {
								return listMsgDeleteResult[TRecord]{*selectedRecord, err}
							}
						},
						progresspopup.WithCancel(),
					)},
				},
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
func (m *ListModel[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]) View() string {
	return m.table.View()
}

// Focus implements util.Model.
func (m *ListModel[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]) Focus(parentKeyMap help.KeyMap) tea.Cmd {
	if m.crud.ReloadOnNextFocus {
		m.crud.ReloadOnNextFocus = false
		return m.reload()
		// no need to focus or announce anything, as the popup interceptor will take it away again.
	}
	m.focussed = true
	m.table.Focus()
	return tea.Batch(
		windowtitle.Announce(m.crud.Texts.EntityNameMultiple()),
		util.AnnounceKeyMapCmd(parentKeyMap, ListBaseKeyMap, m.crud.listGlobalKeyMap),
	)
}

// Blur implements util.Model.
func (m *ListModel[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]) Blur() {
	m.focussed = false
	m.table.Blur()
}

// *[ListModel] implements [util.Model]
var _ util.Model = (*ListModel[any, any, any, any, any])(nil)

func (m *ListModel[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]) reload() tea.Cmd {
	return progresspopup.Open(
		progresspopup.Spinner,
		"Loading "+m.crud.Texts.EntityNameMultiple(), func(ctx context.Context, pc progresspopup.ProgressChan) tea.Cmd {
			records, err := m.crud.getRecords(ctx, util.NewZero[TFilter]())
			return util.TeaMsgToCmd(listMsgReloaded[TRecord]{records, err})
		},
		progresspopup.WithCancel(),
	)
}

func (m *ListModel[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]) refreshTable() {
	// generate and apply columns and rows
	columns, rows := m.crud.buildListTable(m.records, m.size.Width)
	m.table.SetColumns(columns)
	m.table.SetRows(rows)

	// reposition cursor
	if m.table.Cursor() <= 0 && len(m.records) > 0 {
		m.table.MoveUp(1)
	}
}

func (m *ListModel[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]) selectedRecord() *TRecord {
	if m.table.Cursor() == -1 {
		return nil
	}
	// copy selectedRecord to avoid unwanted changes by weird devs
	selectedRecord := m.records[m.table.Cursor()]
	return &selectedRecord
}
