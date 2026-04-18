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
	"github.com/toeirei/keymaster/ui/tui/models/helpers/popup"
	"github.com/toeirei/keymaster/ui/tui/util"
)

const (
	MessageInfo MessageSeverity = iota
	MessageWarning
	MessageError
)

type MessageSeverity int

type MessageModel struct {
	form form.Form[struct{}]
	size util.Size
}

func OpenMessage(severity MessageSeverity, message string, cmd tea.Cmd) tea.Cmd {
	return popup.Open(util.ModelPointer(newMessage(severity, message, cmd)))
}

func newMessage(
	severity MessageSeverity,
	message string,
	cmd tea.Cmd,
) *MessageModel {
	switch severity {
	case MessageInfo:
		message = "INFO: " + message
	case MessageWarning:
		message = "WARNING: " + message
	case MessageError:
		message = "ERROR: " + message
	}
	return &MessageModel{
		form: form.New(
			form.WithRow(
				form.WithItem[struct{}]("_message", formelement.NewLabel(message)),
				form.WithItem[struct{}]("_ok", formelement.NewButton("Ok", formelement.WithButtonActionSubmit())),
			),
			form.WithOnSubmit(func(_ struct{}, _ error) tea.Cmd {
				return tea.Sequence(popup.Close(), cmd)
			}),
		),
	}
}

func (m MessageModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m *MessageModel) Update(msg tea.Msg) tea.Cmd {
	if m.size.UpdateFromMsg(msg) {
		size := util.Size{
			Width:  util.Clamp(6, m.size.Width/2, m.size.Width),
			Height: util.Clamp(7, m.size.Height/2, m.size.Height),
		}
		return m.form.Update(size.ToMsg())
	}
	return m.form.Update(msg)
}

func (m MessageModel) View() string {
	return lipgloss.NewStyle().
		MaxWidth(m.size.Width).
		MaxHeight(m.size.Height).
		Render(m.form.View())
}

func (m *MessageModel) Focus(parentKeyMap help.KeyMap) tea.Cmd { return m.form.Focus(parentKeyMap) }

func (m *MessageModel) Blur() { m.form.Blur() }

// *[MessageModel] implements [util.Model]
var _ util.Model = (*MessageModel)(nil)
