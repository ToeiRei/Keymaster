// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package formelement

import (
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/form"
	"github.com/toeirei/keymaster/ui/tui/util"
	"github.com/toeirei/keymaster/ui/tui/util/keys"
)

// *[Button] implements [form.FormElement]
var _ form.FormElement = (*Button)(nil)

type Button struct {
	Label    string
	Disabled bool
	KeyMap   ButtonKeyMap
	Action   func() (tea.Cmd, form.Action)

	DisabledStyle lipgloss.Style
	BlurredStyle  lipgloss.Style
	FocusedStyle  lipgloss.Style

	globalKeyBindings form.GlobalKeyMap
	focused           bool
}

type ButtonKeyMap struct {
	Click key.Binding
}

func (k ButtonKeyMap) ShortHelp() []key.Binding { return []key.Binding{k.Click} }

func (k ButtonKeyMap) FullHelp() [][]key.Binding { return [][]key.Binding{{k.Click}} }

type ButtonOpt = func(button *Button)

func WithButtonDisabled() ButtonOpt {
	return func(button *Button) { button.Disabled = true }
}

func WithButtonActionSubmit() ButtonOpt {
	return func(button *Button) { button.Action = func() (tea.Cmd, form.Action) { return nil, form.ActionSubmit } }
}

func WithButtonActionCancel() ButtonOpt {
	return func(button *Button) { button.Action = func() (tea.Cmd, form.Action) { return nil, form.ActionCancel } }
}

func WithButtonActionReset() ButtonOpt {
	return func(button *Button) { button.Action = func() (tea.Cmd, form.Action) { return nil, form.ActionReset } }
}

func WithButtonAction(action func() (tea.Cmd, form.Action)) ButtonOpt {
	return func(button *Button) { button.Action = action }
}

func WithButtonGlobalKeyBindings(bindings ...key.Binding) ButtonOpt {
	return func(button *Button) { button.globalKeyBindings = append(button.globalKeyBindings, bindings...) }
}

func NewButton(label string, opts ...ButtonOpt) form.FormElement {
	button := &Button{
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
	for _, opt := range opts {
		opt(button)
	}
	return button
}

func (b *Button) Focus(parentKeyMap help.KeyMap) tea.Cmd {
	b.focused = true
	return util.AnnounceKeyMapCmd(parentKeyMap, b.KeyMap)
}

func (b *Button) Blur() {
	b.focused = false
}

func (b *Button) Update(msg tea.Msg) (tea.Cmd, form.Action) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch {
		// not disabled
		// OnClick not nil
		// key matches click or global binding
		case !b.Disabled && b.Action != nil && (key.Matches(msg, b.KeyMap.Click) || key.Matches(msg, b.globalKeyBindings...)):
			return b.Action()
		case key.Matches(msg, keys.Down(), keys.Right()):
			return nil, form.ActionNext
		case key.Matches(msg, keys.Up(), keys.Left()):
			return nil, form.ActionPrev
		}
	}

	return nil, form.ActionNone
}

func (b *Button) View(width int, eager bool) string {
	var style lipgloss.Style
	if b.Disabled {
		style = b.DisabledStyle
	} else if b.focused {
		style = b.FocusedStyle
	} else {
		style = b.BlurredStyle
	}

	style = style.MaxWidth(width)
	content := b.Label
	if eager {
		style = style.Width(width - 2)
		content = lipgloss.PlaceHorizontal(width-6, lipgloss.Center, b.Label)
	}

	return style.Render(content)
}

func (b *Button) Focusable() bool {
	return !b.Disabled
}

func (b *Button) Init() (tea.Cmd, form.GlobalKeyMap) {
	return nil, b.globalKeyBindings
}

// not needed
func (b *Button) Get() any { return nil }
func (b *Button) Reset()   {}
func (b *Button) Set(any)  {}
