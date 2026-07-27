// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package messagepopup

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/ui/i18n"
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

func Open(severity MessageSeverity, message fmt.Stringer, cmd tea.Cmd) tea.Cmd {
	return popup.Open(util.ModelPointer(New(severity, message, cmd)))
}

func New(
	severity MessageSeverity,
	message fmt.Stringer,
	cmd tea.Cmd,
) *formpopup.Form[struct{}] {
	var title string
	switch severity {
	case Success:
		title = "popup.severity_success"
	case Info:
		title = "popup.severity_info"
	case Warning:
		title = "popup.severity_warning"
	case Error:
		title = "popup.severity_error"
	}
	return formpopup.New(form.New(
		form.WithRowItem[struct{}]("_title", formelement.NewLabel(i18n.Text(title))),
		form.WithRowItem[struct{}]("_message", formelement.NewLabel(message)),
		form.WithRowItem[struct{}]("_ok", formelement.NewButton(i18n.Text("popup.ok"), formelement.WithButtonActionSubmit(), formelement.WithButtonGlobalKeyBindings(keys.Close()))),
		form.WithOnSubmit(func(_ struct{}, _ error) (tea.Cmd, bool) { return cmd, true }),
		form.WithDefaultRowAlign[struct{}](form.Center),
	))
}
