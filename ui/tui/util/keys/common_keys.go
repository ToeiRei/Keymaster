// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package keys

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/toeirei/keymaster/ui/i18n"
)

// The help descriptions are resolved via i18n.T at call time, so a binding
// built fresh (e.g. from a keymap's ShortHelp/FullHelp) picks up the current
// language. The key combos themselves are never translated.

func Quit() key.Binding {
	return key.NewBinding(
		key.WithKeys("q"),
		key.WithHelp("q", i18n.T("keys.quit")),
	)
}
func Exit() key.Binding {
	return key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", i18n.T("keys.exit")),
	)
}
func Close() key.Binding {
	return key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", i18n.T("keys.close")),
	)
}
func Help() key.Binding {
	return key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", i18n.T("keys.help")),
	)
}

func Next() key.Binding {
	return key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", i18n.T("keys.next")),
	)
}
func NextEnter() key.Binding {
	return key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", i18n.T("keys.next")),
	)
}
func Prev() key.Binding {
	return key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("shift+tab", i18n.T("keys.previous")),
	)
}

func Up() key.Binding {
	return key.NewBinding(
		key.WithKeys("k", "up"),
		key.WithHelp("↑/k", i18n.T("keys.up")),
	)
}
func Down() key.Binding {
	return key.NewBinding(
		key.WithKeys("j", "down"),
		key.WithHelp("↓/j", i18n.T("keys.down")),
	)
}
func Left() key.Binding {
	return key.NewBinding(
		key.WithKeys("left"),
		key.WithHelp("←", i18n.T("keys.left")),
	)
}
func Right() key.Binding {
	return key.NewBinding(
		key.WithKeys("right"),
		key.WithHelp("→", i18n.T("keys.right")),
	)
}

func UpArrow() key.Binding {
	return key.NewBinding(
		key.WithKeys("up"),
		key.WithHelp("↑", i18n.T("keys.up")),
	)
}
func DownArrow() key.Binding {
	return key.NewBinding(
		key.WithKeys("down"),
		key.WithHelp("↓", i18n.T("keys.down")),
	)
}
func LeftArrow() key.Binding {
	return key.NewBinding(
		key.WithKeys("left"),
		key.WithHelp("←", i18n.T("keys.left")),
	)
}
func RightArrow() key.Binding {
	return key.NewBinding(
		key.WithKeys("right"),
		key.WithHelp("→", i18n.T("keys.right")),
	)
}

func LeftBack() key.Binding {
	return key.NewBinding(
		key.WithKeys("left", "backspace", "esc"),
		key.WithHelp("←/esc", i18n.T("keys.back")),
	)
}
func RightOpen() key.Binding {
	return key.NewBinding(
		key.WithKeys("right", "enter"),
		key.WithHelp("→/enter", i18n.T("keys.open")),
	)
}

func Open() key.Binding {
	return key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", i18n.T("keys.open")),
	)
}

func LineUp() key.Binding {
	return key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", i18n.T("keys.up")),
	)
}
func LineDown() key.Binding {
	return key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", i18n.T("keys.down")),
	)
}
func PageUp() key.Binding {
	return key.NewBinding(
		key.WithKeys("b", "pgup"),
		key.WithHelp("b/pgup", i18n.T("keys.page_up")),
	)
}
func PageDown() key.Binding {
	return key.NewBinding(
		key.WithKeys("f", "pgdown", " "),
		key.WithHelp("f/pgdn", i18n.T("keys.page_down")),
	)
}
func HalfPageUp() key.Binding {
	return key.NewBinding(
		key.WithKeys("u", "ctrl+u"),
		key.WithHelp("u", i18n.T("keys.half_page_up")),
	)
}
func HalfPageDown() key.Binding {
	return key.NewBinding(
		key.WithKeys("d", "ctrl+d"),
		key.WithHelp("d", i18n.T("keys.half_page_down")),
	)
}
func GotoTop() key.Binding {
	return key.NewBinding(
		key.WithKeys("home", "g"),
		key.WithHelp("g/home", i18n.T("keys.goto_start")),
	)
}
func GotoBottom() key.Binding {
	return key.NewBinding(
		key.WithKeys("end", "G"),
		key.WithHelp("G/end", i18n.T("keys.goto_end")),
	)
}

func Submit() key.Binding {
	return key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", i18n.T("keys.submit")),
	)
}
func Select() key.Binding {
	return key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", i18n.T("keys.select")),
	)
}
func Save() key.Binding {
	return key.NewBinding(
		key.WithKeys("ctrl+s"),
		key.WithHelp("ctrl+s", i18n.T("keys.save")),
	)
}
func SaveCreate() key.Binding {
	return key.NewBinding(
		key.WithKeys("ctrl+s"),
		key.WithHelp("ctrl+s", i18n.T("keys.create")),
	)
}
func Cancel() key.Binding {
	return key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", i18n.T("keys.cancel")),
	)
}

func Create() key.Binding {
	return key.NewBinding(
		key.WithKeys("a", "ctrl+n"),
		key.WithHelp("a/ctrl+n", i18n.T("keys.add_new")),
	)
}
func Edit() key.Binding {
	return key.NewBinding(
		key.WithKeys("e", "enter"),
		key.WithHelp("e/enter", i18n.T("keys.edit")),
	)
}
func Duplicate() key.Binding {
	return key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", i18n.T("keys.duplicate")),
	)
}
func Delete() key.Binding {
	return key.NewBinding(
		key.WithKeys("delete"),
		key.WithHelp("del", i18n.T("keys.delete")),
	)
}
