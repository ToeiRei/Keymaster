// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package crud

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/form"
)

type MsgInterceptor[TCtx any] = func(msg tea.Msg, ctx TCtx) (cmd tea.Cmd, done bool)

type ListMsgInterceptor[
	TRecord any,
	TRecordCreate comparable,
	TRecordEdit comparable,
	TId comparable,
	TFilter comparable,
] = MsgInterceptor[ListMsgInterceptorCtx[TRecord, TRecordCreate, TRecordEdit, TId, TFilter]]

type CreateMsgInterceptor[
	TRecord any,
	TRecordCreate comparable,
	TRecordEdit comparable,
	TId comparable,
	TFilter comparable,
] = MsgInterceptor[CreateMsgInterceptorCtx[TRecord, TRecordCreate, TRecordEdit, TId, TFilter]]

type EditMsgInterceptor[
	TRecord any,
	TRecordCreate comparable,
	TRecordEdit comparable,
	TId comparable,
	TFilter comparable,
] = MsgInterceptor[EditMsgInterceptorCtx[TRecord, TRecordCreate, TRecordEdit, TId, TFilter]]

type ListMsgInterceptorCtx[
	TRecord any,
	TRecordCreate comparable,
	TRecordEdit comparable,
	TId comparable,
	TFilter comparable,
] struct {
	Crud           *Crud[TRecord, TRecordCreate, TRecordEdit, TId, TFilter]
	SelectedRecord *TRecord
}

type CreateMsgInterceptorCtx[
	TRecord any,
	TRecordCreate comparable,
	TRecordEdit comparable,
	TId comparable,
	TFilter comparable,
] struct {
	Crud *Crud[TRecord, TRecordCreate, TRecordEdit, TId, TFilter]
	Form *form.Form[TRecordCreate]
}

type EditMsgInterceptorCtx[
	TRecord any,
	TRecordCreate comparable,
	TRecordEdit comparable,
	TId comparable,
	TFilter comparable,
] struct {
	Crud *Crud[TRecord, TRecordCreate, TRecordEdit, TId, TFilter]
	Form *form.Form[TRecordEdit]
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
