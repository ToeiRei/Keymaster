// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package crud

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/ui/i18n"
	"github.com/toeirei/keymaster/ui/tui/popups/choicepopup"
	"github.com/toeirei/keymaster/ui/tui/util/keys"
)

func discardGuard(confirmCmd tea.Cmd) tea.Cmd {
	return choicepopup.Open(
		i18n.Text("crud.discard_guard_question"),
		choicepopup.Choices{
			{Name: i18n.Text("crud.btn_cancel"), Cmd: nil, KeyBindings: keys.KeyBindingList{keys.Cancel()}},
			{Name: i18n.Text("crud.btn_discard"), Cmd: confirmCmd},
		},
	)
}
