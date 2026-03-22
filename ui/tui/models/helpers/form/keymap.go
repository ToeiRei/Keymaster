// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package form

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/toeirei/keymaster/ui/tui/util/keys"
)

type KeyMap struct {
	Next key.Binding
	Prev key.Binding
}

func (km KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{km.Next, km.Prev}
}

func (km KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{km.Next, km.Prev}}
}

// *KeyMap implements help.KeyMap
var _ help.KeyMap = (*KeyMap)(nil)

var DefaultKeyMap = KeyMap{
	Next: keys.Next(),
	Prev: keys.Prev(),
}
