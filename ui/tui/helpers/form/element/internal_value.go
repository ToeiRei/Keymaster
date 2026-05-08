// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package formelement

import (
	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/ui/tui/helpers/form"
	"github.com/toeirei/keymaster/ui/tui/util"
	"github.com/toeirei/keymaster/ui/tui/util/keys"
)

// *[InternalValue] implements [form.FormElement]
var _ form.FormElement = (*InternalValue)(nil)

type InternalValue struct{ value any }

func NewInternalValue() form.FormElement { return &InternalValue{} }

func (t *InternalValue) Focus(parentKeyMap help.KeyMap) tea.Cmd {
	return util.AnnounceKeyMapCmd(parentKeyMap)
}

func (t *InternalValue) Blur() {}

func (t *InternalValue) Get() any { return t.value }

func (t *InternalValue) Init() (tea.Cmd, keys.KeyBindingList) { return nil, nil }

func (t *InternalValue) Reset() { t.value = nil }

func (t *InternalValue) Set(value any) { t.value = value }

func (t *InternalValue) Update(msg tea.Msg) (tea.Cmd, form.Action) { return nil, form.ActionNone }

func (t *InternalValue) View(width int, eager bool) string { return "" }

func (t *InternalValue) Focusable() bool { return false }
