// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package router

import (
	"github.com/toeirei/keymaster/ui/tui/util"
)

// Router invoked messages
// Router -> Model

type InitMsg struct {
	RouterControll Controll
}

// RouterControll invoked messages
// Model-RouterControll -> Router

type PushMsg struct {
	rid   int
	Model *util.Model
}
type PopMsg struct {
	rid   int
	Count int
}
type ChangeMsg struct {
	rid   int
	Model *util.Model
}

func (m InitMsg) routerId() int   { return m.RouterControll.rid }
func (m PushMsg) routerId() int   { return m.rid }
func (m PopMsg) routerId() int    { return m.rid }
func (m ChangeMsg) routerId() int { return m.rid }
func (m InitMsg) routerID() int   { return m.RouterControll.rid }
func (m PushMsg) routerID() int   { return m.rid }
func (m PopMsg) routerID() int    { return m.rid }
func (m ChangeMsg) routerID() int { return m.rid }

type RouterMsg interface {
	routerID() int
}
