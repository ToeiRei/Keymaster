// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package content

import (
	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/ui/tui/models/components/menu"
	"github.com/toeirei/keymaster/ui/tui/models/components/popup"
	"github.com/toeirei/keymaster/ui/tui/models/components/router"
	"github.com/toeirei/keymaster/ui/tui/models/components/stack"
	"github.com/toeirei/keymaster/ui/tui/models/views/dashboard"
	"github.com/toeirei/keymaster/ui/tui/models/views/testpopup1"
	"github.com/toeirei/keymaster/ui/tui/models/views/testview1"
	"github.com/toeirei/keymaster/ui/tui/util"
)

type Model struct {
	stack          *stack.Model
	router         *util.Model
	routerControll router.Controll
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
		menu.WithItem("test", "Tests",
			menu.WithItem("test.popup1", "Popup Test 1"),
			menu.WithItem("test.view1", "View Test 1"),
		),
		menu.WithItem("projects", "Projects",
			menu.WithItem("proj_active", "Active Projects",
				menu.WithItem("proj_a", "Project Alpha",
					menu.WithItem("a_tasks", "Task List"),
					menu.WithItem("a_milestones", "Milestones"),
				),
				menu.WithItem("proj_b", "Project Beta"),
			),
			menu.WithItem("proj_archived", "Archive"),
		),
		menu.WithItem("users", "User Management",
			menu.WithItem("u_list", "All Users"),
			menu.WithItem("u_roles", "Role Definitions",
				menu.WithItem("role_admin", "Administrators",
					menu.WithItem("perm_full", "Full Permissions"),
				),
				menu.WithItem("role_editor", "Editors"),
			),
		),
		menu.WithItem("analytics", "Analytics",
			menu.WithItem("an_sales", "Sales Reports",
				menu.WithItem("q1_sales", "Q1 Report"),
				menu.WithItem("q2_sales", "Q2 Report"),
			),
			menu.WithItem("an_traffic", "Web Traffic"),
		),
		menu.WithItem("billing", "Billing",
			menu.WithItem("bill_inv", "Invoices"),
			menu.WithItem("bill_meth", "Payment Methods"),
		),
		menu.WithItem("settings", "Settings",
			menu.WithItem("set_gen", "General"),
			menu.WithItem("set_sec", "Security",
				menu.WithItem("sec_2fa", "Two-Factor Auth",
					menu.WithItem("2fa_sms", "SMS Setup"),
					menu.WithItem("2fa_app", "Authenticator App"),
				),
			),
		),
		menu.WithItem("inventory", "Inventory with a name"),
		menu.WithItem("logistics", "Logistics",
			menu.WithItem("log_shipping", "Shipping",
				menu.WithItem("ship_int", "International",
					menu.WithItem("ship_customs", "Customs Forms"),
				),
			),
		),
		menu.WithItem("marketing", "Marketing"),
		menu.WithItem("support", "Support Tickets",
			menu.WithItem("sup_open", "Open Tickets"),
			menu.WithItem("sup_closed", "History"),
		),
		menu.WithItem("hr", "Human Resources",
			menu.WithItem("hr_payroll", "Payroll"),
			menu.WithItem("hr_benefits", "Benefits"),
		),
		menu.WithItem("legal", "Legal Compliance"),
		menu.WithItem("it_assets", "IT Assets",
			menu.WithItem("it_hw", "Hardware",
				menu.WithItem("hw_laptops", "Laptops"),
				menu.WithItem("hw_servers", "Servers"),
			),
		),
		menu.WithItem("api", "API Management",
			menu.WithItem("api_keys", "Access Keys"),
			menu.WithItem("api_docs", "Documentation"),
		),
		menu.WithItem("feedback", "User Feedback"),
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
	return m.stack.Init()
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	// handle menu messages
	if msg, ok := msg.(menu.ItemSelected); ok {
		switch msg.Id {
		case "test.popup1":
			// popup example 1
			return popup.Open(util.ModelPointer(testpopup1.New()))
		case "test.view1":
			// view example 1
			return m.routerControll.Push(util.ModelPointer(testview1.New(m.routerControll)))
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

func (m *Model) Focus(baseKeyMap help.KeyMap) tea.Cmd {
	return m.stack.Focus(baseKeyMap)
}

func (m *Model) Blur() {
	m.stack.Blur()
}

// *Model implements util.Model
var _ util.Model = (*Model)(nil)
