// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package selectpopup

import (
	"context"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/ui/tui/helpers/popup"
	"github.com/toeirei/keymaster/ui/tui/helpers/tablecontroll"
	"github.com/toeirei/keymaster/ui/tui/popups/choicepopup"
	"github.com/toeirei/keymaster/ui/tui/popups/messagepopup"
	"github.com/toeirei/keymaster/ui/tui/popups/progresspopup"
	"github.com/toeirei/keymaster/ui/tui/util"
	"github.com/toeirei/keymaster/ui/tui/util/keys"
)

type (
	FnLoadRecords[T any]    = func(ctx context.Context) ([]T, error)
	FnOnRecordSelect[T any] = func(record T) tea.Cmd
	FnBuildTable[T any]     = func(records []T, width int) ([]table.Column, []table.Row)
	FnFilterRecords[T any]  = func(filter string, records []T) []T
)

type Model[T any] struct {
	title string

	fnLoadRecords    FnLoadRecords[T]
	fnOnRecordSelect FnOnRecordSelect[T]
	tableControll    tablecontroll.Controll[T]
	fnFilterRecords  FnFilterRecords[T] // optional

	records         []T
	filteredRecords []T
	focussed        bool
	size            util.Size
	titleWidth      int
	prevFilterValue string

	textModel  *textinput.Model
	tableModel *table.Model
}

type Option[T any] func(*Model[T])

func WithFilter[T any](fn FnFilterRecords[T]) Option[T] {
	return func(m *Model[T]) {
		m.fnFilterRecords = fn
	}
}

func Open[T any](
	title string,
	fnLoadRecords FnLoadRecords[T],
	fnOnRecordSelect FnOnRecordSelect[T],
	tableControll tablecontroll.Controll[T],
	opts ...Option[T],
) tea.Cmd {
	return popup.Open(util.ModelPointer(New(
		title,
		fnLoadRecords,
		fnOnRecordSelect,
		tableControll,
		opts...,
	)))
}

func New[T any](
	title string,
	fnLoadRecords FnLoadRecords[T],
	fnOnRecordSelect FnOnRecordSelect[T],
	tableControll tablecontroll.Controll[T],
	opts ...Option[T],
) *Model[T] {
	model := &Model[T]{
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

func (m Model[T]) Init() tea.Cmd { return m.reload() }

func (m *Model[T]) Update(msg tea.Msg) tea.Cmd {
	if m.size.UpdateFromMsg(msg) {
		m.refreshTable()
		return nil
	}

	switch msg := msg.(type) {
	case selectMsgReloaded[T]:
		m.records = msg.records
		m.filterRecords()
		m.refreshTable()
		if msg.err != nil {
			return choicepopup.Open("Error loading records:\n"+msg.err.Error(), choicepopup.Choices{
				choicepopup.Choice{Name: "Close", Cmd: popup.Close(), KeyBindings: keys.KeyBindingList{keys.Close()}},
				choicepopup.Choice{Name: "Reload", Cmd: m.reload(), KeyBindings: nil},
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
				return messagepopup.Open(messagepopup.Error, "Please select a record.", nil)
			}
			return tea.Sequence(
				popup.Close(),
				m.fnOnRecordSelect(m.filteredRecords[m.tableModel.Cursor()]),
			)
		case key.Matches(msg, SelectBaseKeyMap.Up, SelectBaseKeyMap.Down):
			return util.UpdateTeaModelInplace(msg, m.tableModel)
		}
	}

	if m.fnFilterRecords != nil {
		cmd := util.UpdateTeaModelInplace(msg, m.textModel)
		currFilterValue := m.textModel.Value()
		if currFilterValue != m.prevFilterValue {
			m.prevFilterValue = currFilterValue
			m.filterRecords()
			m.refreshTable()
		}
		return cmd
	}

	return nil
}

func (m Model[T]) View() string {
	blocks := make([]string, 0, 3)

	if m.title != "" {
		blocks = append(blocks, lipgloss.PlaceHorizontal(m.titleWidth, lipgloss.Center, m.title))
	}

	if m.fnFilterRecords != nil {
		blocks = append(blocks, m.textModel.View())
	}

	blocks = append(blocks, m.tableModel.View())

	return lipgloss.JoinVertical(lipgloss.Center, blocks...)
}

func (m *Model[T]) Focus(parentKeyMap help.KeyMap) tea.Cmd {
	m.focussed = true
	m.tableModel.Focus()
	return tea.Batch(
		m.textModel.Focus(),
		util.AnnounceKeyMapCmd(parentKeyMap, SelectBaseKeyMap),
	)
}

func (m *Model[T]) Blur() {
	m.focussed = false
	m.textModel.Blur()
	m.tableModel.Blur()
}

// *[Model] implements [util.Model]
var _ util.Model = (*Model[any])(nil)

func (m *Model[T]) reload() tea.Cmd {
	return progresspopup.Open(progresspopup.Spinner, "Loading records", func(ctx context.Context, pc progresspopup.ProgressChan) tea.Cmd {
		records, err := m.fnLoadRecords(ctx)
		return util.TeaMsgToCmd(selectMsgReloaded[T]{records, err})
	}, progresspopup.WithCancel())
}

func (m *Model[T]) filterRecords() {
	if m.prevFilterValue == "" {
		m.filteredRecords = m.records
	} else {
		m.filteredRecords = m.fnFilterRecords(m.prevFilterValue, m.records)
	}
}

func (m *Model[T]) refreshTable() {
	// height
	availableHeight := m.size.Height
	if m.title != "" {
		availableHeight--
	}
	if m.fnFilterRecords != nil {
		availableHeight--
	}
	tableHeight := len(m.filteredRecords) + 1
	m.tableModel.SetHeight(min(availableHeight, tableHeight))

	// width
	tableWidth := m.tableControll.PreferredWidth(m.filteredRecords, m.size.Width)
	m.tableModel.SetWidth(tableWidth)
	m.textModel.Width = tableWidth - 1
	m.titleWidth = tableWidth

	// render and apply columns and rows
	columns, rows := m.tableControll.RenderBubblesTable(m.filteredRecords, tableWidth)
	m.tableModel.SetColumns(columns)
	m.tableModel.SetRows(rows)

	// reposition cursor
	if m.tableModel.Cursor() <= 0 && len(m.filteredRecords) > 0 {
		m.tableModel.MoveUp(1)
	}
}
