// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package root

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/toeirei/keymaster/ui/tui/util/keys"
)

type KeyMap struct {
	Help key.Binding
}

func (km KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{km.Help}
}

func (km KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{km.Help}}
}

// *[KeyMap] implements [help.KeyMap]
var _ help.KeyMap = (*KeyMap)(nil)

var BaseKeyMap = KeyMap{
	Help: keys.Help(),
}
