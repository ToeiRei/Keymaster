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
	rid   uint32
	Model *util.Model
}
type PopMsg struct {
	rid   uint32
	Count int
}
type ChangeMsg struct {
	rid   uint32
	Model *util.Model
}

func (m InitMsg) routerID() uint32   { return m.RouterControll.rid }
func (m PushMsg) routerID() uint32   { return m.rid }
func (m PopMsg) routerID() uint32    { return m.rid }
func (m ChangeMsg) routerID() uint32 { return m.rid }

// *[InitMsg] implements [RouterMsg]
var _ RouterMsg = (*InitMsg)(nil)

// *[PushMsg] implements [RouterMsg]
var _ RouterMsg = (*PushMsg)(nil)

// *[PopMsg] implements [RouterMsg]
var _ RouterMsg = (*PopMsg)(nil)

// *[ChangeMsg] implements [RouterMsg]
var _ RouterMsg = (*ChangeMsg)(nil)

type RouterMsg interface {
	routerID() uint32
}
