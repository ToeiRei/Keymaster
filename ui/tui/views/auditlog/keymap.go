// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package auditlog

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/toeirei/keymaster/ui/tui/util/keys"
)

// KeyMap describes the audit-log page's key bindings for the help footer.
type KeyMap struct {
	Up       key.Binding
	Down     key.Binding
	PrevPage key.Binding
	NextPage key.Binding
	Exit     key.Binding
}

func (km KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{km.Up, km.Down, km.PrevPage, km.NextPage, km.Exit}
}

func (km KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{km.Up, km.Down},
		{km.PrevPage, km.NextPage, km.Exit},
	}
}

// *[KeyMap] implements [help.KeyMap]
var _ help.KeyMap = (*KeyMap)(nil)

var BaseKeyMap = KeyMap{
	Up:       keys.Up(),
	Down:     keys.Down(),
	PrevPage: key.NewBinding(key.WithKeys("left"), key.WithHelp("←", "prev page")),
	NextPage: key.NewBinding(key.WithKeys("right"), key.WithHelp("→", "next page")),
	Exit:     keys.Exit(),
}
