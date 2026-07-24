// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package crud

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/toeirei/keymaster/ui/tui/util/keys"
)

type ListKeyMap struct {
	LineUp       key.Binding
	LineDown     key.Binding
	PageUp       key.Binding
	PageDown     key.Binding
	HalfPageUp   key.Binding
	HalfPageDown key.Binding
	GotoTop      key.Binding
	GotoBottom   key.Binding
	Create       key.Binding
	Edit         key.Binding
	Duplicate    key.Binding
	Delete       key.Binding
	Exit         key.Binding
}

// ShortHelp / FullHelp rebuild their bindings from the keys.* builders so the
// help descriptions resolve in the current language at render time. The stored
// fields (and ListBaseKeyMap) remain the source of truth for key.Matches.
func (km ListKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{keys.LineUp(), keys.LineDown(), keys.Create(), keys.Edit(), keys.Duplicate(), keys.Delete(), keys.Exit()}
}

func (km ListKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{keys.LineUp(), keys.LineDown(), keys.PageUp(), keys.PageDown(), keys.HalfPageUp(), keys.HalfPageDown(), keys.GotoTop(), keys.GotoBottom()},
		{keys.Create(), keys.Edit(), keys.Duplicate(), keys.Delete(), keys.Exit()},
	}
}

// *[ListKeyMap] implements [help.KeyMap]
var _ help.KeyMap = (*ListKeyMap)(nil)

var ListBaseKeyMap = ListKeyMap{
	LineUp:       keys.LineUp(),
	LineDown:     keys.LineDown(),
	PageUp:       keys.PageUp(),
	PageDown:     keys.PageDown(),
	HalfPageUp:   keys.HalfPageUp(),
	HalfPageDown: keys.HalfPageDown(),
	GotoTop:      keys.GotoTop(),
	GotoBottom:   keys.GotoBottom(),
	Create:       keys.Create(),
	Edit:         keys.Edit(),
	Duplicate:    keys.Duplicate(),
	Delete:       keys.Delete(),
	Exit:         keys.Exit(),
}
