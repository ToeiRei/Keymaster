// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package formelement

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/form"
)

// *[Keybind] implements [form.FormElement]
var _ form.FormElement = (*Keybind)(nil)

type Keybind struct {
	Action            func() (tea.Cmd, form.Action)
	GlobalKeyBindings form.GlobalKeyMap
}

func NewKeybind(action func() (tea.Cmd, form.Action), globalKeyBindings ...key.Binding) form.FormElement {
	return &Keybind{
		Action:            action,
		GlobalKeyBindings: globalKeyBindings,
	}
}

func (b *Keybind) Update(msg tea.Msg) (tea.Cmd, form.Action) {
	// msg is KeyMsg
	// key matches global binding
	// Action not nil
	if msg, ok := msg.(tea.KeyMsg); ok && key.Matches(msg, b.GlobalKeyBindings...) && b.Action != nil {
		return b.Action()
	}
	return nil, form.ActionNone
}

func (b *Keybind) Init() (tea.Cmd, form.GlobalKeyMap) {
	return nil, b.GlobalKeyBindings
}

// not needed
func (b *Keybind) Focus(parentKeyMap help.KeyMap) tea.Cmd { return nil }
func (b *Keybind) Blur()                                  {}
func (b *Keybind) View(width int) string                  { return "" }
func (b *Keybind) Focusable() bool                        { return false }
func (b *Keybind) Get() any                               { return nil }
func (b *Keybind) Reset()                                 {}
func (b *Keybind) Set(any)                                {}
