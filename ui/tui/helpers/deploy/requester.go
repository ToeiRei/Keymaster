// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package deploy

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/client"
)

func newUserRequester(ctx context.Context) *userRequester {
	return &userRequester{ctx, make(chan any), make(chan any)}
}

type TextRequest string
type TextReply string
type ChoiceRequest []string
type ChoiceReply int

type userRequester struct {
	ctx     context.Context
	request chan any
	reply   chan any
}

func (ur *userRequester) Close() {
	close(ur.request)
	close(ur.reply)
}

func (ur *userRequester) ReplyText(text string) {
	ur.reply <- TextReply(text)
}

func (ur *userRequester) ReplyChoice(index int) {
	ur.reply <- ChoiceReply(index)
}

// replyTextCmd wraps the blocking [userRequester.ReplyText] send in a [tea.Cmd]
// so that replying never blocks the UI render loop.
func (ur *userRequester) replyTextCmd(text string) tea.Cmd {
	return func() tea.Msg { ur.ReplyText(text); return nil }
}

// replyChoiceCmd wraps the blocking [userRequester.ReplyChoice] send in a
// [tea.Cmd] so that replying never blocks the UI render loop.
func (ur *userRequester) replyChoiceCmd(index int) tea.Cmd {
	return func() tea.Msg { ur.ReplyChoice(index); return nil }
}

// *[userRequester] implements [client.UserRequester]
var _ client.UserRequester = (*userRequester)(nil)

// RequestText sends a text request to the TUI and blocks until the user
// replies. It selects on the context so a cancelled operation unblocks the
// parked connector goroutine, returning "" as an abort sentinel. It also
// degrades to "" instead of panicking on a closed channel or unexpected
// reply type.
func (ur *userRequester) RequestText(promt string) string {
	select {
	case ur.request <- TextRequest(promt):
	case <-ur.ctx.Done():
		return ""
	}

	select {
	case reply, ok := <-ur.reply:
		if !ok {
			return ""
		}
		if reply, ok := reply.(TextReply); ok {
			return string(reply)
		}
		return ""
	case <-ur.ctx.Done():
		return ""
	}
}

// RequestChoice sends a choice request to the TUI and blocks until the user
// replies. It selects on the context so a cancelled operation unblocks the
// parked connector goroutine, returning -1 as an abort sentinel. It also
// degrades to -1 instead of panicking on a closed channel or unexpected
// reply type.
func (ur *userRequester) RequestChoice(promts []string) int {
	select {
	case ur.request <- ChoiceRequest(promts):
	case <-ur.ctx.Done():
		return -1
	}

	select {
	case reply, ok := <-ur.reply:
		if !ok {
			return -1
		}
		if reply, ok := reply.(ChoiceReply); ok {
			return int(reply)
		}
		return -1
	case <-ur.ctx.Done():
		return -1
	}
}
