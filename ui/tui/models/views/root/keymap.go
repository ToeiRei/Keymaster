// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package root

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
)

type KeyMap struct {
	Exit key.Binding
	Help key.Binding
}

func (km KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{km.Exit, km.Help}
}

func (km KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{km.Help, km.Exit}}
}

// *KeyMap implements help.KeyMap
var _ help.KeyMap = (*KeyMap)(nil)

var BaseKeyMap = KeyMap{
	Exit: key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("ctrl+c", "exit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
}
