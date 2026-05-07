// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package progresspopup

import tea "github.com/charmbracelet/bubbletea"

type progressMsg interface {
	id() progressId
}

type progressMsgProgress struct {
	pid      progressId
	progress float64
	status   string
}
type progressMsgDone struct {
	pid progressId
	cmd tea.Cmd
}

func (m progressMsgProgress) id() progressId { return m.pid }
func (m progressMsgDone) id() progressId     { return m.pid }

// [progressMsgProgress] implements [progressMsg]
// [progressMsgDone] implements [progressMsg]
var _ progressMsg = progressMsgProgress{}
var _ progressMsg = progressMsgDone{}
