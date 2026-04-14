// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package publickey

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

func (km ListKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{km.LineUp, km.LineDown, km.Create, km.Edit, km.Duplicate, km.Delete, km.Exit}
}

func (km ListKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{km.LineUp, km.LineDown, km.PageUp, km.PageDown, km.HalfPageUp, km.HalfPageDown, km.GotoTop, km.GotoBottom},
		{km.Create, km.Edit, km.Duplicate, km.Delete, km.Exit},
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
