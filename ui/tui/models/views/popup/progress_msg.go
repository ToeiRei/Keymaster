// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package popupviews

import tea "github.com/charmbracelet/bubbletea"

type progressMsg interface {
	id() uint32
}

type progressFadeInMsg struct {
	pid uint32
}
type progressProgressMsg struct {
	pid      uint32
	progress float64
	status   string
}
type progressDoneMsg struct {
	pid uint32
	msg tea.Msg
}

func (m progressFadeInMsg) id() uint32   { return m.pid }
func (m progressProgressMsg) id() uint32 { return m.pid }
func (m progressDoneMsg) id() uint32     { return m.pid }

// [progressMsgFadeIn] implements [progressMsg]
// [progressMsgProgress] implements [progressMsg]
// [progressMsgDone] implements [progressMsg]
var _ progressMsg = progressFadeInMsg{}
var _ progressMsg = progressProgressMsg{}
var _ progressMsg = progressDoneMsg{}
