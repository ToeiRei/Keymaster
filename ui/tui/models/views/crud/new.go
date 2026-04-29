// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package crud

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/ui/tui/models/components/router"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/form"
	popupviews "github.com/toeirei/keymaster/ui/tui/models/views/popup"
	"github.com/toeirei/keymaster/ui/tui/util"
	"github.com/toeirei/keymaster/ui/tui/util/keys"
)

type Option[
	TRecord any,
	TRecordCreate comparable,
	TRecordEdit comparable,
	TId comparable,
	TFilter comparable,
] func(*Crud[TRecord, TRecordCreate, TRecordEdit, TId, TFilter])

func New[
	TRecord any,
	TRecordCreate comparable,
	TRecordEdit comparable,
	TId comparable,
	TFilter comparable,
](
	texts Texts,

	getRecordId func(record TRecord) TId,
	getRecords func(filter TFilter) ([]TRecord, error),
	getRecord func(id TId) (TRecord, error),
	createRecord func(record TRecordCreate) (TRecord, error),
	editRecord func(id TId, record TRecordEdit) (TRecord, error),
	deleteRecord func(id TId) error,
	makeListTable func(record []TRecord, width int) ([]table.Column, []table.Row),
	makeRecordEdit func(record TRecord) TRecordEdit,

	createFormRows func() []form.FormOpt[TRecordCreate],
	editFormRows func() []form.FormOpt[TRecordEdit],

	routerControll router.Controll,

	opts ...Option[TRecord, TRecordCreate, TRecordEdit, TId, TFilter],
) *Crud[TRecord, TRecordCreate, TRecordEdit, TId, TFilter] {
	crud := &Crud[TRecord, TRecordCreate, TRecordEdit, TId, TFilter]{
		Texts: texts,

		getRecordId:  getRecordId,
		getRecords:   getRecords,
		getRecord:    getRecord,
		createRecord: createRecord,
		editRecord:   editRecord,
		deleteRecord: deleteRecord,

		makeListTable:  makeListTable,
		makeRecordEdit: makeRecordEdit,

		createFormRows: createFormRows,
		editFormRows:   editFormRows,

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
	TRecordEdit comparable,
	TId comparable,
	TFilter comparable,
](bindings ...key.Binding) Option[TRecord, TRecordCreate, TRecordEdit, TId, TFilter] {
	return func(c *Crud[TRecord, TRecordCreate, TRecordEdit, TId, TFilter]) {
		c.listGlobalKeyMap = append(c.listGlobalKeyMap, bindings...)
	}
}

func WithListMsgInterceptor[
	TRecord any,
	TRecordCreate comparable,
	TRecordEdit comparable,
	TId comparable,
	TFilter comparable,
](mi ListMsgInterceptor[TRecord, TRecordCreate, TRecordEdit, TId, TFilter]) Option[TRecord, TRecordCreate, TRecordEdit, TId, TFilter] {
	return func(c *Crud[TRecord, TRecordCreate, TRecordEdit, TId, TFilter]) {
		c.listMsgInterceptors = append(c.listMsgInterceptors, mi)
	}
}

func WithCreateMsgInterceptor[
	TRecord any,
	TRecordCreate comparable,
	TRecordEdit comparable,
	TId comparable,
	TFilter comparable,
](mi CreateMsgInterceptor[TRecord, TRecordCreate, TRecordEdit, TId, TFilter]) Option[TRecord, TRecordCreate, TRecordEdit, TId, TFilter] {
	return func(c *Crud[TRecord, TRecordCreate, TRecordEdit, TId, TFilter]) {
		c.createMsgInterceptors = append(c.createMsgInterceptors, mi)
	}
}

func WithEditMsgInterceptor[
	TRecord any,
	TRecordCreate comparable,
	TRecordEdit comparable,
	TId comparable,
	TFilter comparable,
](mi EditMsgInterceptor[TRecord, TRecordCreate, TRecordEdit, TId, TFilter]) Option[TRecord, TRecordCreate, TRecordEdit, TId, TFilter] {
	return func(c *Crud[TRecord, TRecordCreate, TRecordEdit, TId, TFilter]) {
		c.editMsgInterceptors = append(c.editMsgInterceptors, mi)
	}
}

func WithListDuplicateAction[
	TRecord any,
	TRecordCreate comparable,
	TRecordEdit comparable,
	TId comparable,
	TFilter comparable,
](fn func(record TRecord) TRecordCreate) Option[TRecord, TRecordCreate, TRecordEdit, TId, TFilter] {
	return func(c *Crud[TRecord, TRecordCreate, TRecordEdit, TId, TFilter]) {
		// add list key binding
		WithListKeyBindings[TRecord, TRecordCreate, TRecordEdit, TId, TFilter](keys.Duplicate())(c)

		// add list msg interceptor
		WithListMsgInterceptor(func(msg tea.Msg, ctx ListMsgInterceptorCtx[TRecord, TRecordCreate, TRecordEdit, TId, TFilter]) (tea.Cmd, bool) {
			if msg, ok := msg.(tea.KeyMsg); ok && key.Matches(msg, keys.Duplicate()) {
				if ctx.SelectedRecord == nil {
					return popupviews.OpenMessage(popupviews.MessageError, "Please select a "+ctx.Crud.Texts.EntityNameSingular+" to duplicate.", nil), true
				}
				return ctx.Crud.OpenCreate(util.NewPointer(fn(*ctx.SelectedRecord))), true
			}
			return nil, false
		})(c)
	}
}
