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
	MessageInfo MessageSeverity = iota
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
	switch severity {
	case MessageInfo:
		message = "INFO: " + message
	case MessageWarning:
		message = "WARNING: " + message
	case MessageError:
		message = "ERROR: " + message
	}
	return newForm(form.New(
		form.WithRowItem[struct{}]("_message", formelement.NewLabel(message)),
		form.WithRowItem(
			"_ok",
			formelement.NewButton("Ok",
				formelement.WithButtonActionSubmit(),
				formelement.WithButtonGlobalKeyBindings(keys.Exit()),
			),
			form.WithAlign[struct{}](form.Center),
		),
		form.WithOnSubmit(func(_ struct{}, _ error) tea.Cmd {
			return tea.Sequence(popup.Close(), cmd)
		}),
	))
}
