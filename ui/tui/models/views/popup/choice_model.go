// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package popupviews

import (
	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/form"
	formelement "github.com/toeirei/keymaster/ui/tui/models/helpers/form/element"
	"github.com/toeirei/keymaster/ui/tui/util"
)

type ChoiceModel struct {
	form      form.Form[struct{}]
	innerSize util.Size
	size      util.Size
}

type Choices map[string]func() tea.Cmd

func NewChoice(question string, choices Choices, width, height int) *ChoiceModel {
	opts := make([]form.NewOpt[struct{}], len(choices)+1)

	opts = append(opts, form.WithElement[struct{}]("", formelement.NewLabel(question)))

	for name, callback := range choices {
		opts = append(opts, form.WithElementInline[struct{}]("", formelement.NewButton(
			name,
			false,
			func() (tea.Cmd, form.Action) {
				return callback(), form.ActionNone
			},
		)))
	}

	form := form.New(opts...)

	return &ChoiceModel{
		form: form,
		innerSize: util.Size{
			Width:  width,
			Height: height,
		},
	}
}

func (m ChoiceModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m *ChoiceModel) Update(msg tea.Msg) tea.Cmd {
	if m.size.Update(msg) {
		size := util.Size{
			Width:  min(m.innerSize.Width, m.size.Width),
			Height: min(m.innerSize.Height, m.size.Height),
		}
		return m.form.Update(size.ToMsg())
	}
	return m.form.Update(msg)
}

func (m ChoiceModel) View() string {
	// TODO only for testing... size of form needs to be made non greedy
	return lipgloss.NewStyle().MaxWidth(40).Render(m.form.View())
	// return m.form.View()
}

func (m *ChoiceModel) Focus(baseKeyMap help.KeyMap) tea.Cmd {
	return m.form.Focus(baseKeyMap)
}
func (m *ChoiceModel) Blur() {
	m.form.Blur()
}

// *ChoiceModel implements util.Model
var _ util.Model = (*ChoiceModel)(nil)
