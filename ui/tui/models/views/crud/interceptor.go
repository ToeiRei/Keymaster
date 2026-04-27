// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package crud

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/form"
)

type MsgInterceptor[T any] = func(msg tea.Msg, ctx T) (cmd tea.Cmd, done bool)

type ListMsgInterceptor = MsgInterceptor[struct{}]
type CreateMsgInterceptor[T comparable] = MsgInterceptor[*form.Form[T]]
type EditMsgInterceptor[T comparable] = MsgInterceptor[*form.Form[T]]

func Intercept[T any](msg tea.Msg, ctx T, interceptors ...MsgInterceptor[T]) (cmd tea.Cmd, done bool) {
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
