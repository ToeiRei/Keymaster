// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package popupviews

import tea "github.com/charmbracelet/bubbletea"

type progressMsg interface {
	id() progressId
}

type progressFadeInMsg struct {
	pid progressId
}
type progressProgressMsg struct {
	pid      progressId
	progress float64
	status   string
}
type progressDoneMsg struct {
	pid progressId
	cmd tea.Cmd
}

func (m progressFadeInMsg) id() progressId   { return m.pid }
func (m progressProgressMsg) id() progressId { return m.pid }
func (m progressDoneMsg) id() progressId     { return m.pid }

// [progressMsgFadeIn] implements [progressMsg]
// [progressMsgProgress] implements [progressMsg]
// [progressMsgDone] implements [progressMsg]
var _ progressMsg = progressFadeInMsg{}
var _ progressMsg = progressProgressMsg{}
var _ progressMsg = progressDoneMsg{}
