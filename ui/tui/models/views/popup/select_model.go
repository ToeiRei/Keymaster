// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package popupviews

import (
	"context"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/form"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/popup"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/tablecontroll"
	"github.com/toeirei/keymaster/ui/tui/util"
	"github.com/toeirei/keymaster/ui/tui/util/keys"
)

type (
	SelectFnLoadRecords[T any]    = func(ctx context.Context) ([]T, error)
	SelectFnOnRecordSelect[T any] = func(record T) tea.Cmd
	SelectFnBuildTable[T any]     = func(records []T, width int) ([]table.Column, []table.Row)
	SelectFnFilterRecords[T any]  = func(filter string, records []T) []T
)

type SelectModel[T any] struct {
	title string

	fnLoadRecords    SelectFnLoadRecords[T]
	fnOnRecordSelect SelectFnOnRecordSelect[T]
	tableControll    tablecontroll.Controll[T]
	fnFilterRecords  SelectFnFilterRecords[T] // optional

	records  []T
	focussed bool
	size     util.Size

	textModel  *textinput.Model
	tableModel *table.Model
}

type SelectOption[T any] func(*SelectModel[T])

func WithSelectFilter[T any](fn SelectFnFilterRecords[T]) SelectOption[T] {
	return func(m *SelectModel[T]) {
		m.fnFilterRecords = fn
	}
}

func OpenSelect[T any](
	title string,
	fnLoadRecords SelectFnLoadRecords[T],
	fnOnRecordSelect SelectFnOnRecordSelect[T],
	tableControll tablecontroll.Controll[T],
	opts ...SelectOption[T],
) tea.Cmd {
	return popup.Open(util.ModelPointer(newSelect(
		title,
		fnLoadRecords,
		fnOnRecordSelect,
		tableControll,
		opts...,
	)))
}

func newSelect[T any](
	title string,
	fnLoadRecords SelectFnLoadRecords[T],
	fnOnRecordSelect SelectFnOnRecordSelect[T],
	tableControll tablecontroll.Controll[T],
	opts ...SelectOption[T],
) *SelectModel[T] {
	model := &SelectModel[T]{
		title:            title,
		fnLoadRecords:    fnLoadRecords,
		fnOnRecordSelect: fnOnRecordSelect,
		tableControll:    tableControll,
		textModel:        util.NewPointer(textinput.New()),
		tableModel:       util.NewPointer(table.New()),
	}

	for _, opt := range opts {
		opt(model)
	}

	return model
}

func (m SelectModel[T]) Init() tea.Cmd { return m.reload() }

func (m *SelectModel[T]) Update(msg tea.Msg) tea.Cmd {
	if m.size.UpdateFromMsg(msg) {
		m.refreshTable()
		return nil
	}

	switch msg := msg.(type) {
	case selectMsgReloaded[T]:
		m.records = msg.records
		m.refreshTable()
		if msg.err != nil {
			return OpenChoice("Error loading records:\n"+msg.err.Error(), Choices{
				Choice{"Close", popup.Close(), form.GlobalKeyMap{keys.Close()}},
				Choice{"Reload", m.reload(), nil},
			})
		}
		return nil
	case tea.KeyMsg:
		if !m.focussed {
			return nil
		}
		switch {
		case key.Matches(msg, SelectBaseKeyMap.Cancel):
			return popup.Close()
		case key.Matches(msg, SelectBaseKeyMap.Select):
			if m.tableModel.Cursor() == -1 {
				return OpenMessage(MessageError, "Please select a record.", nil)
			}
			return tea.Sequence(
				popup.Close(),
				m.fnOnRecordSelect(m.records[m.tableModel.Cursor()]),
			)
		case key.Matches(
			msg,
			SelectBaseKeyMap.LineUp,
			SelectBaseKeyMap.LineDown,
			SelectBaseKeyMap.PageUp,
			SelectBaseKeyMap.PageDown,
			SelectBaseKeyMap.HalfPageUp,
			SelectBaseKeyMap.HalfPageDown,
			SelectBaseKeyMap.GotoTop,
			SelectBaseKeyMap.GotoBottom,
		):
			return util.UpdateTeaModelInplace(msg, m.tableModel)
		default:
			if m.fnFilterRecords != nil {
				return util.UpdateTeaModelInplace(msg, m.textModel)
			}
			return nil
		}
	default:
		if m.fnFilterRecords != nil {
			return util.UpdateTeaModelInplace(msg, m.textModel)
		}
		return nil
	}
}

func (m SelectModel[T]) View() string {
	blocks := make([]string, 0, 3)

	if m.title != "" {
		blocks = append(blocks, m.title)
	}

	if m.fnFilterRecords != nil {
		blocks = append(blocks, m.textModel.View())
	}

	blocks = append(blocks, m.tableModel.View())

	return lipgloss.JoinVertical(lipgloss.Center, blocks...)
}

func (m *SelectModel[T]) Focus(parentKeyMap help.KeyMap) tea.Cmd {
	m.focussed = true
	m.tableModel.Focus()
	return tea.Batch(
		m.textModel.Focus(),
		util.AnnounceKeyMapCmd(parentKeyMap, SelectBaseKeyMap),
	)
}

func (m *SelectModel[T]) Blur() {
	m.focussed = false
	m.textModel.Blur()
	m.tableModel.Blur()
}

// *[SelectModel] implements [util.Model]
var _ util.Model = (*SelectModel[any])(nil)

func (m *SelectModel[T]) reload() tea.Cmd {
	return OpenProgress(ProgressSpinner, "Loading records", func(ctx context.Context, pc ProgressChan) tea.Cmd {
		records, err := m.fnLoadRecords(ctx)
		return util.TeaMsgToCmd(selectMsgReloaded[T]{records, err})
	}, WithProgressCancel())
}

func (m *SelectModel[T]) refreshTable() {
	// height
	availableHeight := m.size.Height
	if m.title != "" {
		availableHeight--
	}
	if m.fnFilterRecords != nil {
		availableHeight--
	}
	if availableHeight <= len(m.records)+1 {
		m.tableModel.SetHeight(availableHeight)
	} else {
		m.tableModel.SetHeight(len(m.records) + 1)
	}

	// width
	tableWidth := m.tableControll.PreferredWidth(m.records, m.size.Width)
	m.tableModel.SetWidth(tableWidth)
	m.textModel.Width = m.size.Width - 1

	// render and apply columns and rows
	columns, rows := m.tableControll.RenderBubblesTable(m.records, tableWidth)
	m.tableModel.SetColumns(columns)
	m.tableModel.SetRows(rows)

	// reposition cursor
	if m.tableModel.Cursor() <= 0 && len(m.records) > 0 {
		m.tableModel.MoveUp(1)
	}
}
