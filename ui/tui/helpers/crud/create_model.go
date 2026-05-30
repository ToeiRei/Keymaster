// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package crud

import (
	"context"

	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/ui/tui/helpers/form"
	formelement "github.com/toeirei/keymaster/ui/tui/helpers/form/element"
	windowtitle "github.com/toeirei/keymaster/ui/tui/helpers/title"
	"github.com/toeirei/keymaster/ui/tui/popups/messagepopup"
	"github.com/toeirei/keymaster/ui/tui/popups/progresspopup"
	"github.com/toeirei/keymaster/ui/tui/util"
	"github.com/toeirei/keymaster/ui/tui/util/keys"
)

type CreateModel[
	TRecord any,
	TRecordCreate comparable,
	TRecordUpdate comparable,
	TRecordId comparable,
	TFilter comparable,
] struct {
	// configuration
	crud *Crud[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]

	// state
	focussed bool
	preset   TRecordCreate

	// util
	size util.Size

	// sub models
	form *form.Form[TRecordCreate]
}

func NewCreate[
	TRecord any,
	TRecordCreate comparable,
	TRecordUpdate comparable,
	TRecordId comparable,
	TFilter comparable,
](crud *Crud[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter], preset TRecordCreate) *CreateModel[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter] {
	return &CreateModel[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]{
		crud:   crud,
		preset: preset,
	}
}

func (m *CreateModel[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]) Init() tea.Cmd {
	formOpts := append(m.crud.createFormRows(),
		// buttons
		form.WithRow(
			form.WithAlign[TRecordCreate](form.Strech),
			form.WithItem[TRecordCreate]("_reset", formelement.NewButton("Reset",
				formelement.WithButtonActionReset(),
			)),
			form.WithItem[TRecordCreate]("_cancel", formelement.NewButton("Cancel",
				formelement.WithButtonActionCancel(),
				formelement.WithButtonGlobalKeyBindings(keys.Cancel()),
			)),
			form.WithItem[TRecordCreate]("_create", formelement.NewButton("Create",
				formelement.WithButtonActionSubmit(),
				formelement.WithButtonGlobalKeyBindings(keys.SaveCreate()),
			)),
		),
		// events
		form.WithOnSubmit(func(result TRecordCreate, err error) (tea.Cmd, bool) {
			return progresspopup.Open(
				progresspopup.Spinner,
				"Creating "+m.crud.Texts.EntityNameSingular(),
				func(ctx context.Context, _ progresspopup.ProgressChan) tea.Cmd {
					record, err := m.crud.createRecord(ctx, result)
					return util.TeaMsgToCmd(createMsgCreateResult[TRecord]{record, err})
				},
				progresspopup.WithCancel(),
			), true
		}),
		form.WithOnCancel[TRecordCreate](func() tea.Cmd {
			return m.crud.routerControll.Pop(1)
		}),
		form.WithOnDiscardGuard[TRecordCreate](discardGuard),
		// data
		form.WithInitialData(m.preset),
	)

	m.form = util.NewPointer(form.New(formOpts...))

	return m.form.Init()
}

func (m *CreateModel[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]) Update(msg tea.Msg) tea.Cmd {
	// Handle resizing
	if m.size.UpdateFromMsg(msg) {
		return m.form.Update(msg)
	}

	// Intercept messages
	if cmd, done := Intercept(
		msg,
		CreateMsgInterceptorCtx[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]{m.crud, m.form},
		m.crud.createMsgInterceptors...,
	); cmd != nil || done {
		return cmd
	}

	// Handle messages
	switch msg := msg.(type) {
	case createMsgCreateResult[TRecord]:
		if msg.err != nil {
			return messagepopup.Open(messagepopup.Error, "Error creating "+m.crud.Texts.EntityNameSingular()+":\n"+msg.err.Error(), nil)
		}
		return tea.Sequence(m.crud.routerControll.Pop(1), util.TeaMsgToCmd(CreateMsgCreated[TRecord]{msg.record}))
	}

	if !m.focussed {
		return nil
	}

	// pass remaining msgs to form
	return m.form.Update(msg)
}

func (m *CreateModel[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]) View() string {
	return m.form.View()
}

func (m *CreateModel[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]) Focus(parentKeyMap help.KeyMap) tea.Cmd {
	if m.crud.ReloadOnNextFocus {
		m.crud.ReloadOnNextFocus = false
		return m.Init()
		// no need to focus or announce anything, as the popup interceptor will take it away again.
	}
	m.focussed = true
	return tea.Batch(
		windowtitle.Announce(m.crud.Texts.EntityNameMultiple()),
		m.form.Focus(parentKeyMap),
	)
}

func (m *CreateModel[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]) Blur() {
	m.focussed = false
	m.form.Blur()
}

// *[CreateModel] implements [util.Model]
var _ util.Model = (*CreateModel[any, any, any, any, any])(nil)
