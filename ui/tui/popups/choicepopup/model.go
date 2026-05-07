// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package choicepopup

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/ui/tui/helpers/form"
	formelement "github.com/toeirei/keymaster/ui/tui/helpers/form/element"
	"github.com/toeirei/keymaster/ui/tui/helpers/popup"
	"github.com/toeirei/keymaster/ui/tui/popups/formpopup"
	"github.com/toeirei/keymaster/ui/tui/util"
	"github.com/toeirei/keymaster/ui/tui/util/keys"
	"github.com/toeirei/keymaster/util/slicest"
)

type Choice struct {
	Name        string
	Cmd         tea.Cmd
	KeyBindings keys.KeyBindingList
}
type Choices []Choice

func Open(question string, choices Choices) tea.Cmd {
	return popup.Open(util.ModelPointer(New(question, choices)))
}

func New(question string, choices Choices) *formpopup.Form[struct{}] {
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

	return formpopup.New(form.New(
		form.WithRowItem[struct{}]("choice_label", formelement.NewLabel(question)),
		form.WithRow(rowOpts...),
	))
}
