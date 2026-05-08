// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package selectpopup

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/toeirei/keymaster/ui/tui/util/keys"
)

type SelectKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Select key.Binding
	Cancel key.Binding
}

func (km SelectKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{km.Up, km.Down, km.Select, km.Cancel}
}

func (km SelectKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{km.Up, km.Down},
		{km.Select, km.Cancel},
	}
}

// *[SelectKeyMap] implements [help.KeyMap]
var _ help.KeyMap = (*SelectKeyMap)(nil)

var SelectBaseKeyMap = SelectKeyMap{
	Up:     keys.UpArrow(),
	Down:   keys.DownArrow(),
	Select: keys.Select(),
	Cancel: keys.Cancel(),
}
