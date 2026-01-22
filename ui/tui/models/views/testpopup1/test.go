// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package testpopup1

import (
	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/ui/tui/models/components/popup"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/form"
	forminput "github.com/toeirei/keymaster/ui/tui/models/helpers/form/input"
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
			form.WithInput[formData]("firstname", forminput.NewText("Vorname", "Max")),
			form.WithInput[formData]("lastname", forminput.NewText("Nachname", "Mustermann")),
			form.WithInput[formData]("", forminput.NewButton("Submit", false)),
			form.WithOnSubmit(func(result formData, err error) tea.Cmd {
				return tea.Sequence(
					popup.Close(),
					func() tea.Msg { return result },
				)
			}),
		),
	}
}

func (m Model) Init() tea.Cmd {
	return m.form.Init()
}

func (m *Model) Update(msg tea.Msg) (cmd tea.Cmd) {
	m.form, cmd = m.form.Update(msg)
	return
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
