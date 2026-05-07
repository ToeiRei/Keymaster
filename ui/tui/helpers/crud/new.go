// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package crud

import (
	"context"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/ui/tui/components/router"
	"github.com/toeirei/keymaster/ui/tui/helpers/form"
	"github.com/toeirei/keymaster/ui/tui/popups/messagepopup"
	"github.com/toeirei/keymaster/ui/tui/util"
	"github.com/toeirei/keymaster/ui/tui/util/keys"
)

type Option[
	TRecord any,
	TRecordCreate comparable,
	TRecordUpdate comparable,
	TRecordId comparable,
	TFilter comparable,
] func(*Crud[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter])

func New[
	TRecord any,
	TRecordCreate comparable,
	TRecordUpdate comparable,
	TRecordId comparable,
	TFilter comparable,
](
	texts Texts,

	getRecordId func(record TRecord) TRecordId,
	getRecords func(ctx context.Context, filter TFilter) ([]TRecord, error),
	getRecord func(ctx context.Context, id TRecordId) (TRecord, error),
	createRecord func(ctx context.Context, recordCreate TRecordCreate) (TRecord, error),
	updateRecord func(ctx context.Context, id TRecordId, recordUpdate TRecordUpdate) (TRecord, error),
	deleteRecord func(ctx context.Context, id TRecordId) error,

	buildListTable func(records []TRecord, width int) ([]table.Column, []table.Row),
	recordToRecordUpdate func(record TRecord) TRecordUpdate,

	createFormRows func() []form.FormOpt[TRecordCreate],
	updateFormRows func() []form.FormOpt[TRecordUpdate],

	routerControll router.Controll,

	opts ...Option[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter],
) *Crud[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter] {
	crud := &Crud[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]{
		Texts: texts,

		getRecordId:  getRecordId,
		getRecords:   getRecords,
		getRecord:    getRecord,
		createRecord: createRecord,
		updateRecord: updateRecord,
		deleteRecord: deleteRecord,

		buildListTable:       buildListTable,
		recordToRecordUpdate: recordToRecordUpdate,
		createRecordPreset:   func() TRecordCreate { return util.NewZero[TRecordCreate]() },

		createFormRows: createFormRows,
		updateFormRows: updateFormRows,

		routerControll: routerControll,
	}

	for _, opt := range opts {
		opt(crud)
	}

	return crud
}

func WithListKeyBindings[
	TRecord any,
	TRecordCreate comparable,
	TRecordUpdate comparable,
	TRecordId comparable,
	TFilter comparable,
](bindings ...key.Binding) Option[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter] {
	return func(c *Crud[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]) {
		c.listGlobalKeyMap = append(c.listGlobalKeyMap, bindings...)
	}
}

func WithListMsgInterceptor[
	TRecord any,
	TRecordCreate comparable,
	TRecordUpdate comparable,
	TRecordId comparable,
	TFilter comparable,
](mi ListMsgInterceptor[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]) Option[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter] {
	return func(c *Crud[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]) {
		c.listMsgInterceptors = append(c.listMsgInterceptors, mi)
	}
}

func WithCreateMsgInterceptor[
	TRecord any,
	TRecordCreate comparable,
	TRecordUpdate comparable,
	TRecordId comparable,
	TFilter comparable,
](mi CreateMsgInterceptor[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]) Option[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter] {
	return func(c *Crud[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]) {
		c.createMsgInterceptors = append(c.createMsgInterceptors, mi)
	}
}

func WithUpdateMsgInterceptor[
	TRecord any,
	TRecordCreate comparable,
	TRecordUpdate comparable,
	TRecordId comparable,
	TFilter comparable,
](mi UpdateMsgInterceptor[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]) Option[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter] {
	return func(c *Crud[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]) {
		c.updateMsgInterceptors = append(c.updateMsgInterceptors, mi)
	}
}

func WithListReloadAfterChange[
	TRecord any,
	TRecordCreate comparable,
	TRecordUpdate comparable,
	TRecordId comparable,
	TFilter comparable,
](reload bool) Option[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter] {
	return func(c *Crud[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]) {
		c.listReloadAfterChange = reload
	}
}

func WithListAction[
	TRecord any,
	TRecordCreate comparable,
	TRecordUpdate comparable,
	TRecordId comparable,
	TFilter comparable,
](action func(ctx ListMsgInterceptorCtx[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]) tea.Cmd, bindings ...key.Binding) Option[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter] {
	return func(c *Crud[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]) {
		// add list key binding
		WithListKeyBindings[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter](bindings...)(c)

		// add list msg interceptor
		WithListMsgInterceptor(func(msg tea.Msg, ctx ListMsgInterceptorCtx[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]) (tea.Cmd, bool) {
			if msg, ok := msg.(tea.KeyMsg); ok && key.Matches(msg, bindings...) {
				return action(ctx), true
			}
			return nil, false
		})(c)
	}
}

func WithListDuplicateAction[
	TRecord any,
	TRecordCreate comparable,
	TRecordUpdate comparable,
	TRecordId comparable,
	TFilter comparable,
](recordToRecordCreate func(record TRecord) TRecordCreate) Option[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter] {
	return WithListAction(func(ctx ListMsgInterceptorCtx[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]) tea.Cmd {
		if ctx.SelectedRecord == nil {
			return messagepopup.Open(messagepopup.Error, "Please select a "+ctx.Crud.Texts.EntityNameSingular+" to duplicate.", nil)
		}

		return ctx.Crud.OpenCreate(recordToRecordCreate(*ctx.SelectedRecord))
	}, keys.Duplicate())
}

func WithCreateRecordPreset[
	TRecord any,
	TRecordCreate comparable,
	TRecordUpdate comparable,
	TRecordId comparable,
	TFilter comparable,
](createRecordPreset func() TRecordCreate) Option[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter] {
	return func(c *Crud[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]) {
		c.createRecordPreset = createRecordPreset
	}
}
