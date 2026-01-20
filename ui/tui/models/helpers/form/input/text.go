// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package forminput

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/form"
)

type Text struct {
	Label       string
	Placeholder string
	KeyMap      TextKeyMap

	input   textinput.Model
	focused bool
}

type TextKeyMap struct {
	Next key.Binding
}

func (k TextKeyMap) ShortHelp() []key.Binding { return []key.Binding{} }

func (k TextKeyMap) FullHelp() [][]key.Binding { return [][]key.Binding{} }

func NewText(label, placeholder string) form.FormInput {
	return &Text{
		Label:       label,
		Placeholder: placeholder,
		KeyMap: TextKeyMap{
			Next: key.NewBinding(
				key.WithKeys("enter"),
				key.WithHelp("enter", "next"),
			),
		},
		input: textinput.New(),
	}
}

func (t *Text) Blur() {
	t.input.Blur()
	t.focused = false
}

func (t *Text) Focus() (tea.Cmd, help.KeyMap) {
	t.focused = true
	return t.input.Focus(), t.KeyMap
}

func (t *Text) Get() any {
	return t.input.Value()
}

func (t *Text) Init() tea.Cmd {
	return nil
}

func (t *Text) Reset() {
	t.input.SetValue("")
}

func (t *Text) Set(value any) {
	if value, ok := value.(string); ok {
		t.input.SetValue(value)
	}
}

func (t *Text) Update(msg tea.Msg) (tea.Cmd, form.Action) {
	if msg, ok := msg.(tea.KeyMsg); ok && key.Matches(msg, t.KeyMap.Next) {
		return nil, form.ActionNext
	}

	var cmd tea.Cmd
	t.input, cmd = t.input.Update(msg)
	return cmd, form.ActionNone
}

func (t *Text) View(width int) string {
	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Width(width)

	focusedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true)

	label := t.Label
	if t.focused {
		label = focusedStyle.Render(label)
	} else {
		label = labelStyle.Render(label)
	}

	t.input.Width = width - 2
	t.input.Placeholder = t.Placeholder

	return lipgloss.JoinVertical(lipgloss.Left, label, t.input.View())
}

var _ form.FormInput = (*Text)(nil)
