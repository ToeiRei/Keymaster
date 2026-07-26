// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package crud

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/ui/tui/helpers/form"
)

type MsgInterceptor[TCtx any] = func(msg tea.Msg, ctx TCtx) (cmd tea.Cmd, done bool)

type ListMsgInterceptor[
	TRecord any,
	TRecordCreate any,
	TRecordUpdate any,
	TRecordId comparable,
	TFilter any,
] = MsgInterceptor[ListMsgInterceptorCtx[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]]

type CreateMsgInterceptor[
	TRecord any,
	TRecordCreate any,
	TRecordUpdate any,
	TRecordId comparable,
	TFilter any,
] = MsgInterceptor[CreateMsgInterceptorCtx[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]]

type UpdateMsgInterceptor[
	TRecord any,
	TRecordCreate any,
	TRecordUpdate any,
	TRecordId comparable,
	TFilter any,
] = MsgInterceptor[UpdateMsgInterceptorCtx[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]]

type ListMsgInterceptorCtx[
	TRecord any,
	TRecordCreate any,
	TRecordUpdate any,
	TRecordId comparable,
	TFilter any,
] struct {
	Crud           *Crud[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]
	SelectedRecord *TRecord
}

type CreateMsgInterceptorCtx[
	TRecord any,
	TRecordCreate any,
	TRecordUpdate any,
	TRecordId comparable,
	TFilter any,
] struct {
	Crud *Crud[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]
	Form *form.Form[TRecordCreate]
}

type UpdateMsgInterceptorCtx[
	TRecord any,
	TRecordCreate any,
	TRecordUpdate any,
	TRecordId comparable,
	TFilter any,
] struct {
	Crud *Crud[TRecord, TRecordCreate, TRecordUpdate, TRecordId, TFilter]
	Form *form.Form[TRecordUpdate]
}

func Intercept[TCtx any](msg tea.Msg, ctx TCtx, interceptors ...MsgInterceptor[TCtx]) (cmd tea.Cmd, done bool) {
	var cmds []tea.Cmd

	for _, interceptor := range interceptors {
		iCmd, iDone := interceptor(msg, ctx)
		cmds = append(cmds, iCmd)
		if iDone {
			done = true
			break
		}
	}

	cmd = tea.Sequence(cmds...)
	return
}
