// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package crud

import (
	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/form"
	formelement "github.com/toeirei/keymaster/ui/tui/models/helpers/form/element"
	popupviews "github.com/toeirei/keymaster/ui/tui/models/views/popup"
	"github.com/toeirei/keymaster/ui/tui/util"
	"github.com/toeirei/keymaster/ui/tui/util/keys"
)

type EditModel[
	TRecord any,
	TRecordCreate comparable,
	TRecordEdit comparable,
	TId comparable,
	TFilter comparable,
] struct {
	// configuration
	crud *Crud[TRecord, TRecordCreate, TRecordEdit, TId, TFilter]

	// state
	record   TRecord
	locked   *string
	focussed bool

	// util
	size util.Size

	// sub models
	form *form.Form[TRecordEdit]
}

func NewEdit[
	TRecord any,
	TRecordCreate comparable,
	TRecordEdit comparable,
	TId comparable,
	TFilter comparable,
](crud *Crud[TRecord, TRecordCreate, TRecordEdit, TId, TFilter], record TRecord) *EditModel[TRecord, TRecordCreate, TRecordEdit, TId, TFilter] {
	return &EditModel[TRecord, TRecordCreate, TRecordEdit, TId, TFilter]{
		crud:   crud,
		record: record,
	}
}

func (m *EditModel[TRecord, TRecordCreate, TRecordEdit, TId, TFilter]) Init() tea.Cmd {
	formOpts := m.crud.editFormRows()
	formOpts = append(formOpts,
		form.WithRow(
			form.WithItem[TRecordEdit]("_reset", formelement.NewButton("Reset",
				formelement.WithButtonActionReset(),
			)),
			form.WithItem[TRecordEdit]("_cancel", formelement.NewButton("Cancel",
				formelement.WithButtonActionCancel(),
				formelement.WithButtonGlobalKeyBindings(keys.Cancel()),
			)),
			form.WithItem[TRecordEdit]("_save", formelement.NewButton("Save",
				formelement.WithButtonActionSubmit(),
				formelement.WithButtonGlobalKeyBindings(keys.Save()),
			)),
		),
		// events
		form.WithOnSubmit(func(result TRecordEdit, err error) tea.Cmd {
			m.locked = util.NewPointer("Updating Record...")
			return func() tea.Msg {
				record, err := m.crud.editRecord(m.crud.getRecordId(m.record), result)
				return editMsgUpdateResult[TRecord]{record, err}
			}
		}),
		form.WithOnCancel[TRecordEdit](func() tea.Cmd {
			return m.crud.routerControll.Pop(1)
		}),
		form.WithOnReset[TRecordEdit](func() tea.Cmd {
			_ = m.refreshForm()
			return nil
		}),
		form.WithOnDiscardGuard[TRecordEdit](discardGuard),
		// data
		form.WithInitialData(m.crud.makeRecordEdit(m.record)),
	)

	m.form = util.NewPointer(form.New(formOpts...))

	return m.form.Init()
}

func (m *EditModel[TRecord, TRecordCreate, TRecordEdit, TId, TFilter]) Update(msg tea.Msg) tea.Cmd {
	// Handle resizing
	if m.size.UpdateFromMsg(msg) {
		return m.form.Update(msg)
	}

	// Intercept messages
	if cmd, done := Intercept(msg, m.form, m.crud.editMsgInterceptors...); cmd != nil || done {
		return cmd
	}

	// Handle messages
	switch msg := msg.(type) {
	case editMsgUpdateResult[TRecord]:
		m.locked = nil
		if msg.err != nil {
			if msg.err != nil {
				return popupviews.OpenMessage(popupviews.MessageError, "Error updating Record:\n"+msg.err.Error(), nil)
			}
			return nil
		}
		return tea.Sequence(m.crud.routerControll.Pop(1), func() tea.Msg { return editMsgUpdated[TRecord]{msg.record} })
	}

	if !m.focussed || m.locked != nil {
		return nil
	}

	// pass key msg to form
	return m.form.Update(msg)
}

func (m *EditModel[TRecord, TRecordCreate, TRecordEdit, TId, TFilter]) View() string {
	if m.locked != nil {
		return *m.locked
	}
	return m.form.View()
}

func (m *EditModel[TRecord, TRecordCreate, TRecordEdit, TId, TFilter]) Focus(parentKeyMap help.KeyMap) tea.Cmd {
	m.focussed = true
	return m.form.Focus(parentKeyMap)
}

func (m *EditModel[TRecord, TRecordCreate, TRecordEdit, TId, TFilter]) Blur() {
	m.focussed = false
	m.form.Blur()
}

// *[EditModel] implements [util.Model]
var _ util.Model = (*EditModel[any, any, any, any, any])(nil)

func (m *EditModel[TRecord, TRecordCreate, TRecordEdit, TId, TFilter]) refreshForm() error {
	data := m.crud.makeRecordEdit(m.record)
	m.form.InitialData = data
	return m.form.Set(data)
}
