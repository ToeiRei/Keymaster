// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package crud

import (
	"context"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/ui/tui/components/router"
	"github.com/toeirei/keymaster/ui/tui/helpers/form"
	"github.com/toeirei/keymaster/ui/tui/util"
	"github.com/toeirei/keymaster/ui/tui/util/keys"
)

type Texts struct {
	EntityNameSingular string
	EntityNameMultiple string
}

type Crud[
	TRecord any,
	TRecordCreate comparable,
	TRecordUpdate comparable,
	TRecordId comparable,
	TFilter comparable,
] struct {
	Texts Texts

	getRecordId  func(record TRecord) TRecordId
	getRecords   func(ctx context.Context, filter TFilter) ([]TRecord, error)
	getRecord    func(ctx context.Context, id TRecordId) (TRecord, error)
	createRecord func(ctx context.Context, recordCreate TRecordCreate) (TRecord, error)
	updateRecord func(ctx context.Context, id TRecordId, recordUpdate TRecordUpdate) (TRecord, error)
	deleteRecord func(ctx context.Context, id TRecordId) error

	buildListTable       func(records []TRecord, width int) ([]table.Column, []table.Row)
	recordToRecordUpdate func(record TRecord) TRecordUpdate

	createFormRows     func() []form.FormOpt[TRecordCreate]
	updateFormRows     func() []form.FormOpt[TRecordUpdate]
	createRecordPreset func() TRecordCreate

	listGlobalKeyMap keys.KeyBindingList

	routerControll router.Controll

	// extra options

	listMsgInterceptors   []ListMsgInterceptor[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]
	createMsgInterceptors []CreateMsgInterceptor[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]
	updateMsgInterceptors []UpdateMsgInterceptor[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]

	listReloadAfterChange bool

	ReloadOnNextFocus bool
}

func (c *Crud[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]) OpenList() tea.Cmd {
	return c.routerControll.Push(util.ModelPointer(NewList(c)))
}
func (c *Crud[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]) OpenCreate(preset TRecordCreate) tea.Cmd {
	return c.routerControll.Push(util.ModelPointer(NewCreate(c, preset)))
}
func (c *Crud[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]) OpenEdit(record TRecord) tea.Cmd {
	return c.routerControll.Push(util.ModelPointer(NewUpdate(c, record)))
}
