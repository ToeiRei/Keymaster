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
	LineUp       key.Binding
	LineDown     key.Binding
	PageUp       key.Binding
	PageDown     key.Binding
	HalfPageUp   key.Binding
	HalfPageDown key.Binding
	GotoTop      key.Binding
	GotoBottom   key.Binding
	Select       key.Binding
	Cancel       key.Binding
}

func (km SelectKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{km.LineUp, km.LineDown, km.Select, km.Cancel}
}

func (km SelectKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{km.LineUp, km.LineDown, km.PageUp, km.PageDown, km.HalfPageUp, km.HalfPageDown, km.GotoTop, km.GotoBottom},
		{km.Select, km.Cancel},
	}
}

// *[SelectKeyMap] implements [help.KeyMap]
var _ help.KeyMap = (*SelectKeyMap)(nil)

var SelectBaseKeyMap = SelectKeyMap{
	LineUp:       keys.LineUp(),
	LineDown:     keys.LineDown(),
	PageUp:       keys.PageUp(),
	PageDown:     keys.PageDown(),
	HalfPageUp:   keys.HalfPageUp(),
	HalfPageDown: keys.HalfPageDown(),
	GotoTop:      keys.GotoTop(),
	GotoBottom:   keys.GotoBottom(),
	Select:       keys.Select(),
	Cancel:       keys.Cancel(),
}
