// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package formelement

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/toeirei/keymaster/ui/tui/helpers/form"
	"github.com/toeirei/keymaster/ui/tui/util"
	"github.com/toeirei/keymaster/ui/tui/util/keys"
)

// *[Popup] implements [form.FormElement]
var _ form.FormElement = (*Popup[any])(nil)

type popupReturnValueMsg[T any] struct{ value T }

type Popup[T any] struct {
	Label       string
	fnOpenPopup func(returnValue func(value T) tea.Cmd) tea.Cmd
	fnToString  func(value T) string

	DisabledStyle lipgloss.Style
	BlurredStyle  lipgloss.Style
	FocusedStyle  lipgloss.Style

	value    T
	focused  bool
	Disabled bool
}

func NewPopup[T any](
	label string,
	fnOpenPopup func(returnValue func(value T) tea.Cmd) tea.Cmd,
	fnToString func(value T) string,
) form.FormElement {
	return &Popup[T]{
		Label:       label,
		fnOpenPopup: fnOpenPopup,
		fnToString:  fnToString,

		DisabledStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),
		BlurredStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),
		FocusedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true),
	}
}

func (p *Popup[T]) Focus(parentKeyMap help.KeyMap) tea.Cmd {
	p.focused = true
	return util.AnnounceKeyMapCmd(parentKeyMap, keys.KeyBindingList{keys.Open()})
}

func (p *Popup[T]) Blur() { p.focused = false }

func (p *Popup[T]) Get() any { return p.value }

func (p *Popup[T]) Init() (tea.Cmd, keys.KeyBindingList) { return nil, nil }

func (p *Popup[T]) Reset() { p.value = util.NewZero[T]() }

func (p *Popup[T]) Set(value any) {
	if value, ok := value.(T); ok {
		p.value = value
	}
}

func (p *Popup[T]) Update(msg tea.Msg) (tea.Cmd, form.Action) {
	switch msg := msg.(type) {
	case popupReturnValueMsg[T]:
		p.value = msg.value
	case tea.KeyMsg:
		if p.Disabled {
			break
		}

		switch {
		case key.Matches(msg, keys.Open()):
			return p.fnOpenPopup(
				func(value T) tea.Cmd { return util.TeaMsgToCmd(popupReturnValueMsg[T]{value}) },
			), form.ActionNone
		case key.Matches(msg, keys.DownArrow(), keys.RightArrow()):
			return nil, form.ActionNext
		case key.Matches(msg, keys.UpArrow(), keys.LeftArrow()):
			return nil, form.ActionPrev
		}
	}

	return nil, form.ActionNone
}

func (p *Popup[T]) View(width int, eager bool) string {
	var style lipgloss.Style
	if p.Disabled {
		style = p.DisabledStyle
	} else if p.focused {
		style = p.FocusedStyle
	} else {
		style = p.BlurredStyle
	}

	label := ansi.Truncate(p.Label, width, "…")
	content := ansi.Truncate(p.fnToString(p.value), width-4, "…")

	return lipgloss.JoinVertical(lipgloss.Left, style.Render(label), "[ "+content+" ]")
}

func (p *Popup[T]) Focusable() bool { return !p.Disabled }
