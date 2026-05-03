// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package util

import (
	tea "github.com/charmbracelet/bubbletea"
)

func TeaMsgToCmd(msg tea.Msg) tea.Cmd {
	return func() tea.Msg { return msg }
}
