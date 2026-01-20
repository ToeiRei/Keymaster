package menu

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
)

type KeyMap struct {
	Up    key.Binding
	Down  key.Binding
	Left  key.Binding
	Right key.Binding
}

func (km KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{km.Up, km.Down, km.Right, km.Left}
}

func (km KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{km.Up, km.Down}, {km.Right, km.Left}}
}

// *KeyMap implements help.KeyMap
var _ help.KeyMap = (*KeyMap)(nil)

// ↑ ↓ ← →
var DefaultKeyMap = KeyMap{
	Up: key.NewBinding(
		key.WithKeys("k", "up"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("j", "down"),
		key.WithHelp("↓/j", "down"),
	),
	Left: key.NewBinding(
		key.WithKeys("left", "backspace", "esc"),
		key.WithHelp("←/esc", "back"),
	),
	Right: key.NewBinding(
		key.WithKeys("right", "enter"),
		key.WithHelp("→/enter", "select/open"),
	),
}
