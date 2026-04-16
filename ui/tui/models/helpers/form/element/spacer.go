// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package formelement

import (
	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/form"
)

// *[Spacer] implements [form.FormElement]
var _ form.FormElement = (*Spacer)(nil)

type Spacer struct {
	Space int
}

func NewSpacer(space int) form.FormElement {
	return &Spacer{space}
}

func (s *Spacer) View(width int, eager bool) string {
	style := lipgloss.NewStyle().MaxWidth(width - 2)
	if eager {
		style = style.Width(width - 2)
	}
	if s.Space > 1 {
		style = style.MarginBottom(s.Space - 1)
	}
	return style.Render("")
}

func (s *Spacer) Focusable() bool { return false }

// not needed
func (s *Spacer) Get() any                                  { return nil }
func (s *Spacer) Init() (tea.Cmd, form.GlobalKeyMap)        { return nil, nil }
func (s *Spacer) Update(msg tea.Msg) (tea.Cmd, form.Action) { return nil, form.ActionNone }
func (s *Spacer) Reset()                                    {}
func (s *Spacer) Set(any)                                   {}
func (s *Spacer) Focus(parentKeyMap help.KeyMap) tea.Cmd    { return nil }
func (s *Spacer) Blur()                                     {}

var _ form.FormElement = (*Spacer)(nil)
