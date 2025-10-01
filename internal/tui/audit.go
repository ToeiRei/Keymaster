// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package tui // import "github.com/toeirei/keymaster/internal/tui"

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/deploy"
	"github.com/toeirei/keymaster/internal/i18n"
	"github.com/toeirei/keymaster/internal/model"
)

// auditModeType represents the comparison mode for the audit.
type auditModeType int

const (
	auditModeStrict auditModeType = iota
	auditModeSerial
)

// auditState represents the current view within the audit workflow.
type auditState int

const (
	auditStateMenu auditState = iota
	auditStateSelectAccount
	auditStateSelectTag
	auditStateFleetInProgress
	auditStateInProgress
	auditStateComplete
	auditStateRemediationConfirm
	auditStateRemediationInProgress
)

// auditResultMsg is a message to signal audit completion for one account.
type auditResultMsg struct {
	account model.Account
	err     error
}

// remediationResultMsg is a message to signal remediation completion.
type remediationResultMsg struct {
	account model.Account
	result  *model.RemediationResult
	err     error
}

// auditModel represents the state of the audit view.
type auditModel struct {
	state              auditState
	mode               auditModeType
	menuCursor         int
	accountCursor      int
	tagCursor          int
	accounts           []model.Account
	accountsInFleet    []model.Account
	fleetResults       map[int]error          // map account ID to error
	driftAnalysis      map[int]*model.DriftAnalysis // map account ID to drift analysis
	selectedAccount    model.Account
	tags               []string
	status             string
	err                error
	accountFilter      string
	isFilteringAccount bool
	remediationResults map[int]*model.RemediationResult // map account ID to remediation result
	width              int                              // Terminal width for responsive layout
	height             int                              // Terminal height for responsive layout
}

func newAuditModel() auditModel {
	return auditModel{
		state:              auditStateMenu,
		mode:               auditModeStrict,
		fleetResults:       make(map[int]error),
		driftAnalysis:      make(map[int]*model.DriftAnalysis),
		remediationResults: make(map[int]*model.RemediationResult),
		width:              120, // Default width
		height:             40,  // Default height
	}
}

func (m auditModel) Init() tea.Cmd { return nil }

func (m auditModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	}

	switch m.state {
	case auditStateMenu:
		return m.updateMenu(msg)
	case auditStateSelectAccount:
		return m.updateAccountSelection(msg)
	case auditStateSelectTag:
		return m.updateSelectTag(msg)
	case auditStateFleetInProgress:
		if res, ok := msg.(auditResultMsg); ok {
			m.fleetResults[res.account.ID] = res.err
			if len(m.fleetResults) == len(m.accountsInFleet) {
				m.state = auditStateComplete
				m.status = i18n.T("audit.tui.fleet_complete")
			}
		}
		return m, nil
	case auditStateInProgress:
		if res, ok := msg.(auditResultMsg); ok {
			m.state = auditStateComplete
			if res.err != nil {
				m.err = res.err
				m.status = i18n.T("audit.tui.failed_short")
			} else {
				m.err = nil
				m.status = i18n.T("audit.tui.ok_short")
			}
		}
		return m, nil
	case auditStateComplete:
		return m.updateComplete(msg)
	case auditStateRemediationConfirm:
		return m.updateRemediationConfirm(msg)
	case auditStateRemediationInProgress:
		if res, ok := msg.(remediationResultMsg); ok {
			m.remediationResults[res.account.ID] = res.result
			// If remediation was successful, remove the drift from fleetResults
			if res.result != nil && res.result.Success {
				m.fleetResults[res.account.ID] = nil
			}
			if len(m.remediationResults) == len(m.accountsInFleet) {
				m.state = auditStateComplete
				m.status = i18n.T("remediation.tui.complete")
			}
		}
		return m, nil
	}
	return m, nil
}

func (m auditModel) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			return m, func() tea.Msg { return backToMenuMsg{} }
		case "up", "k":
			if m.menuCursor > 0 {
				m.menuCursor--
			}
		case "down", "j":
			if m.menuCursor < 3 {
				m.menuCursor++
			}
		case "m": // toggle mode
			if m.mode == auditModeStrict {
				m.mode = auditModeSerial
			} else {
				m.mode = auditModeStrict
			}
		case "enter":
			switch m.menuCursor {
			case 0: // Audit Fleet
				var err error
				m.accountsInFleet, err = db.GetAllActiveAccounts()
				if err != nil {
					m.err = err
					return m, nil
				}
				if len(m.accountsInFleet) == 0 {
					m.status = i18n.T("audit.tui.no_accounts")
					return m, nil
				}
				m.fleetResults = make(map[int]error, len(m.accountsInFleet))
				m.state = auditStateFleetInProgress
				m.status = i18n.T("audit.tui.starting_fleet")
				cmds := make([]tea.Cmd, len(m.accountsInFleet))
				for i, acc := range m.accountsInFleet {
					cmds[i] = performAuditCmd(acc, m.mode)
				}
				return m, tea.Batch(cmds...)
			case 1: // Audit Single
				m.accounts, _ = db.GetAllActiveAccounts()
				m.state = auditStateSelectAccount
				m.accountCursor = 0
				m.status = ""
				return m, nil
			case 2: // Audit Tag
				allAccounts, err := db.GetAllAccounts()
				if err != nil {
					m.err = err
					return m, nil
				}
				uniq := map[string]struct{}{}
				for _, acc := range allAccounts {
					if acc.Tags != "" {
						for _, t := range strings.Split(acc.Tags, ",") {
							if s := strings.TrimSpace(t); s != "" {
								uniq[s] = struct{}{}
							}
						}
					}
				}
				for tag := range uniq {
					m.tags = append(m.tags, tag)
				}
				sort.Strings(m.tags)
				m.tagCursor = 0
				m.state = auditStateSelectTag
				return m, nil
			case 3: // Toggle appears as mode indicator; pressing enter also toggles
				if m.mode == auditModeStrict {
					m.mode = auditModeSerial
				} else {
					m.mode = auditModeStrict
				}
			}
		}
	}
	return m, nil
}

func (m auditModel) updateAccountSelection(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "/":
			m.isFilteringAccount = true
			m.accountFilter = ""
			return m, nil
		case "up", "k":
			fa := m.getFilteredAccounts()
			if m.accountCursor > 0 {
				m.accountCursor--
			} else if len(fa) > 0 {
				m.accountCursor = len(fa) - 1
			}
			return m, nil
		case "down", "j":
			fa := m.getFilteredAccounts()
			if len(fa) > 0 {
				if m.accountCursor < len(fa)-1 {
					m.accountCursor++
				} else {
					m.accountCursor = 0
				}
			}
			return m, nil
		case "q":
			if m.accountFilter != "" && !m.isFilteringAccount {
				m.accountFilter = ""
				m.accountCursor = 0
				return m, nil
			}
			return m, func() tea.Msg { return backToMenuMsg{} }
		case "esc":
			if m.isFilteringAccount {
				m.isFilteringAccount = false
				// Do NOT clear m.accountFilter; persist filter after exiting filter mode
				m.status = ""
				return m, nil
			}
			if m.accountFilter != "" {
				m.accountFilter = ""
				m.status = ""
				m.accountCursor = 0
				return m, nil
			}
			m.status = ""
			m.state = auditStateMenu
			m.err = nil
			return m, nil
		case "backspace":
			if m.isFilteringAccount && m.accountFilter != "" {
				r := []rune(m.accountFilter)
				m.accountFilter = string(r[:len(r)-1])
				return m, nil
			}
		case "enter":
			if m.isFilteringAccount {
				m.isFilteringAccount = false
				return m, nil
			}
			fa := m.getFilteredAccounts()
			if len(fa) == 0 {
				return m, nil
			}
			m.selectedAccount = fa[m.accountCursor]
			m.state = auditStateInProgress
			m.status = ""
			return m, performAuditCmd(m.selectedAccount, m.mode)
		default:
			if m.isFilteringAccount && len(msg.String()) == 1 && msg.Type == tea.KeyRunes {
				m.accountFilter += msg.String()
				return m, nil
			}
		}
	}
	return m, nil
}

func (m auditModel) getFilteredAccounts() []model.Account {
	if m.accountFilter == "" {
		return m.accounts
	}
	var out []model.Account
	for _, acc := range m.accounts {
		if strings.Contains(strings.ToLower(acc.String()), strings.ToLower(m.accountFilter)) {
			out = append(out, acc)
		}
	}
	return out
}

func (m auditModel) updateSelectTag(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.state = auditStateMenu
			m.err = nil
			return m, nil
		case "up", "k":
			if m.tagCursor > 0 {
				m.tagCursor--
			}
		case "down", "j":
			if m.tagCursor < len(m.tags)-1 {
				m.tagCursor++
			}
		case "enter":
			if len(m.tags) == 0 {
				return m, nil
			}
			selected := m.tags[m.tagCursor]
			all, err := db.GetAllActiveAccounts()
			if err != nil {
				m.err = err
				return m, nil
			}
			var tagged []model.Account
			for _, acc := range all {
				for _, t := range strings.Split(acc.Tags, ",") {
					if strings.TrimSpace(t) == selected {
						tagged = append(tagged, acc)
						break
					}
				}
			}
			m.accountsInFleet = tagged
			if len(tagged) == 0 {
				m.status = i18n.T("audit.tui.no_accounts_tag", selected)
				m.state = auditStateMenu
				return m, nil
			}
			m.fleetResults = make(map[int]error, len(tagged))
			m.state = auditStateFleetInProgress
			m.status = i18n.T("audit.tui.starting_tag", selected)
			cmds := make([]tea.Cmd, len(tagged))
			for i, acc := range tagged {
				cmds[i] = performAuditCmd(acc, m.mode)
			}
			return m, tea.Batch(cmds...)
		}
	}
	return m, nil
}

func (m auditModel) updateComplete(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			// Initiate remediation for accounts with drift
			if len(m.fleetResults) > 0 {
				// Filter accounts with drift
				var accountsWithDrift []model.Account
				for _, acc := range m.accountsInFleet {
					if err, ok := m.fleetResults[acc.ID]; ok && err != nil {
						accountsWithDrift = append(accountsWithDrift, acc)
					}
				}
				if len(accountsWithDrift) > 0 {
					m.accountsInFleet = accountsWithDrift
					m.state = auditStateRemediationConfirm
					return m, nil
				}
			}
		case "esc", "enter":
			if len(m.fleetResults) > 0 {
				m.fleetResults = make(map[int]error)
				return m, func() tea.Msg { return backToMenuMsg{} }
			}
			m.state = auditStateSelectAccount
			m.err = nil
			m.status = ""
			return m, nil
		}
	}
	return m, nil
}

// updateRemediationConfirm handles the remediation confirmation screen.
func (m auditModel) updateRemediationConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "enter":
			// Start remediation
			m.state = auditStateRemediationInProgress
			m.status = i18n.T("remediation.tui.starting")
			m.remediationResults = make(map[int]*model.RemediationResult, len(m.accountsInFleet))
			cmds := make([]tea.Cmd, len(m.accountsInFleet))
			for i, acc := range m.accountsInFleet {
				cmds[i] = performRemediationCmd(acc)
			}
			return m, tea.Batch(cmds...)
		case "n", "esc":
			m.state = auditStateComplete
			return m, nil
		}
	}
	return m, nil
}

// View renders the audit UI.
func (m auditModel) View() string {
	// Header
	title := mainTitleStyle.Render("🔍 " + i18n.T("audit.tui.title"))

	modeLabel := i18n.T("audit.tui.mode_strict")
	if m.mode == auditModeSerial {
		modeLabel = i18n.T("audit.tui.mode_serial")
	}
	subTitle := helpStyle.Render(fmt.Sprintf("%s: %s", i18n.T("audit.tui.menu.toggle_mode"), modeLabel))
	header := lipgloss.JoinVertical(lipgloss.Left, title, subTitle)

	// Pane styles
	paneStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorSubtle).
		Padding(1, 2)

	paneTitleStyle := lipgloss.NewStyle().Bold(true)
	helpFooterStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Background(lipgloss.Color("236")).Padding(0, 1).Italic(true)

	// Calculate dimensions
	headerHeight := lipgloss.Height(header)
	footerHeight := lipgloss.Height(helpFooterStyle.Render(""))
	paneHeight := m.height - headerHeight - footerHeight - 2

	leftPaneWidth := 38
	rightPaneWidth := m.width - 4 - leftPaneWidth - 2

	if m.err != nil {
		content := errorStyle.Render(fmt.Sprintf(i18n.T("account_form.error"), m.err))
		leftPane := paneStyle.Width(leftPaneWidth).Height(paneHeight).Render(content)
		rightPane := paneStyle.Width(rightPaneWidth).Height(paneHeight).MarginLeft(2).Render("")
		mainArea := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
		help := helpFooterStyle.Render(i18n.T("audit.tui.help_failed"))
		return lipgloss.JoinVertical(lipgloss.Left, header, "", mainArea, "", help)
	}

	switch m.state {
	case auditStateMenu:
		// Left pane - Menu
		var leftItems []string
		leftItems = append(leftItems, paneTitleStyle.Render(i18n.T("audit.tui.menu.title")), "")
		menuItems := []string{"audit.tui.menu.audit_fleet", "audit.tui.menu.audit_single", "audit.tui.menu.audit_tag", "audit.tui.menu.toggle_mode"}
		for i, key := range menuItems {
			label := i18n.T(key)
			if key == "audit.tui.menu.toggle_mode" {
				label = i18n.T(key)
			}
			if m.menuCursor == i {
				leftItems = append(leftItems, selectedItemStyle.Render("▸ "+label))
			} else {
				leftItems = append(leftItems, itemStyle.Render("  "+label))
			}
		}
		leftContent := lipgloss.JoinVertical(lipgloss.Left, leftItems...)

		// Right pane - Info/Status
		var rightItems []string
		if m.status != "" {
			rightItems = append(rightItems, statusMessageStyle.Render(m.status))
		} else {
			rightItems = append(rightItems, paneTitleStyle.Render(i18n.T("audit.tui.current_mode")), "")
			rightItems = append(rightItems, helpStyle.Render(modeLabel))
		}
		rightContent := lipgloss.JoinVertical(lipgloss.Left, rightItems...)

		leftPane := paneStyle.Width(leftPaneWidth).Height(paneHeight).Render(leftContent)
		rightPane := paneStyle.Width(rightPaneWidth).Height(paneHeight).MarginLeft(2).Render(rightContent)
		mainArea := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
		help := helpFooterStyle.Render(i18n.T("audit.tui.help_menu"))
		return lipgloss.JoinVertical(lipgloss.Left, header, "", mainArea, "", help)

	case auditStateSelectAccount:
		// Calculate 50/50 split
		halfWidth := (m.width - 6) / 2 // -6 for borders and spacing

		// Left pane - Account list
		var leftItems []string
		leftItems = append(leftItems, paneTitleStyle.Render(i18n.T("audit.tui.select_account")), "")
		filtered := m.getFilteredAccounts()
		if m.accountCursor >= len(filtered) {
			m.accountCursor = 0
		}
		if len(filtered) == 0 {
			leftItems = append(leftItems, helpStyle.Render(i18n.T("audit.tui.no_accounts")))
		} else {
			for i, acc := range filtered {
				line := acc.String()
				if m.accountCursor == i {
					leftItems = append(leftItems, selectedItemStyle.Render("▸ "+line))
				} else {
					leftItems = append(leftItems, itemStyle.Render("  "+line))
				}
			}
		}
		leftContent := lipgloss.JoinVertical(lipgloss.Left, leftItems...)

		// Right pane - Host details for selected account
		var rightItems []string
		if len(filtered) > 0 && m.accountCursor < len(filtered) {
			selectedAcc := filtered[m.accountCursor]
			rightItems = append(rightItems, paneTitleStyle.Render(i18n.T("audit.tui.account_details")), "")
			rightItems = append(rightItems, fmt.Sprintf("%s: %s", i18n.T("audit.tui.detail_username"), selectedAcc.Username))
			rightItems = append(rightItems, fmt.Sprintf("%s: %s", i18n.T("audit.tui.detail_hostname"), selectedAcc.Hostname))
			if selectedAcc.Label != "" {
				rightItems = append(rightItems, fmt.Sprintf("%s: %s", i18n.T("audit.tui.detail_label"), selectedAcc.Label))
			}
			if selectedAcc.Tags != "" {
				rightItems = append(rightItems, fmt.Sprintf("%s: %s", i18n.T("audit.tui.detail_tags"), selectedAcc.Tags))
			}
			rightItems = append(rightItems, "")
			rightItems = append(rightItems, paneTitleStyle.Render(i18n.T("audit.tui.filter")), "")
		} else {
			rightItems = append(rightItems, paneTitleStyle.Render(i18n.T("audit.tui.filter")), "")
		}

		var filterStatus string
		if m.isFilteringAccount {
			filterStatus = i18n.T("audit.tui.filtering", m.accountFilter)
		} else if m.accountFilter != "" {
			filterStatus = i18n.T("audit.tui.filter_active", m.accountFilter)
		} else {
			filterStatus = i18n.T("audit.tui.filter_hint")
		}
		rightItems = append(rightItems, helpStyle.Render(filterStatus))
		rightContent := lipgloss.JoinVertical(lipgloss.Left, rightItems...)

		leftPane := paneStyle.Width(halfWidth).Height(paneHeight).Render(leftContent)
		rightPane := paneStyle.Width(halfWidth).Height(paneHeight).MarginLeft(2).Render(rightContent)
		mainArea := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
		help := helpFooterStyle.Render(i18n.T("audit.tui.help_select"))
		return lipgloss.JoinVertical(lipgloss.Left, header, "", mainArea, "", help)

	case auditStateSelectTag, auditStateFleetInProgress, auditStateInProgress, auditStateComplete, auditStateRemediationConfirm, auditStateRemediationInProgress:
		// Calculate 50/50 split
		halfWidth := (m.width - 6) / 2 // -6 for borders and spacing

		var leftItems []string
		var rightItems []string
		var helpText string

		switch m.state {
		case auditStateSelectTag:
			leftItems = append(leftItems, paneTitleStyle.Render(i18n.T("audit.tui.select_tag")), "")
			if len(m.tags) == 0 {
				leftItems = append(leftItems, helpStyle.Render(i18n.T("audit.tui.no_tags")))
			} else {
				for i, tag := range m.tags {
					if m.tagCursor == i {
						leftItems = append(leftItems, selectedItemStyle.Render("▸ "+tag))
					} else {
						leftItems = append(leftItems, itemStyle.Render("  "+tag))
					}
				}
			}
			helpText = i18n.T("audit.tui.help_select")

		case auditStateFleetInProgress:
			leftItems = append(leftItems, paneTitleStyle.Render(i18n.T("audit.tui.auditing_fleet")), "")
			for _, acc := range m.accountsInFleet {
				res, ok := m.fleetResults[acc.ID]
				var status string
				if !ok {
					status = helpStyle.Render(i18n.T("audit.tui.pending"))
				} else if res != nil {
					status = "💥 " + errorStyle.Render(i18n.T("audit.tui.failed_short"))
				} else {
					status = "✅ " + successStyle.Render(i18n.T("audit.tui.ok_short"))
				}
				leftItems = append(leftItems, fmt.Sprintf("  %s %s", acc.String(), status))
			}
			helpText = i18n.T("audit.tui.help_wait")
			if m.status != "" {
				rightItems = append(rightItems, paneTitleStyle.Render(i18n.T("audit.tui.status")), "")
				rightItems = append(rightItems, statusMessageStyle.Render(m.status))
			}

		case auditStateInProgress:
			leftItems = append(leftItems, paneTitleStyle.Render(i18n.T("audit.tui.auditing")), "")
			if m.selectedAccount.ID > 0 {
				leftItems = append(leftItems, fmt.Sprintf("%s: %s", i18n.T("audit.tui.detail_username"), m.selectedAccount.Username))
				leftItems = append(leftItems, fmt.Sprintf("%s: %s", i18n.T("audit.tui.detail_hostname"), m.selectedAccount.Hostname))
				if m.selectedAccount.Label != "" {
					leftItems = append(leftItems, fmt.Sprintf("%s: %s", i18n.T("audit.tui.detail_label"), m.selectedAccount.Label))
				}
			}
			rightItems = append(rightItems, paneTitleStyle.Render(i18n.T("audit.tui.status")), "")
			rightItems = append(rightItems, helpStyle.Render(i18n.T("audit.tui.help_wait")))
			helpText = i18n.T("audit.tui.help_wait")

		case auditStateComplete:
			leftItems = append(leftItems, paneTitleStyle.Render(i18n.T("audit.tui.complete")), "")
			if len(m.fleetResults) > 0 {
				okCount := 0
				driftCount := 0
				for _, acc := range m.accountsInFleet {
					if err, ok := m.fleetResults[acc.ID]; ok {
						if err == nil {
							okCount++
						} else {
							driftCount++
						}
					}
				}
				leftItems = append(leftItems, successStyle.Render(fmt.Sprintf("✅ %s: %d", i18n.T("audit.tui.ok_short"), okCount)))
				if driftCount > 0 {
					leftItems = append(leftItems, errorStyle.Render(fmt.Sprintf("💥 %s: %d", i18n.T("audit.tui.failed_short"), driftCount)))
				}
				leftItems = append(leftItems, "")
				// Show failed accounts in right pane
				for _, acc := range m.accountsInFleet {
					if err, ok := m.fleetResults[acc.ID]; ok && err != nil {
						rightItems = append(rightItems, errorStyle.Render(fmt.Sprintf("✗ %s: %v", acc.String(), err)))
					}
				}
			} else {
				leftItems = append(leftItems, m.status)
			}
			hasDrift := false
			for _, err := range m.fleetResults {
				if err != nil {
					hasDrift = true
					break
				}
			}
			if hasDrift {
				helpText = i18n.T("audit.tui.help_complete_with_drift")
			} else {
				helpText = i18n.T("audit.tui.help_complete")
			}

		case auditStateRemediationConfirm:
			leftItems = append(leftItems, paneTitleStyle.Render(i18n.T("remediation.tui.confirm_title")), "")
			leftItems = append(leftItems, i18n.T("remediation.tui.confirm_message", len(m.accountsInFleet)), "")
			for _, acc := range m.accountsInFleet {
				leftItems = append(leftItems, fmt.Sprintf("  • %s", acc.String()))
			}
			leftItems = append(leftItems, "", i18n.T("remediation.tui.confirm_question"))
			helpText = i18n.T("remediation.tui.help_confirm")

		case auditStateRemediationInProgress:
			leftItems = append(leftItems, paneTitleStyle.Render(i18n.T("remediation.tui.in_progress_title")), "")
			for _, acc := range m.accountsInFleet {
				result, ok := m.remediationResults[acc.ID]
				var status string
				if !ok {
					status = helpStyle.Render(i18n.T("remediation.tui.pending"))
				} else if result.Success {
					status = "✅ " + successStyle.Render(i18n.T("remediation.tui.success"))
				} else {
					status = "❌ " + errorStyle.Render(i18n.T("remediation.tui.failed"))
				}
				leftItems = append(leftItems, fmt.Sprintf("  %s %s", acc.String(), status))
			}
			helpText = i18n.T("remediation.tui.help_wait")
		}

		leftContent := lipgloss.JoinVertical(lipgloss.Left, leftItems...)
		rightContent := lipgloss.JoinVertical(lipgloss.Left, rightItems...)

		leftPane := paneStyle.Width(halfWidth).Height(paneHeight).Render(leftContent)
		rightPane := paneStyle.Width(halfWidth).Height(paneHeight).MarginLeft(2).Render(rightContent)
		mainArea := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
		help := helpFooterStyle.Render(helpText)
		return lipgloss.JoinVertical(lipgloss.Left, header, "", mainArea, "", help)
	}
	return ""
}

// performAuditCmd executes the selected audit mode for an account.
func performAuditCmd(account model.Account, mode auditModeType) tea.Cmd {
	return func() tea.Msg {
		var err error
		if mode == auditModeSerial {
			err = deploy.AuditAccountSerial(account)
		} else {
			err = deploy.AuditAccountStrict(account)
		}
		return auditResultMsg{account: account, err: err}
	}
}

// performRemediationCmd executes remediation for an account.
func performRemediationCmd(account model.Account) tea.Cmd {
	return func() tea.Msg {
		result, err := deploy.RemediateAccount(account, false)
		return remediationResultMsg{account: account, result: result, err: err}
	}
}
