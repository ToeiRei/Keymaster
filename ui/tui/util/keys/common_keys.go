// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package keys

import "github.com/charmbracelet/bubbles/key"

func Quit() key.Binding {
	return key.NewBinding(
		key.WithKeys("q"),
		key.WithHelp("q", "quit"),
	)
}
func Exit() key.Binding {
	return key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "exit"),
	)
}
func Help() key.Binding {
	return key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	)
}

func Next() key.Binding {
	return key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "next"),
	)
}
func Prev() key.Binding {
	return key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("shift+tab", "previous"),
	)
}

func Up() key.Binding {
	return key.NewBinding(
		key.WithKeys("k", "up"),
		key.WithHelp("↑/k", "up"),
	)
}
func Down() key.Binding {
	return key.NewBinding(
		key.WithKeys("j", "down"),
		key.WithHelp("↓/j", "down"),
	)
}
func Left() key.Binding {
	return key.NewBinding(
		key.WithKeys("left", "backspace", "esc"),
		key.WithHelp("←/esc", "back"),
	)
}
func Right() key.Binding {
	return key.NewBinding(
		key.WithKeys("right", "enter"),
		key.WithHelp("→/enter", "select/open"),
	)
}

func LineUp() key.Binding {
	return key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	)
}
func LineDown() key.Binding {
	return key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	)
}
func PageUp() key.Binding {
	return key.NewBinding(
		key.WithKeys("b", "pgup"),
		key.WithHelp("b/pgup", "page up"),
	)
}
func PageDown() key.Binding {
	return key.NewBinding(
		key.WithKeys("f", "pgdown", " "),
		key.WithHelp("f/pgdn", "page down"),
	)
}
func HalfPageUp() key.Binding {
	return key.NewBinding(
		key.WithKeys("u", "ctrl+u"),
		key.WithHelp("u", "½ page up"),
	)
}
func HalfPageDown() key.Binding {
	return key.NewBinding(
		key.WithKeys("d", "ctrl+d"),
		key.WithHelp("d", "½ page down"),
	)
}
func GotoTop() key.Binding {
	return key.NewBinding(
		key.WithKeys("home", "g"),
		key.WithHelp("g/home", "go to start"),
	)
}
func GotoBottom() key.Binding {
	return key.NewBinding(
		key.WithKeys("end", "G"),
		key.WithHelp("G/end", "go to end"),
	)
}

func Submit() key.Binding {
	return key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "submit"),
	)
}
func Cancel() key.Binding {
	return key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	)
}

func Create() key.Binding {
	return key.NewBinding(
		key.WithKeys("a", "ctrl+n"),
		key.WithHelp("a/ctrl+n", "add/new"),
	)
}
func Edit() key.Binding {
	return key.NewBinding(
		key.WithKeys("e", "enter"),
		key.WithHelp("e/enter", "edit"),
	)
}
func Delete() key.Binding {
	return key.NewBinding(
		key.WithKeys("del"),
		key.WithHelp("del", "delete"),
	)
}
