// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package formelement

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/ui/tui/helpers/form"
	"github.com/toeirei/keymaster/ui/tui/util"
	"github.com/toeirei/keymaster/ui/tui/util/keys"
)

// *[TextArea] implements [form.FormElement]
var _ form.FormElement = (*TextArea)(nil)

type TextArea struct {
	Label       string
	Placeholder string
	Disabled    bool

	textarea textarea.Model
	focused  bool

	minHeight int
	maxHeight int
}

func NewTextarea(label, placeholder string, minHeight, maxHeight int) form.FormElement {
	ta := textarea.New()
	ta.ShowLineNumbers = false
	ta.MaxWidth = 0
	ta.MaxHeight = 0

	return &TextArea{
		Label:       label,
		Placeholder: placeholder,
		textarea:    ta,
		minHeight:   minHeight,
		maxHeight:   maxHeight,
	}
}

func (t *TextArea) Focus(parentKeyMap help.KeyMap) tea.Cmd {
	t.focused = true
	return tea.Batch(
		t.textarea.Focus(),
		util.AnnounceKeyMapCmd(parentKeyMap),
	)
}

func (t *TextArea) Blur() {
	t.focused = false
	t.textarea.Blur()
}

func (t *TextArea) Get() any {
	return t.textarea.Value()
}

func (t *TextArea) Init() (tea.Cmd, keys.KeyBindingList) {
	return nil, nil
}

func (t *TextArea) Reset() {
	t.textarea.SetValue("")
}

func (t *TextArea) Set(value any) {
	if value, ok := value.(string); ok {
		t.textarea.SetValue(value)
	}
}

func (t *TextArea) Update(msg tea.Msg) (tea.Cmd, form.Action) {
	return util.UpdateTeaModelInplace(msg, &t.textarea), form.ActionNone
}

func (t *TextArea) View(width int, eager bool) string {
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
	t.textarea.SetWidth(width - 2)
	t.textarea.SetHeight(util.Clamp(
		t.minHeight,
		t.textarea.LineCount(),
		t.maxHeight,
	))
	// if !eager {
	// 	t.textarea.Width() = min(t.textarea.Width, max(
	// 		len(t.Placeholder),
	// 		len(t.Label),
	// 		len(t.textarea.Value()),
	// 	))
	// }
	t.textarea.Placeholder = t.Placeholder
	views = append(views, t.textarea.View())

	return lipgloss.JoinVertical(lipgloss.Left, views...)
}

func (t *TextArea) Focusable() bool {
	return !t.Disabled
}
