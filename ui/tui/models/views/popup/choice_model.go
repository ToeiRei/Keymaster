// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package popupviews

import (
	"fmt"

	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/form"
	formelement "github.com/toeirei/keymaster/ui/tui/models/helpers/form/element"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/popup"
	"github.com/toeirei/keymaster/ui/tui/util"
	"github.com/toeirei/keymaster/util/slicest"
)

type ChoiceModel struct {
	form      form.Form[struct{}]
	innerSize util.Size
	size      util.Size
}

type Choice struct {
	Name string
	Cmd  tea.Cmd
}
type Choices []Choice

func OpenChoice(question string, choices Choices, width, height int) tea.Cmd {
	return popup.Open(util.ModelPointer(newChoice(question, choices, width, height)))
}

func newChoice(question string, choices Choices, width, height int) *ChoiceModel {
	rowOpts := slicest.MapI(choices, func(i int, choice Choice) form.RowOpt[struct{}] {
		return form.WithItem[struct{}]("choice_"+fmt.Sprint(i), formelement.NewButton(
			choice.Name,
			formelement.WithButtonAction(func() (tea.Cmd, form.Action) {
				return tea.Sequence(popup.Close(), choice.Cmd), form.ActionNone
			}),
		))
	})
	rowOpts = append(rowOpts, form.WithAlign[struct{}](form.Center))

	return &ChoiceModel{
		form: form.New(
			form.WithRowItem[struct{}]("choice_label", formelement.NewLabel(question)),
			form.WithRow(rowOpts...),
		),
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
	if m.size.UpdateFromMsg(msg) {
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

func (m *ChoiceModel) Focus(parentKeyMap help.KeyMap) tea.Cmd {
	return m.form.Focus(parentKeyMap )
}
func (m *ChoiceModel) Blur() {
	m.form.Blur()
}

// *[ChoiceModel] implements [util.Model]
var _ util.Model = (*ChoiceModel)(nil)
