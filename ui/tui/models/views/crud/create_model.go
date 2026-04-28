// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package crud

import (
	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/client"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/form"
	formelement "github.com/toeirei/keymaster/ui/tui/models/helpers/form/element"
	popupviews "github.com/toeirei/keymaster/ui/tui/models/views/popup"
	"github.com/toeirei/keymaster/ui/tui/util"
	"github.com/toeirei/keymaster/ui/tui/util/keys"
)

type CreateModel[
	TRecord any,
	TRecordCreate comparable,
	TRecordEdit comparable,
	TId comparable,
	TFilter comparable,
] struct {
	// configuration
	crud *Crud[TRecord, TRecordCreate, TRecordEdit, TId, TFilter]

	// state
	publicKey client.PublicKey
	locked    *string
	focussed  bool
	preset    *TRecordCreate

	// util
	size util.Size

	// sub models
	form *form.Form[TRecordCreate]
}

func NewCreate[
	TRecord any,
	TRecordCreate comparable,
	TRecordEdit comparable,
	TId comparable,
	TFilter comparable,
](crud *Crud[TRecord, TRecordCreate, TRecordEdit, TId, TFilter], preset *TRecordCreate) *CreateModel[TRecord, TRecordCreate, TRecordEdit, TId, TFilter] {
	return &CreateModel[TRecord, TRecordCreate, TRecordEdit, TId, TFilter]{
		crud:   crud,
		preset: preset,
	}
}

func (m *CreateModel[TRecord, TRecordCreate, TRecordEdit, TId, TFilter]) Init() tea.Cmd {
	formOpts := m.crud.createFormRows()
	formOpts = append(formOpts,
		form.WithRow(
			form.WithAlign[TRecordCreate](form.Strech),
			form.WithItem[TRecordCreate]("_reset", formelement.NewButton("Reset",
				formelement.WithButtonActionReset(),
			)),
			form.WithItem[TRecordCreate]("_cancel", formelement.NewButton("Cancel",
				formelement.WithButtonActionCancel(),
				formelement.WithButtonGlobalKeyBindings(keys.Cancel()),
			)),
			form.WithItem[TRecordCreate]("_save", formelement.NewButton("Save",
				formelement.WithButtonActionSubmit(),
				formelement.WithButtonGlobalKeyBindings(keys.Save()),
			)),
		),
		// events
		form.WithOnSubmit(func(result TRecordCreate, err error) tea.Cmd {
			m.locked = util.NewPointer("Creating Record...")
			return func() tea.Msg {
				record, err := m.crud.createRecord(result)
				return createMsgCreateResult[TRecord]{record, err}
			}
		}),
		form.WithOnCancel[TRecordCreate](func() tea.Cmd {
			return m.crud.routerControll.Pop(1)
		}),
		form.WithOnDiscardGuard[TRecordCreate](discardGuard),
		// data
		form.WithInitialData(util.DerefOrNullValue(m.preset)),
	)

	m.form = util.NewPointer(form.New(formOpts...))

	return m.form.Init()
}

func (m *CreateModel[TRecord, TRecordCreate, TRecordEdit, TId, TFilter]) Update(msg tea.Msg) tea.Cmd {
	// Handle resizing
	if m.size.UpdateFromMsg(msg) {
		return m.form.Update(msg)
	}

	// Intercept messages
	if cmd, done := Intercept(
		msg,
		CreateMsgInterceptorCtx[TRecord, TRecordCreate, TRecordEdit, TId, TFilter]{m.crud, m.form},
		m.crud.createMsgInterceptors...,
	); cmd != nil || done {
		return cmd
	}

	// Handle messages
	switch msg := msg.(type) {
	case createMsgCreateResult[TRecord]:
		m.locked = nil
		if msg.err != nil {
			return popupviews.OpenMessage(popupviews.MessageError, "Error creating Record:\n"+msg.err.Error(), nil)
		}
		return tea.Sequence(m.crud.routerControll.Pop(1), func() tea.Msg { return createMsgCreated[TRecord]{msg.record} })
	}

	if !m.focussed || m.locked != nil {
		return nil
	}

	// pass remaining msgs to form
	return m.form.Update(msg)
}

func (m *CreateModel[TRecord, TRecordCreate, TRecordEdit, TId, TFilter]) View() string {
	if m.locked != nil {
		return *m.locked
	}
	return m.form.View()
}

func (m *CreateModel[TRecord, TRecordCreate, TRecordEdit, TId, TFilter]) Focus(parentKeyMap help.KeyMap) tea.Cmd {
	m.focussed = true
	return m.form.Focus(parentKeyMap)
}

func (m *CreateModel[TRecord, TRecordCreate, TRecordEdit, TId, TFilter]) Blur() {
	m.focussed = false
	m.form.Blur()
}

// *[CreateModel] implements [util.Model]
var _ util.Model = (*CreateModel[any, any, any, any, any])(nil)
