// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package crud

import (
	"github.com/charmbracelet/bubbles/table"
	"github.com/toeirei/keymaster/ui/tui/models/components/router"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/form"
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
		getRecordId:    getRecordId,
		getRecords:     getRecords,
		getRecord:      getRecord,
		createRecord:   createRecord,
		editRecord:     editRecord,
		deleteRecord:   deleteRecord,
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

func WithListMsgInterceptor[
	TRecord any,
	TRecordCreate comparable,
	TRecordEdit comparable,
	TId comparable,
	TFilter comparable,
](mi ListMsgInterceptor) Option[TRecord, TRecordCreate, TRecordEdit, TId, TFilter] {
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
](mi CreateMsgInterceptor[TRecordCreate]) Option[TRecord, TRecordCreate, TRecordEdit, TId, TFilter] {
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
](mi EditMsgInterceptor[TRecordEdit]) Option[TRecord, TRecordCreate, TRecordEdit, TId, TFilter] {
	return func(c *Crud[TRecord, TRecordCreate, TRecordEdit, TId, TFilter]) {
		c.editMsgInterceptors = append(c.editMsgInterceptors, mi)
	}
}
