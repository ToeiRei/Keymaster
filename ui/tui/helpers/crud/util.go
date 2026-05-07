// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package crud

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/ui/tui/popups/choicepopup"
	"github.com/toeirei/keymaster/ui/tui/util/keys"
)

func discardGuard(confirmCmd tea.Cmd) tea.Cmd {
	return choicepopup.Open(
		"You have unsaved changes. Do you want to discard them?",
		choicepopup.Choices{
			{Name: "Cancel", Cmd: nil, KeyBindings: keys.KeyBindingList{keys.Cancel()}},
			{Name: "Discard", Cmd: confirmCmd},
		},
	)
}
