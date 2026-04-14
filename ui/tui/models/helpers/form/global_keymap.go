// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package form

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
)

type GlobalKeyMap []key.Binding

func (km GlobalKeyMap) ShortHelp() []key.Binding {
	return km
}

func (km GlobalKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{km}
}

// *[GlobalKeyMap] implements [help.KeyMap]
var _ help.KeyMap = (*GlobalKeyMap)(nil)
