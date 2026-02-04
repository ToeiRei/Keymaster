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
	"github.com/toeirei/keymaster/ui/tui/models/views/debug"
	"github.com/toeirei/keymaster/ui/tui/models/views/testpopup1"
	"github.com/toeirei/keymaster/ui/tui/util"
)

type Model struct {
	stack          *stack.Model
	routerControll router.Controll
}

func New() *Model {
	// stack {
	//   router {
	// 	   menu
	// 	   debug // TODO replace with dashboard later
	// 	 }
	// }

	_menu := util.ModelPointer(menu.New(
		menu.WithItem("dashboard", "Dashboard"),
		menu.WithItem("test", "Tests",
			menu.WithItem("test.popup1", "Popup Test 1"),
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
	_debug := util.ModelPointer(debug.New())
	_router, routerControll := router.New(_debug)
	_stack := stack.New(
		stack.WithOrientation(stack.Horizontal),
		stack.WithFocusNext(),
		stack.WithItem(_menu, menu.SizeConfig),
		// TODO replace with dashboard when ready
		stack.WithItem(util.ModelPointer(_router), stack.VariableSize(1)),
	)

	return &Model{
		stack:          _stack,
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
			// popup example
			return popup.Open(util.ModelPointer(testpopup1.New()))
		}
	}

	// pass other messages to stack
	return m.stack.Update(msg)
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
