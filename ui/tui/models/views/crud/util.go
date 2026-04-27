// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package crud

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/form"
	popupviews "github.com/toeirei/keymaster/ui/tui/models/views/popup"
	"github.com/toeirei/keymaster/ui/tui/util/keys"
)

func discardGuard(confirmCmd tea.Cmd) tea.Cmd {
	return popupviews.OpenChoice(
		"You have unsaved changes. Do you want to discard them?",
		popupviews.Choices{
			{Name: "Cancel", Cmd: nil, KeyBindings: form.GlobalKeyMap{keys.Cancel()}},
			{Name: "Discard", Cmd: confirmCmd},
		},
		40, 40,
	)
}
