// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package keys

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
)

type KeyBindingList []key.Binding

func (km KeyBindingList) ShortHelp() []key.Binding {
	return km
}

func (km KeyBindingList) FullHelp() [][]key.Binding {
	return [][]key.Binding{km}
}

// *[KeyBindingList] implements [help.KeyMap]
var _ help.KeyMap = (*KeyBindingList)(nil)
