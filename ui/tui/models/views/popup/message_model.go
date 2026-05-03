// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package popupviews

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/form"
	formelement "github.com/toeirei/keymaster/ui/tui/models/helpers/form/element"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/popup"
	"github.com/toeirei/keymaster/ui/tui/util"
	"github.com/toeirei/keymaster/ui/tui/util/keys"
)

const (
	MessageSuccess MessageSeverity = iota
	MessageInfo
	MessageWarning
	MessageError
)

type MessageSeverity int

func OpenMessage(severity MessageSeverity, message string, cmd tea.Cmd) tea.Cmd {
	return popup.Open(util.ModelPointer(newMessage(severity, message, cmd)))
}

func newMessage(
	severity MessageSeverity,
	message string,
	cmd tea.Cmd,
) *FormModel[struct{}] {
	var title string
	switch severity {
	case MessageSuccess:
		title = "SUCCESS"
	case MessageInfo:
		title = "INFO"
	case MessageWarning:
		title = "WARNING"
	case MessageError:
		title = "ERROR"
	}
	return newForm(form.New(
		form.WithRowItem[struct{}]("_title", formelement.NewLabel(title)),
		form.WithRowItem[struct{}]("_message", formelement.NewLabel(message)),
		form.WithRowItem[struct{}]("_ok", formelement.NewButton("Ok", formelement.WithButtonActionSubmit(), formelement.WithButtonGlobalKeyBindings(keys.Close()))),
		form.WithOnSubmit(func(_ struct{}, _ error) (tea.Cmd, bool) { return tea.Sequence(popup.Close(), cmd), true }),
		form.WithDefaultRowAlign[struct{}](form.Center),
	))
}
