// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package content

import (
	"context"

	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/client"
	"github.com/toeirei/keymaster/ui/tui/models/components/menu"
	"github.com/toeirei/keymaster/ui/tui/models/components/router"
	"github.com/toeirei/keymaster/ui/tui/models/components/stack"
	"github.com/toeirei/keymaster/ui/tui/models/views/dashboard"
	"github.com/toeirei/keymaster/ui/tui/models/views/publickey"
	"github.com/toeirei/keymaster/ui/tui/util"
)

type Model struct {
	stack          *stack.Model
	router         *util.Model
	routerControll router.Controll
	client         client.Client
}

func New() *Model {
	// stack {
	// 	 menu
	//   router {
	// 	   dashboard
	// 	 }
	// }

	menuPtr := util.ModelPointer(menu.New(
		menu.WithItem("dashboard", "Dashboard"),
		menu.WithItem("publickey.list", "Public Keys"),
	))
	dashboardPtr := util.ModelPointer(dashboard.New())
	routerModel, routerControll := router.New(dashboardPtr)
	routerPtr := util.ModelPointer(routerModel)
	stackModel := stack.New(
		stack.WithOrientation(stack.Horizontal),
		stack.WithFocusNext(),
		stack.WithItem(menuPtr, menu.SizeConfig),
		stack.WithItem(routerPtr, stack.VariableSize(1)),
	)

	return &Model{
		stack:          stackModel,
		router:         routerPtr,
		routerControll: routerControll,
	}
}

func (m *Model) Init() tea.Cmd {
	m.client = client.NewTestUIClient()

	// create development test data
	_, _ = m.client.CreatePublicKey(context.Background(), "Sha-your-mom ashtdjhk-fbaskjdfhal_sdvkhaösdljhask-ödtjfb", "my-key", []string{"user:jannes", "company:none"})
	_, _ = m.client.CreatePublicKey(context.Background(), "Sha-420 asdjhk-fbaskjdfhal_sdvkhathrösdljhask-ödjfb", "420", []string{"user:toeirei", "company:another"})
	_, _ = m.client.CreatePublicKey(context.Background(), "Sha-69 asdjkhk-fbaskjdftrhhal_sdvkhaösdljhask-ödjhtfb", "69", []string{"user:somebodyelse", "company:evilgoogle"})

	return m.stack.Init()
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	// handle menu messages
	if msg, ok := msg.(menu.ItemSelected); ok {
		switch msg.Id {
		// case "test.popup1":
		// 	// popup example 1
		// 	return popup.Open(util.ModelPointer(testpopup1.New()))
		// case "test.view1":
		// 	// view example 1
		// 	return m.routerControll.Push(util.ModelPointer(testview1.New(m.routerControll)))
		// }
		case "publickey.list":
			return m.routerControll.Push(util.ModelPointer(publickey.NewList(m.client, m.routerControll)))
		}
	}

	// pass other messages to stack
	cmd1 := m.stack.Update(msg)

	// adjust the stacks focus if the potentially router changed
	var cmd2 tea.Cmd
	if router.IsRouterMsg(msg) {
		util.BorrowModelFunc(m.router, func(r *router.Model) {
			cmd2 = m.stack.SetFocus(min(len(r.GetStack())-1, 1))
		})
	}

	return tea.Batch(cmd1, cmd2)
}

func (m *Model) View() string {
	return m.stack.View()
}

func (m *Model) Focus(parentKeyMap help.KeyMap) tea.Cmd {
	return m.stack.Focus(parentKeyMap )
}

func (m *Model) Blur() {
	m.stack.Blur()
}

// *[Model] implements [util.Model]
var _ util.Model = (*Model)(nil)
