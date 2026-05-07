// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package messagepopup

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/ui/tui/helpers/form"
	formelement "github.com/toeirei/keymaster/ui/tui/helpers/form/element"
	"github.com/toeirei/keymaster/ui/tui/helpers/popup"
	"github.com/toeirei/keymaster/ui/tui/popups/formpopup"
	"github.com/toeirei/keymaster/ui/tui/util"
	"github.com/toeirei/keymaster/ui/tui/util/keys"
)

const (
	Success MessageSeverity = iota
	Info
	Warning
	Error
)

type MessageSeverity int

func Open(severity MessageSeverity, message string, cmd tea.Cmd) tea.Cmd {
	return popup.Open(util.ModelPointer(New(severity, message, cmd)))
}

func New(
	severity MessageSeverity,
	message string,
	cmd tea.Cmd,
) *formpopup.Form[struct{}] {
	var title string
	switch severity {
	case Success:
		title = "SUCCESS"
	case Info:
		title = "INFO"
	case Warning:
		title = "WARNING"
	case Error:
		title = "ERROR"
	}
	return formpopup.New(form.New(
		form.WithRowItem[struct{}]("_title", formelement.NewLabel(title)),
		form.WithRowItem[struct{}]("_message", formelement.NewLabel(message)),
		form.WithRowItem[struct{}]("_ok", formelement.NewButton("Ok", formelement.WithButtonActionSubmit(), formelement.WithButtonGlobalKeyBindings(keys.Close()))),
		form.WithOnSubmit(func(_ struct{}, _ error) (tea.Cmd, bool) { return tea.Sequence(popup.Close(), cmd), true }),
		form.WithDefaultRowAlign[struct{}](form.Center),
	))
}
