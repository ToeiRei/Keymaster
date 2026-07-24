// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package auditlog

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/toeirei/keymaster/ui/i18n"
	"github.com/toeirei/keymaster/ui/tui/util/keys"
)

// prevPage / nextPage are the audit log's arrow-based paging bindings. They live
// here (not in the shared keys package) because their labels are page-specific;
// like the keys.* builders they resolve their help text via i18n.T at call time.
func prevPage() key.Binding {
	return key.NewBinding(key.WithKeys("left"), key.WithHelp("←", i18n.T("keys.prev_page")))
}
func nextPage() key.Binding {
	return key.NewBinding(key.WithKeys("right"), key.WithHelp("→", i18n.T("keys.next_page")))
}

// KeyMap describes the audit-log page's key bindings for the help footer.
type KeyMap struct {
	Up       key.Binding
	Down     key.Binding
	PrevPage key.Binding
	NextPage key.Binding
	Exit     key.Binding
}

func (km KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{keys.Up(), keys.Down(), prevPage(), nextPage(), keys.Exit()}
}

func (km KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{keys.Up(), keys.Down()},
		{prevPage(), nextPage(), keys.Exit()},
	}
}

// *[KeyMap] implements [help.KeyMap]
var _ help.KeyMap = (*KeyMap)(nil)

var BaseKeyMap = KeyMap{
	Up:       keys.Up(),
	Down:     keys.Down(),
	PrevPage: prevPage(),
	NextPage: nextPage(),
	Exit:     keys.Exit(),
}
