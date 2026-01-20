package form

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
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
	Next: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "next"),
	),
	Prev: key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("shift+tab", "previous"),
	),
}
