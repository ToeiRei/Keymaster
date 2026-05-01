// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package popupviews

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/form"
	formelement "github.com/toeirei/keymaster/ui/tui/models/helpers/form/element"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/popup"
	"github.com/toeirei/keymaster/ui/tui/util"
	"github.com/toeirei/keymaster/util/slicest"
)

type Choice struct {
	Name        string
	Cmd         tea.Cmd
	KeyBindings form.GlobalKeyMap
}
type Choices []Choice

func OpenChoice(question string, choices Choices) tea.Cmd {
	return popup.Open(util.ModelPointer(newChoice(question, choices)))
}

func newChoice(question string, choices Choices) *FormModel[struct{}] {
	rowOpts := slicest.MapI(choices, func(i int, choice Choice) form.RowOpt[struct{}] {
		return form.WithItem[struct{}]("choice_"+fmt.Sprint(i), formelement.NewButton(
			choice.Name,
			formelement.WithButtonAction(func() (tea.Cmd, form.Action) {
				return tea.Sequence(popup.Close(), choice.Cmd), form.ActionNone
			}),
			formelement.WithButtonGlobalKeyBindings(choice.KeyBindings...),
		))
	})
	rowOpts = append(rowOpts, form.WithAlign[struct{}](form.Center))

	return newForm(form.New(
		form.WithRowItem[struct{}]("choice_label", formelement.NewLabel(question)),
		form.WithRow(rowOpts...),
	))
}
