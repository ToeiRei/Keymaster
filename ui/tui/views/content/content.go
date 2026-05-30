// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package content

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/client"
	"github.com/toeirei/keymaster/i18n"
	"github.com/toeirei/keymaster/ui/tui/components/menu"
	"github.com/toeirei/keymaster/ui/tui/components/router"
	"github.com/toeirei/keymaster/ui/tui/components/stack"
	"github.com/toeirei/keymaster/ui/tui/helpers/deploy"
	"github.com/toeirei/keymaster/ui/tui/helpers/tablecontroll"
	"github.com/toeirei/keymaster/ui/tui/popups/messagepopup"
	"github.com/toeirei/keymaster/ui/tui/popups/progresspopup"
	"github.com/toeirei/keymaster/ui/tui/popups/selectpopup"
	"github.com/toeirei/keymaster/ui/tui/util"
	"github.com/toeirei/keymaster/ui/tui/views/account"
	"github.com/toeirei/keymaster/ui/tui/views/dashboard"
	"github.com/toeirei/keymaster/ui/tui/views/publickey"
	"github.com/toeirei/keymaster/util/slicest"
)

type Model struct {
	stack          *stack.Model
	router         *util.Model
	routerControll router.Controll
	client         client.Client
}

func New(c client.Client) *Model {
	// stack {
	// 	 menu
	//   router {
	// 	   dashboard
	// 	 }
	// }

	menuPtr := util.ModelPointer(menu.New(
		menu.WithItem("dashboard.show", i18n.T("menu.dashboard")),
		menu.WithItem("publickey.list", "Public Keys"),
		menu.WithItem("account.list", "Accounts"),
		menu.WithItem("", "Deploy",
			menu.WithItem("deploy.dirty", "Deploy dirty"),
			menu.WithItem("deploy.all", "Deploy all"),
			menu.WithItem("deploy.verify", "Verify all"),
		),
		menu.WithItem("", "Test",
			menu.WithItem("", "Popup",
				menu.WithItem("test.popup.select", "Select"),
				menu.WithItem("test.popup.select_with_filter", "Select with Filter"),
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
		case "dashboard.show":
			return m.routerControll.Change(util.ModelPointer(dashboard.New(m.client)))

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

		case "test.popup.select":
			return selectpopup.Open(
				"Choose Account",
				func(ctx context.Context) ([]client.Account, error) {
					return m.client.ListAccounts(ctx)
				},
				func(r client.Account) tea.Cmd {
					return messagepopup.Open(messagepopup.Info, "You selected: "+r.String(), nil)
				},
				tablecontroll.New(tablecontroll.Columns[client.Account]{
					{Title: func() string { return "Username" }, View: func(r client.Account) string { return r.Username }},
					{Title: func() string { return "Host" }, View: func(r client.Account) string { return r.Host }},
					{Title: func() string { return "Port" }, View: func(r client.Account) string { return fmt.Sprint(r.Port) }},
					{Title: func() string { return "Deploy Method" }, View: func(r client.Account) string { return r.DeployMethod }},
				}),
			)

		case "test.popup.select_with_filter":
			return selectpopup.Open(
				"Choose Account",
				func(ctx context.Context) ([]client.Account, error) {
					return m.client.ListAccounts(ctx)
				},
				func(r client.Account) tea.Cmd {
					return messagepopup.Open(messagepopup.Info, "You selected: "+r.String(), nil)
				},
				tablecontroll.New(tablecontroll.Columns[client.Account]{
					{Title: func() string { return "Username" }, View: func(r client.Account) string { return r.Username }},
					{Title: func() string { return "Host" }, View: func(r client.Account) string { return r.Host }},
					{Title: func() string { return "Port" }, View: func(r client.Account) string { return fmt.Sprint(r.Port) }},
					{Title: func() string { return "Deploy Method" }, View: func(r client.Account) string { return r.DeployMethod }},
				}),
				selectpopup.WithFilter(func(filter string, records []client.Account) []client.Account {
					return slicest.Filter(records, func(record client.Account) bool {
						return strings.Contains(record.Username, filter) ||
							strings.Contains(record.Host, filter) ||
							strings.Contains(fmt.Sprint(record.Port), filter) ||
							strings.Contains(record.DeployMethod, filter)
					})
				}),
			)

		case "test.popup.progress.spinner":
			return progresspopup.Open(
				progresspopup.Spinner,
				"Test Progress Spinner",
				func(_ context.Context, _ progresspopup.ProgressChan) tea.Cmd {
					time.Sleep(time.Second * 2)
					return nil
				},
			)

		case "test.popup.progress.bar":
			return progresspopup.Open(
				progresspopup.Bar,
				"Test Progress Bar",
				func(_ context.Context, pc progresspopup.ProgressChan) tea.Cmd {
					for i := range 100 {
						pc <- progresspopup.Progress{
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
