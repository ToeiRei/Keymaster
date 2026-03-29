// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package root

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/buildvars"
	"github.com/toeirei/keymaster/ui/tui/models/components/header"
	"github.com/toeirei/keymaster/ui/tui/models/components/stack"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/form"
	formelement "github.com/toeirei/keymaster/ui/tui/models/helpers/form/element"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/popup"
	windowtitle "github.com/toeirei/keymaster/ui/tui/models/helpers/title"
	"github.com/toeirei/keymaster/ui/tui/models/views/content"
	"github.com/toeirei/keymaster/ui/tui/models/views/footer"
	popupviews "github.com/toeirei/keymaster/ui/tui/models/views/popup"
	"github.com/toeirei/keymaster/ui/tui/util"
)

const title string = "Keymaster"

type Model struct {
	stack        *stack.Model
	footer       *util.Model
	titleHandler *windowtitle.TitleHandler
}

func New() *Model {
	headerPtr := util.ModelPointer(header.New())
	footerPtr := util.ModelPointer(footer.New(&BaseKeyMap))

	version := "unknown version"
	if len(buildvars.Version) > 0 {
		version = buildvars.Version
	}
	titleHandler := windowtitle.NewHandler(fmt.Sprintf("%s %s", title, version), " | ")

	return &Model{
		stack: stack.New(
			stack.WithOrientation(stack.Vertical),
			stack.WithItem(headerPtr, header.SizeConfig),
			stack.WithFocusNext(),
			stack.WithItem(
				util.ModelPointer(popup.NewInjector(
					util.ModelPointer(content.New()),
				)),
				stack.VariableSize(1)),
			stack.WithItem(footerPtr, footer.SizeConfig),
		),
		footer:       footerPtr,
		titleHandler: titleHandler,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Sequence(
		m.titleHandler.Init(),
		m.stack.Init(),
		m.stack.Focus(util.EmptyKeyMap{}),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// handle keys messages
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(msg, BaseKeyMap.Exit):
			return m, popup.Open(util.ModelPointer(popupviews.NewForm(
				form.New(
					form.WithRow(
						form.WithElement[struct{}]("", formelement.NewLabel("Do you realy wanna exit KeyMaster?")),
					),
					form.WithRow(
						form.WithAlign[struct{}](form.Center),
						form.WithElement[struct{}]("", formelement.NewButton("Cancel", false, func() (tea.Cmd, form.Action) { return popup.Close(), form.ActionNone })),
						form.WithElement[struct{}]("", formelement.NewButton("Exit", false, func() (tea.Cmd, form.Action) { return tea.Quit, form.ActionNone })),
					),
				),
				40, 40,
			)))
			// TODO maybe add popup
			// return m, tea.Quit
		case key.Matches(msg, BaseKeyMap.Help):
			util.BorrowModelFunc(m.footer, func(_footer *footer.Model) {
				_footer.ToggleExpanded()
			})
		}

		return m, m.stack.Update(msg)
	}
	// handle window title messages
	if cmd := m.titleHandler.Handle(msg); cmd != nil {
		return m, cmd
	}
	// handle other messages
	return m, m.stack.Update(msg)
}

func (m Model) View() string {
	return m.stack.View()
}

// *Model implements util.Model
var _ tea.Model = (*Model)(nil)
