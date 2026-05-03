// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package windowtitle

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/ui/tui/util"
)

type announceTitleMsg string
type denounceTitleMsg string

func Announce(title string) tea.Cmd {
	return util.TeaMsgToCmd(announceTitleMsg(title))
}

func Denounce(title string) tea.Cmd {
	return util.TeaMsgToCmd(denounceTitleMsg(title))
}
