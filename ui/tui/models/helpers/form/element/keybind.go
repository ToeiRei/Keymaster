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

func (k *Keybind) Update(msg tea.Msg) (tea.Cmd, form.Action) {
	// msg is KeyMsg
	// key matches global binding
	// Action not nil
	if msg, ok := msg.(tea.KeyMsg); ok && key.Matches(msg, k.GlobalKeyBindings...) && k.Action != nil {
		return k.Action()
	}
	return nil, form.ActionNone
}

func (k *Keybind) Init() (tea.Cmd, form.GlobalKeyMap) {
	return nil, k.GlobalKeyBindings
}

// not needed
func (k *Keybind) Focus(parentKeyMap help.KeyMap) tea.Cmd { return nil }
func (k *Keybind) Blur()                                  {}
func (k *Keybind) View(width int) string                  { return "" }
func (k *Keybind) Focusable() bool                        { return false }
func (k *Keybind) Get() any                               { return nil }
func (k *Keybind) Reset()                                 {}
func (k *Keybind) Set(any)                                {}
