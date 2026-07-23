// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package deploy

import (
	"fmt"

	"github.com/toeirei/keymaster/client"
)

func newUserRequester() *userRequester {
	return &userRequester{make(chan any), make(chan any)}
}

type TextRequest string
type TextReply string
type ChoiceRequest []string
type ChoiceReply int

type userRequester struct {
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

// *[userRequester] implements [client.UserRequester]
var _ client.UserRequester = (*userRequester)(nil)

func (ur *userRequester) RequestText(promt string) string {
	ur.request <- TextRequest(promt)
	reply := <-ur.reply
	if reply, ok := reply.(TextReply); ok {
		return string(reply)
	}
	panic(fmt.Sprintf("expected reply of type TextReply, got %T", reply))
}

func (ur *userRequester) RequestChoice(promts []string) int {
	ur.request <- ChoiceRequest(promts)
	reply := <-ur.reply
	if reply, ok := reply.(ChoiceReply); ok {
		return int(reply)
	}
	panic(fmt.Sprintf("expected reply of type ChoiceReply, got %T", reply))
}
