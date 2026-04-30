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

type Texts struct {
	EntityNameSingular string
	EntityNameMultiple string
}

type Crud[
	TRecord any,
	TRecordCreate comparable,
	TRecordUpdate comparable,
	TId comparable,
	TFilter comparable,
] struct {
	Texts Texts

	getRecordId  func(record TRecord) TId
	getRecords   func(filter TFilter) ([]TRecord, error)
	getRecord    func(id TId) (TRecord, error)
	createRecord func(record TRecordCreate) (TRecord, error)
	updateRecord   func(id TId, record TRecordUpdate) (TRecord, error)
	deleteRecord func(id TId) error

	makeListTable  func(record []TRecord, width int) ([]table.Column, []table.Row)
	makeRecordUpdate func(record TRecord) TRecordUpdate

	createFormRows func() []form.FormOpt[TRecordCreate]
	updateFormRows   func() []form.FormOpt[TRecordUpdate]

	listGlobalKeyMap form.GlobalKeyMap

	routerControll router.Controll

	listMsgInterceptors   []ListMsgInterceptor[TRecord, TRecordCreate, TRecordUpdate, TId, TFilter]
	createMsgInterceptors []CreateMsgInterceptor[TRecord, TRecordCreate, TRecordUpdate, TId, TFilter]
	updateMsgInterceptors   []UpdateMsgInterceptor[TRecord, TRecordCreate, TRecordUpdate, TId, TFilter]
}

func (c *Crud[TRecord, TRecordCreate, TRecordUpdate, TId, TFilter]) OpenList() tea.Cmd {
	return c.routerControll.Push(util.ModelPointer(NewList(c)))
}
func (c *Crud[TRecord, TRecordCreate, TRecordUpdate, TId, TFilter]) OpenCreate(preset *TRecordCreate) tea.Cmd {
	return c.routerControll.Push(util.ModelPointer(NewCreate(c, preset)))
}
func (c *Crud[TRecord, TRecordCreate, TRecordUpdate, TId, TFilter]) OpenEdit(record TRecord) tea.Cmd {
	return c.routerControll.Push(util.ModelPointer(NewUpdate(c, record)))
}
