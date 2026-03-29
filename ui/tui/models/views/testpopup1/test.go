// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package testpopup1

import (
	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/form"
	formelement "github.com/toeirei/keymaster/ui/tui/models/helpers/form/element"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/popup"
	"github.com/toeirei/keymaster/ui/tui/util"
)

type formData struct {
	Firstname string `mapstructure:"firstname"`
	Lastname  string `mapstructure:"lastname"`
}

type Model struct {
	form form.Form[formData]
}

func New() *Model {
	return &Model{
		form: form.New(
			form.WithSingleElementRow[formData]("firstname", formelement.NewText("Vorname", "Max")),
			form.WithSingleElementRow[formData]("lastname", formelement.NewText("Nachname", "Mustermann")),
			form.WithRow(
				form.WithElement[formData]("", formelement.NewButton(
					"Cancel",
					false,
					func() (tea.Cmd, form.Action) { return nil, form.ActionCancel },
				)),
				form.WithElement[formData]("", formelement.NewButton(
					"Submit",
					false,
					func() (tea.Cmd, form.Action) { return nil, form.ActionSubmit },
				)),
			),
			form.WithOnSubmit(func(result formData, err error) tea.Cmd {
				return tea.Sequence(
					popup.Close(),
					func() tea.Msg { return result },
				)
			}),
			form.WithOnCancel[formData](func() tea.Cmd {
				return popup.Close()
			}),
		),
	}
}

func (m Model) Init() tea.Cmd {
	return m.form.Init()
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	return m.form.Update(msg)
}

func (m Model) View() string {
	// TODO only for testing... size of form needs to be made non greedy
	return lipgloss.NewStyle().MaxWidth(40).Render(m.form.View())
	// return m.form.View()
}

func (m *Model) Focus(baseKeyMap help.KeyMap) tea.Cmd {
	return m.form.Focus(baseKeyMap)
}
func (m *Model) Blur() {
	m.form.Blur()
}

// *Model implements util.Model
var _ util.Model = (*Model)(nil)
