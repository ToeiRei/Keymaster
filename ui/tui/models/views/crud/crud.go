// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package crud

import (
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/ui/tui/models/components/router"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/form"
	"github.com/toeirei/keymaster/ui/tui/util"
)

type Crud[
	TRecord any,
	TRecordCreate comparable,
	TRecordEdit comparable,
	TId comparable,
	TFilter comparable,
] struct {
	getRecordId  func(record TRecord) TId
	getRecords   func(filter TFilter) ([]TRecord, error)
	getRecord    func(id TId) (TRecord, error)
	createRecord func(record TRecordCreate) (TRecord, error)
	editRecord   func(id TId, record TRecordEdit) (TRecord, error)
	deleteRecord func(id TId) error

	makeListTable  func(record []TRecord, width int) ([]table.Column, []table.Row)
	makeRecordEdit func(record TRecord) TRecordEdit

	createFormRows func() []form.FormOpt[TRecordCreate]
	editFormRows   func() []form.FormOpt[TRecordEdit]

	listGlobalKeyMap form.GlobalKeyMap

	routerControll router.Controll

	listMsgInterceptors   []ListMsgInterceptor[TRecord, TRecordCreate, TRecordEdit, TId, TFilter]
	createMsgInterceptors []CreateMsgInterceptor[TRecord, TRecordCreate, TRecordEdit, TId, TFilter]
	editMsgInterceptors   []EditMsgInterceptor[TRecord, TRecordCreate, TRecordEdit, TId, TFilter]
}

func (c *Crud[TRecord, TRecordCreate, TRecordEdit, TId, TFilter]) OpenList() tea.Cmd {
	return c.routerControll.Push(util.ModelPointer(NewList(c)))
}
func (c *Crud[TRecord, TRecordCreate, TRecordEdit, TId, TFilter]) OpenCreate(preset *TRecordCreate) tea.Cmd {
	return c.routerControll.Push(util.ModelPointer(NewCreate(c, preset)))
}
func (c *Crud[TRecord, TRecordCreate, TRecordEdit, TId, TFilter]) OpenEdit(record TRecord) tea.Cmd {
	return c.routerControll.Push(util.ModelPointer(NewEdit(c, record)))
}
