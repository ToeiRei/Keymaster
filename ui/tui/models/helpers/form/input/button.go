// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package forminput

import (
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/form"
	"github.com/toeirei/keymaster/ui/tui/util"
)

type Button struct {
	Label    string
	Disabled bool
	KeyMap   ButtonKeyMap

	DisabledStyle lipgloss.Style
	BlurredStyle  lipgloss.Style
	FocusedStyle  lipgloss.Style

	focused bool
}

type ButtonKeyMap struct {
	Click key.Binding
}

func (k ButtonKeyMap) ShortHelp() []key.Binding { return []key.Binding{k.Click} }

func (k ButtonKeyMap) FullHelp() [][]key.Binding { return [][]key.Binding{{k.Click}} }

func NewButton(label string, disabled bool) form.FormInput {
	return &Button{
		Label: label,
		KeyMap: ButtonKeyMap{
			Click: key.NewBinding(
				key.WithKeys("enter"),
				key.WithHelp("enter", strings.ToLower(label)),
			),
		},
		DisabledStyle: lipgloss.NewStyle().
			Padding(0, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Foreground(lipgloss.Color("240")),
		BlurredStyle: lipgloss.NewStyle().
			Padding(0, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Foreground(lipgloss.Color("240")),
		FocusedStyle: lipgloss.NewStyle().
			Padding(0, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("205")).
			Foreground(lipgloss.Color("240")).
			Bold(true),
	}
}

func (b *Button) Focus(baseKeyMap help.KeyMap) tea.Cmd {
	b.focused = true
	return util.AnnounceKeyMapCmd(baseKeyMap, b.KeyMap)
}

func (b *Button) Blur() {
	b.focused = false
}

func (b *Button) Update(msg tea.Msg) (tea.Cmd, form.Action) {
	if msg, ok := msg.(tea.KeyMsg); ok && !b.Disabled && key.Matches(msg, b.KeyMap.Click) {
		return nil, form.ActionSubmit
	}
	return nil, form.ActionNone
}

func (b *Button) View(width int) string {
	if b.Disabled {
		return b.DisabledStyle.MaxWidth(width - 2).Render(b.Label)
	} else if b.focused {
		return b.FocusedStyle.MaxWidth(width - 2).Render(b.Label)
	} else {
		return b.BlurredStyle.MaxWidth(width - 2).Render(b.Label)
	}
}

// not needed
func (b *Button) Get() any      { return nil }
func (b *Button) Init() tea.Cmd { return nil }
func (b *Button) Reset()        {}
func (b *Button) Set(any)       {}

var _ form.FormInput = (*Button)(nil)
