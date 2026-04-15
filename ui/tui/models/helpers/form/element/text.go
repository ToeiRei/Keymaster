// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package formelement

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/form"
	"github.com/toeirei/keymaster/ui/tui/util"
)

// *[Text] implements [form.FormElement]
var _ form.FormElement = (*Text)(nil)

type Text struct {
	Label       string
	Placeholder string
	KeyMap      TextKeyMap
	Disabled    bool

	input   textinput.Model
	focused bool
}

type TextKeyMap struct {
	Next key.Binding
}

func (k TextKeyMap) ShortHelp() []key.Binding { return []key.Binding{} }

func (k TextKeyMap) FullHelp() [][]key.Binding { return [][]key.Binding{} }

func NewText(label, placeholder string) form.FormElement {
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

func (t *Text) Focus(parentKeyMap help.KeyMap) tea.Cmd {
	t.focused = true
	return tea.Batch(
		t.input.Focus(),
		util.AnnounceKeyMapCmd(parentKeyMap, t.KeyMap),
	)
}

func (t *Text) Blur() {
	t.focused = false
	t.input.Blur()
}

func (t *Text) Get() any {
	return t.input.Value()
}

func (t *Text) Init() (tea.Cmd, form.GlobalKeyMap) {
	return nil, nil
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

func (t *Text) View(width int, eager bool) string {
	views := make([]string, 0, 2)

	// render label
	if t.Label != "" {
		labelStyle := lipgloss.NewStyle().MaxWidth(width).Foreground(lipgloss.Color("240"))
		if eager {
			labelStyle = labelStyle.Width(width)
		}
		if t.focused {
			labelStyle = labelStyle.Foreground(lipgloss.Color("205")).Bold(true)
		}
		views = append(views, labelStyle.Render(t.Label))
	}

	// render input
	t.input.Width = width - 2
	if !eager {
		t.input.Width = min(t.input.Width, max(
			len(t.Placeholder),
			len(t.Label),
			len(t.input.Value()),
		))
	}
	t.input.Placeholder = t.Placeholder
	views = append(views, t.input.View())

	return lipgloss.JoinVertical(lipgloss.Left, views...)
}

func (t *Text) Focusable() bool {
	return !t.Disabled
}

var _ form.FormElement = (*Text)(nil)
