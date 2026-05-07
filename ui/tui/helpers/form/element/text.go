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
	"github.com/toeirei/keymaster/ui/tui/helpers/form"
	"github.com/toeirei/keymaster/ui/tui/util"
	"github.com/toeirei/keymaster/ui/tui/util/keys"
)

// *[Text] implements [form.FormElement]
var _ form.FormElement = (*Text)(nil)

type Text struct {
	Label       string
	Placeholder string
	Disabled    bool

	input   textinput.Model
	focused bool
}

type TextOption func(*Text)

func WithTextDisable() TextOption {
	return func(t *Text) { t.Disabled = true }
}

func NewText(label, placeholder string, opts ...TextOption) form.FormElement {
	text := &Text{
		Label:       label,
		Placeholder: placeholder,
		input:       textinput.New(),
	}
	for _, opt := range opts {
		opt(text)
	}
	return text
}

func (t *Text) Focus(parentKeyMap help.KeyMap) tea.Cmd {
	t.focused = true
	return tea.Batch(
		t.input.Focus(),
		util.AnnounceKeyMapCmd(parentKeyMap),
	)
}

func (t *Text) Blur() {
	t.focused = false
	t.input.Blur()
}

func (t *Text) Get() any {
	return t.input.Value()
}

func (t *Text) Init() (tea.Cmd, keys.KeyBindingList) {
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
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(msg, keys.NextEnter()), key.Matches(msg, keys.DownArrow()):
			return nil, form.ActionNext
		case key.Matches(msg, keys.UpArrow()):
			return nil, form.ActionPrev
		}
	}

	if t.Disabled {
		return nil, form.ActionNone
	}

	return util.UpdateTeaModelInplace(msg, &t.input), form.ActionNone
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
	t.input.Width = width - 2 - 1 // -1 because bubbles doesn't think a cursor takes up any space
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
