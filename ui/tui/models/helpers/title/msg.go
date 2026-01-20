// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package windowtitle

import tea "github.com/charmbracelet/bubbletea"

type titleMsg string

func Set(title string) tea.Cmd {
	return func() tea.Msg { return titleMsg(title) }
}
