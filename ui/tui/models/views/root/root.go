package root

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/buildvars"
	"github.com/toeirei/keymaster/ui/tui/models/components/header"
	"github.com/toeirei/keymaster/ui/tui/models/components/menu"
	"github.com/toeirei/keymaster/ui/tui/models/components/popup"
	"github.com/toeirei/keymaster/ui/tui/models/components/stack"
	windowtitle "github.com/toeirei/keymaster/ui/tui/models/helpers/title"
	"github.com/toeirei/keymaster/ui/tui/models/views/debug"
	"github.com/toeirei/keymaster/ui/tui/models/views/footer"
	"github.com/toeirei/keymaster/ui/tui/models/views/testpopup1"
	"github.com/toeirei/keymaster/ui/tui/util"
)

const title string = "Keymaster"

type Model struct {
	stack        *stack.Model
	menu         *util.Model
	footer       *util.Model
	titleHandler *windowtitle.TitleHandler
}

func New() *Model {
	_header := header.New()
	_menu := menu.New(
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
	)
	_footer := footer.New(&BaseKeyMap)

	// create model pointers for multiple references
	_menu_ptr := util.ModelPointer(_menu)
	_footer_ptr := util.ModelPointer(_footer)

	version := "unknown version"
	if len(buildvars.Version) > 0 {
		version = buildvars.Version
	}

	return &Model{
		stack: stack.New(
			stack.WithOrientation(stack.Vertical),
			stack.WithFocus(stack.Focus(1)),
			stack.WithItem(util.ModelPointer(_header), header.SizeConfig),
			stack.WithItem(
				util.ModelPointer(popup.NewInjector(
					util.ModelPointer(stack.New(
						stack.WithOrientation(stack.Horizontal),
						stack.WithFocus(stack.Focus(0)),
						stack.WithItem(_menu_ptr, menu.SizeConfig),
						stack.WithItem(util.ModelPointer(debug.New()), stack.VariableSize(1)),
					)),
				)),
				stack.VariableSize(1)),
			stack.WithItem(_footer_ptr, footer.SizeConfig),
		),
		menu:         _menu_ptr,
		footer:       _footer_ptr,
		titleHandler: windowtitle.NewHandler(fmt.Sprintf("%s %s", title, version), " | "),
	}
}

func (m Model) Init() tea.Cmd {
	titleCmd := m.titleHandler.Init()
	initCmd := m.stack.Init()
	focusCmd, keyMap := m.stack.Focus()
	keyMapCmd := util.AnnounceKeyMapCmd(keyMap)

	return tea.Sequence(titleCmd, initCmd, focusCmd, keyMapCmd)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// handle keys messages
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(msg, BaseKeyMap.Exit):
			// TODO maybe add popup
			return m, tea.Quit
		case key.Matches(msg, BaseKeyMap.Help):
			util.BorrowModelFunc(m.footer, func(_footer *footer.Model) {
				_footer.ToggleExpanded()
			})
		}

		return m, m.stack.Update(msg)
	}
	// handle menu messages
	if msg, ok := msg.(menu.ItemSelected); ok {
		switch msg.Id {
		case "test.popup1":
			// popup example
			return m, popup.Open(util.ModelPointer(testpopup1.New()))
		}
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
