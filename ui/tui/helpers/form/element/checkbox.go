// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package formelement

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/ui/tui/helpers/form"
	"github.com/toeirei/keymaster/ui/tui/util"
	"github.com/toeirei/keymaster/ui/tui/util/keys"
)

// *[Checkbox] implements [form.FormElement]
var _ form.FormElement = (*Checkbox)(nil)

type Checkbox struct {
	Label    string
	Disabled bool
	KeyMap   CheckboxKeyMap

	checked bool
	focused bool
}

type CheckboxKeyMap struct {
	Toggle key.Binding
}

func (k CheckboxKeyMap) ShortHelp() []key.Binding { return []key.Binding{k.Toggle} }

func (k CheckboxKeyMap) FullHelp() [][]key.Binding { return [][]key.Binding{{k.Toggle}} }

type CheckboxOption func(*Checkbox)

func WithCheckboxDisable() CheckboxOption {
	return func(c *Checkbox) { c.Disabled = true }
}

func NewCheckbox(label string, opts ...CheckboxOption) form.FormElement {
	checkbox := &Checkbox{
		Label: label,
		KeyMap: CheckboxKeyMap{
			Toggle: key.NewBinding(
				key.WithKeys(" ", "enter"),
				key.WithHelp("space", "toggle"),
			),
		},
	}
	for _, opt := range opts {
		opt(checkbox)
	}
	return checkbox
}

func (c *Checkbox) Focus(parentKeyMap help.KeyMap) tea.Cmd {
	c.focused = true
	return util.AnnounceKeyMapCmd(parentKeyMap, c.KeyMap)
}

func (c *Checkbox) Blur() {
	c.focused = false
}

func (c *Checkbox) Get() any {
	return c.checked
}

func (c *Checkbox) Init() (tea.Cmd, keys.KeyBindingList) {
	return nil, nil
}

func (c *Checkbox) Reset() {
	c.checked = false
}

func (c *Checkbox) Set(value any) {
	if value, ok := value.(bool); ok {
		c.checked = value
	}
}

func (c *Checkbox) Update(msg tea.Msg) (tea.Cmd, form.Action) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch {
		case !c.Disabled && key.Matches(msg, c.KeyMap.Toggle):
			c.checked = !c.checked
			return nil, form.ActionNone
		case key.Matches(msg, keys.DownArrow()):
			return nil, form.ActionNext
		case key.Matches(msg, keys.UpArrow()):
			return nil, form.ActionPrev
		}
	}

	return nil, form.ActionNone
}

func (c *Checkbox) View(width int, eager bool) string {
	box := "[ ]"
	if c.checked {
		box = "[x]"
	}

	style := lipgloss.NewStyle().MaxWidth(width).Foreground(lipgloss.Color("240"))
	if eager {
		style = style.Width(width)
	}
	if c.focused {
		style = style.Foreground(lipgloss.Color("205")).Bold(true)
	}
	if c.Disabled {
		style = style.Foreground(lipgloss.Color("240"))
	}

	content := box
	if c.Label != "" {
		content = box + " " + c.Label
	}

	return style.Render(content)
}

func (c *Checkbox) Focusable() bool {
	return !c.Disabled
}
