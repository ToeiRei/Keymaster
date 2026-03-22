// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package formelement

import (
	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/form"
)

type Label struct {
	Text string
}

func NewLabel(text string) form.FormElement {
	return &Label{text}
}

func (b *Label) View(width int) string {
	return b.Text
}

func (b *Label) Focusable() bool {
	return false
}

// not needed
func (b *Label) Get() any                                  { return nil }
func (b *Label) Init() tea.Cmd                             { return nil }
func (b *Label) Update(msg tea.Msg) (tea.Cmd, form.Action) { return nil, form.ActionNone }
func (b *Label) Reset()                                    {}
func (b *Label) Set(any)                                   {}
func (b *Label) Focus(baseKeyMap help.KeyMap) tea.Cmd      { return nil }
func (b *Label) Blur()                                     {}

var _ form.FormElement = (*Label)(nil)
