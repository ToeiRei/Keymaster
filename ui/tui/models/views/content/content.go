// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package content

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/client"
	"github.com/toeirei/keymaster/ui/tui/models/components/menu"
	"github.com/toeirei/keymaster/ui/tui/models/components/router"
	"github.com/toeirei/keymaster/ui/tui/models/components/stack"
	"github.com/toeirei/keymaster/ui/tui/models/views/account"
	"github.com/toeirei/keymaster/ui/tui/models/views/dashboard"
	popupviews "github.com/toeirei/keymaster/ui/tui/models/views/popup"
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

	c := client.Client(client.NewTestUIClient())
	// create development test data
	_, _ = c.CreatePublicKey(context.Background(), "Sha-your-mom ashtdjhk-fbaskjdfhal_sdvkhaösdljhask-ödtjfb", "my-key", []string{"user:jannes", "company:none"})
	_, _ = c.CreatePublicKey(context.Background(), "Sha-420 asdjhk-fbaskjdfhal_sdvkhathrösdljhask-ödjfb", "420", []string{"user:toeirei", "company:another"})
	_, _ = c.CreatePublicKey(context.Background(), "Sha-69 asdjkhk-fbaskjdftrhhal_sdvkhaösdljhask-ödjhtfb", "69", []string{"user:somebodyelse", "company:evilgoogle"})

	c = client.NewMockClient(client.WitchMockBaseClient(c), client.WitchMockPre(func(method string, args map[string]any) {
		time.Sleep(time.Second)
	}))

	menuPtr := util.ModelPointer(menu.New(
		menu.WithItem("publickey.list", "Public Keys"),
		menu.WithItem("account.list", "Accounts"),
		menu.WithItem("test.popup.progress.spinner", "Test Progress Spinner"),
		menu.WithItem("test.popup.progress.bar", "Test Progress Bar"),
	))
	dashboardPtr := util.ModelPointer(dashboard.New(c))
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
		client:         c,
	}
}

func (m *Model) Init() tea.Cmd {
	return m.stack.Init()
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	// handle menu messages
	if msg, ok := msg.(menu.ItemSelected); ok {
		switch msg.Id {
		case "publickey.list":
			return publickey.NewCrud(m.client, m.routerControll).OpenList()
		case "account.list":
			return account.NewCrud(m.client, m.routerControll).OpenList()
		case "test.popup.progress.bar":
			return popupviews.OpenProgress(
				popupviews.ProgressBar,
				"Test Progress",
				func(pc popupviews.ProgressChan) tea.Msg {
					for i := range 100 {
						pc <- popupviews.Progress{
							Progress: float64(i+1) / 100,
							Status:   fmt.Sprintf("%d / 100", i+1),
						}
						time.Sleep(time.Second / 40)
					}
					return nil
				},
			)
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

	return tea.Sequence(cmd1, cmd2)
}

func (m *Model) View() string {
	return m.stack.View()
}

func (m *Model) Focus(parentKeyMap help.KeyMap) tea.Cmd {
	return m.stack.Focus(parentKeyMap)
}

func (m *Model) Blur() {
	m.stack.Blur()
}

// *[Model] implements [util.Model]
var _ util.Model = (*Model)(nil)
