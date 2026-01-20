// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package stack

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/ui/tui/util"
	"github.com/toeirei/keymaster/util/slicest"
)

type MsgFilter = func(model util.Model, msg tea.Msg) tea.Msg

func applyMessageFilters(model util.Model, msg tea.Msg, msg_filters []MsgFilter) tea.Msg {
	return slicest.ReduceD(msg_filters, msg, func(msg_filter MsgFilter, msg tea.Msg) tea.Msg {
		if msg == nil {
			return nil
		}
		return msg_filter(model, msg)
	})
}
