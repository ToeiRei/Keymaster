// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package testview1

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
)

type KeyMap struct {
	Quit key.Binding
}

func (km KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{km.Quit}
}

func (km KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{km.Quit}}
}

// *KeyMap implements help.KeyMap
var _ help.KeyMap = (*KeyMap)(nil)

var DefaultKeyMap = KeyMap{
	Quit: key.NewBinding(
		key.WithKeys("q"),
		key.WithHelp("q", "quit test view"),
	),
}
