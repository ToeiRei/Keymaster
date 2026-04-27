// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package crud

import (
	"github.com/charmbracelet/bubbles/table"
	"github.com/toeirei/keymaster/ui/tui/models/components/router"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/form"
)

type Crud[
	TRecord any,
	TRecordCreate comparable,
	TRecordEdit comparable,
	TId comparable,
	TFilter comparable,
] struct {
	getRecordId    func(record TRecord) TId
	getRecords     func(filter TFilter) ([]TRecord, error)
	getRecord      func(id TId) (TRecord, error)
	createRecord   func(record TRecordCreate) (TRecord, error)
	editRecord     func(id TId, record TRecordEdit) (TRecord, error)
	deleteRecord   func(id TId) error
	makeListTable  func(record []TRecord, width int) ([]table.Column, []table.Row)
	makeRecordEdit func(record TRecord) TRecordEdit

	createFormRows func() []form.FormOpt[TRecordCreate]
	editFormRows   func() []form.FormOpt[TRecordEdit]

	routerControll router.Controll

	listMsgInterceptors   []ListMsgInterceptor
	createMsgInterceptors []CreateMsgInterceptor[TRecordCreate]
	editMsgInterceptors   []EditMsgInterceptor[TRecordEdit]
}
