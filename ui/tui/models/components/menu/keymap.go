// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// Package menu contains menu components used by the TUI.
package menu

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/toeirei/keymaster/ui/tui/util/keys"
)

type KeyMap struct {
	Up    key.Binding
	Down  key.Binding
	Left  key.Binding
	Right key.Binding
	Quit  key.Binding
}

func (km KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{km.Up, km.Down, km.Right, km.Left}
}

func (km KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{km.Up, km.Down}, {km.Right, km.Left}}
}

// *[KeyMap] implements [help.KeyMap]
var _ help.KeyMap = (*KeyMap)(nil)

var DefaultKeyMap = KeyMap{
	Up:    keys.Up(),
	Down:  keys.Down(),
	Left:  keys.Left(),
	Right: keys.Right(),
	Quit:  keys.Quit(),
}
