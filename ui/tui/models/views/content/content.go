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
	"github.com/toeirei/keymaster/client/mock"
	"github.com/toeirei/keymaster/client/testui"
	"github.com/toeirei/keymaster/tags"
	"github.com/toeirei/keymaster/ui/tui/models/components/menu"
	"github.com/toeirei/keymaster/ui/tui/models/components/router"
	"github.com/toeirei/keymaster/ui/tui/models/components/stack"
	"github.com/toeirei/keymaster/ui/tui/models/helpers/deploy"
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

	c := client.Client(testui.NewClient())

	// test accounts
	_, _ = c.CreateAccount(context.Background(), "root", "1.2.3.4", 22, "ssh", "password123")
	_, _ = c.CreateAccount(context.Background(), "user", "1.2.3.4", 22, "ssh", "password123")
	_, _ = c.CreateAccount(context.Background(), "srv", "10.0.0.1", 22, "ssh", "password123")
	_, _ = c.CreateAccount(context.Background(), "mark", "1.2.3.4", 22, "ssh", "password123")
	_, _ = c.CreateAccount(context.Background(), "admin", "10.20.0.1", 222, "cisco", "password123")
	// test publicKeys
	_, _ = c.CreatePublicKey(context.Background(), "Sha-your-mom ashtdjhk-fbaskjdfhal_sdvkhaösdljhask-zdpjwb", "my-key", tags.Tags{"user:jannes", "company:work", "server-ci"})
	_, _ = c.CreatePublicKey(context.Background(), "Sha-your-mom ashtdjhk-fbaskjdfhal_sdvkhaösdljhask-öutyfb", "my-key", tags.Tags{"user:jannes", "company:none"})
	_, _ = c.CreatePublicKey(context.Background(), "Sha-420 asdjhk-fbaskdasral_jklkhathrösdljhask-fdjfb", "419", tags.Tags{"user:toeirei", "company:big_money"})
	_, _ = c.CreatePublicKey(context.Background(), "Sha-420 asdjhk-fbaskjdfhal_sdvtzuthrösdljhaha-ögjfb", "420", tags.Tags{"user:toeirei", "company:work", "server-ci"})
	_, _ = c.CreatePublicKey(context.Background(), "Sha-420 asdjhk-fbaskjterhl_sdvkhaghdjfdljhask-ödhfb", "421", tags.Tags{"user:toeirei", "company:none"})
	_, _ = c.CreatePublicKey(context.Background(), "Sha-69 asdjkhk-fbdfhtdftrhhal_sdvkhaösu656zsk-ödjhtfb", "69", tags.Tags{"user:somebodyelse", "company:evilgoogle", "server-ci"})
	// test links
	_, _ = c.CreateLink(context.Background(), 1, "(user:jannes | user:toeirei) & !company:work", time.Now().Add(time.Hour))
	_, _ = c.CreateLink(context.Background(), 2, "!user:somebodyelse", time.Now().Add(time.Hour))
	_, _ = c.CreateLink(context.Background(), 3, "server-ci", time.Now().Add(time.Hour))
	_, _ = c.CreateLink(context.Background(), 4, "company:evilgoogle", time.Now().Add(time.Hour))
	_, _ = c.CreateLink(context.Background(), 5, "company:work", time.Now().Add(time.Hour))
	_, _ = c.CreateLink(context.Background(), 5, "company:big_money", time.Now())

	c = mock.NewClient(mock.WitchBaseClient(c), mock.WitchPre(func(method string, args map[string]any) {
		time.Sleep(time.Millisecond * 200)
	}))

	menuPtr := util.ModelPointer(menu.New(
		menu.WithItem("publickey.list", "Public Keys"),
		menu.WithItem("account.list", "Accounts"),
		menu.WithItem("deploy", "Deploy",
			menu.WithItem("deploy.dirty", "Deploy dirty"),
			menu.WithItem("deploy.all", "Deploy all"),
			menu.WithItem("deploy.verify", "Verify all"),
		),
		menu.WithItem("", "Test",
			menu.WithItem("", "Popup",
				menu.WithItem("test.popup.progress.spinner", "Progress Spinner"),
				menu.WithItem("test.popup.progress.bar", "Progress Bar"),
			),
		),
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

		case "deploy.dirty":
			return deploy.DeployDirty(context.Background(), m.client)

		case "deploy.all":
			return deploy.DeployAll(context.Background(), m.client)

		case "deploy.verify":
			return deploy.VerifyAll(context.Background(), m.client)

		case "test.popup.progress.spinner":
			return popupviews.OpenProgress(
				popupviews.ProgressSpinner,
				"Test Progress Spinner",
				func(_ popupviews.ProgressChan) tea.Cmd {
					time.Sleep(time.Second * 2)
					return nil
				},
			)

		case "test.popup.progress.bar":
			return popupviews.OpenProgress(
				popupviews.ProgressBar,
				"Test Progress Bar",
				func(pc popupviews.ProgressChan) tea.Cmd {
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
