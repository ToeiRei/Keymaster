// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package crud

import (
	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/form"
	formelement "github.com/toeirei/keymaster/ui/tui/models/helpers/form/element"
	windowtitle "github.com/toeirei/keymaster/ui/tui/models/helpers/title"
	popupviews "github.com/toeirei/keymaster/ui/tui/models/views/popup"
	"github.com/toeirei/keymaster/ui/tui/util"
	"github.com/toeirei/keymaster/ui/tui/util/keys"
)

type UpdateModel[
	TRecord any,
	TRecordCreate comparable,
	TRecordUpdate comparable,
	TRecordId comparable,
	TFilter comparable,
] struct {
	// configuration
	crud *Crud[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]

	// state
	record   TRecord
	focussed bool

	// util
	size util.Size

	// sub models
	form *form.Form[TRecordUpdate]
}

func NewUpdate[
	TRecord any,
	TRecordCreate comparable,
	TRecordUpdate comparable,
	TRecordId comparable,
	TFilter comparable,
](crud *Crud[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter], record TRecord) *UpdateModel[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter] {
	return &UpdateModel[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]{
		crud:   crud,
		record: record,
	}
}

func (m *UpdateModel[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]) Init() tea.Cmd {
	formOpts := append(m.crud.updateFormRows(),
		// buttons
		form.WithRow(
			form.WithItem[TRecordUpdate]("_reset", formelement.NewButton("Reset",
				formelement.WithButtonActionReset(),
			)),
			form.WithItem[TRecordUpdate]("_cancel", formelement.NewButton("Cancel",
				formelement.WithButtonActionCancel(),
				formelement.WithButtonGlobalKeyBindings(keys.Cancel()),
			)),
			form.WithItem[TRecordUpdate]("_save", formelement.NewButton("Save",
				formelement.WithButtonActionSubmit(),
				formelement.WithButtonGlobalKeyBindings(keys.Save()),
			)),
		),
		// events
		form.WithOnSubmit(func(result TRecordUpdate, err error) (tea.Cmd, bool) {
			return popupviews.OpenProgress(
				popupviews.ProgressSpinner,
				"Updating "+m.crud.Texts.EntityNameSingular,
				func(_ popupviews.ProgressChan) tea.Msg {
					record, err := m.crud.updateRecord(m.crud.getRecordId(m.record), result)
					return updateMsgUpdateResult[TRecord]{record, err}
				},
			), true
		}),
		form.WithOnCancel[TRecordUpdate](func() tea.Cmd {
			return m.crud.routerControll.Pop(1)
		}),
		form.WithOnReset[TRecordUpdate](func() tea.Cmd {
			_ = m.refreshForm()
			return nil
		}),
		form.WithOnDiscardGuard[TRecordUpdate](discardGuard),
		// data
		form.WithInitialData(m.crud.recordToRecordUpdate(m.record)),
	)

	m.form = util.NewPointer(form.New(formOpts...))

	return m.form.Init()
}

func (m *UpdateModel[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]) Update(msg tea.Msg) tea.Cmd {
	// Handle resizing
	if m.size.UpdateFromMsg(msg) {
		return m.form.Update(msg)
	}

	// Intercept messages
	if cmd, done := Intercept(
		msg,
		UpdateMsgInterceptorCtx[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]{m.crud, m.form},
		m.crud.updateMsgInterceptors...,
	); cmd != nil || done {
		return cmd
	}

	// Handle messages
	switch msg := msg.(type) {
	case updateMsgUpdateResult[TRecord]:
		if msg.err != nil {
			if msg.err != nil {
				return popupviews.OpenMessage(popupviews.MessageError, "Error updating "+m.crud.Texts.EntityNameSingular+":\n"+msg.err.Error(), nil)
			}
			return nil
		}
		return tea.Sequence(m.crud.routerControll.Pop(1), func() tea.Msg { return UpdateMsgUpdated[TRecord]{msg.record} })
	}

	if !m.focussed {
		return nil
	}

	// pass key msg to form
	return m.form.Update(msg)
}

func (m *UpdateModel[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]) View() string {
	return m.form.View()
}

func (m *UpdateModel[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]) Focus(parentKeyMap help.KeyMap) tea.Cmd {
	m.focussed = true
	return tea.Batch(
		windowtitle.Announce(m.crud.Texts.EntityNameMultiple),
		m.form.Focus(parentKeyMap),
	)
}

func (m *UpdateModel[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]) Blur() {
	m.focussed = false
	m.form.Blur()
}

// *[UpdateModel] implements [util.Model]
var _ util.Model = (*UpdateModel[any, any, any, any, any])(nil)

func (m *UpdateModel[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]) refreshForm() error {
	data := m.crud.recordToRecordUpdate(m.record)
	m.form.InitialData = data
	return m.form.Set(data)
}
