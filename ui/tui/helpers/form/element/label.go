// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package formelement

import (
	"fmt"

	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/ui/tui/helpers/form"
	"github.com/toeirei/keymaster/ui/tui/util/keys"
)

// Form elements take their labels/placeholders as fmt.Stringer (typically an
// i18n.Text) so the text resolves in the current language at render time.

// *[Label] implements [form.FormElement]
var _ form.FormElement = (*Label)(nil)

type Label struct {
	Text fmt.Stringer
}

func NewLabel(text fmt.Stringer) form.FormElement {
	return &Label{text}
}

func (l *Label) View(width int, eager bool) string {
	style := lipgloss.NewStyle().MaxWidth(width)
	if eager {
		style = style.Width(width)
	}
	return style.Render(l.Text.String())
}

func (l *Label) Focusable() bool {
	return false
}

// not needed
func (l *Label) Get() any                                  { return nil }
func (l *Label) Init() (tea.Cmd, keys.KeyBindingList)      { return nil, nil }
func (l *Label) Update(msg tea.Msg) (tea.Cmd, form.Action) { return nil, form.ActionNone }
func (l *Label) Reset()                                    {}
func (l *Label) Set(any)                                   {}
func (l *Label) Focus(parentKeyMap help.KeyMap) tea.Cmd    { return nil }
func (l *Label) Blur()                                     {}
