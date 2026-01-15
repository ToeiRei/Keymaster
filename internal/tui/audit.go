// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package tui // import "github.com/toeirei/keymaster/internal/tui"

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/internal/core"
	"github.com/toeirei/keymaster/internal/i18n"
	"github.com/toeirei/keymaster/internal/model"
	"github.com/toeirei/keymaster/internal/state"
	"github.com/toeirei/keymaster/internal/ui"
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
	auditStateEnterPassphrase
	auditStateInProgress
	auditStateComplete
)

// auditResultMsg is a message to signal audit completion for one account.
type auditResultMsg struct {
	account model.Account
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
	fleetResults       map[int]error // map account ID to error
	selectedAccount    model.Account
	tags               []string
	status             string
	err                error
	accountFilter      string
	isFilteringAccount bool
	passphraseInput    textinput.Model
	pendingCmd         tea.Cmd // Command to re-run after getting passphrase
	wasFleetDeploy     bool
	width, height      int
}

func newAuditModel() auditModel {
	return auditModel{
		state:           auditStateMenu,
		mode:            auditModeStrict,
		fleetResults:    make(map[int]error),
		passphraseInput: newPassphraseInput(),
	}
}

func (m auditModel) Init() tea.Cmd { return nil }

func (m auditModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.state {
	case auditStateMenu:
		return m.updateMenu(msg)
	case auditStateSelectAccount:
		return m.updateAccountSelection(msg)
	case auditStateSelectTag:
		return m.updateSelectTag(msg)
	case auditStateFleetInProgress:
		return m.updateFleetInProgress(msg)
	case auditStateEnterPassphrase:
		return m.updateEnterPassphrase(msg)
	case auditStateInProgress:
		if res, ok := msg.(auditResultMsg); ok {
			if res.err != nil {
				if deployAdapter.IsPassphraseRequired(res.err) {
					m.state = auditStateEnterPassphrase
					m.err = nil // We are handling this error
					m.passphraseInput.Focus()
					m.pendingCmd = performAuditCmd(res.account, m.mode)
					return m, textinput.Blink
				}
				// It's a different, final error.
				m.state = auditStateComplete
				m.err = res.err
				m.status = i18n.T("audit.tui.failed_short")
			} else {
				m.state = auditStateComplete
				m.err = nil
				m.status = i18n.T("audit.tui.ok_short")
			}
		}
		return m, nil
	case auditStateComplete:
		return m.updateComplete(msg)
	}
	return m, nil
}

func (m auditModel) updateFleetInProgress(msg tea.Msg) (tea.Model, tea.Cmd) {
	if res, ok := msg.(auditResultMsg); ok {
		m.fleetResults[res.account.ID] = res.err

		// If any audit requires a passphrase, stop and ask for it.
		if res.err != nil && deployAdapter.IsPassphraseRequired(res.err) {
			m.state = auditStateEnterPassphrase
			m.err = nil // Clear the error as we are handling it
			m.passphraseInput.Focus()
			// Store a command to re-run the entire fleet audit.
			cmds := make([]tea.Cmd, len(m.accountsInFleet))
			for i, acc := range m.accountsInFleet {
				cmds[i] = performAuditCmd(acc, m.mode)
			}
			m.pendingCmd = tea.Batch(cmds...)
			return m, textinput.Blink
		}

		if len(m.fleetResults) == len(m.accountsInFleet) {
			m.state = auditStateComplete
			m.status = i18n.T("audit.tui.fleet_complete")
		}
	}
	return m, nil
}

func (m auditModel) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
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
				m.wasFleetDeploy = true
				var err error
				m.accountsInFleet, err = ui.GetAllActiveAccounts()
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
				m.wasFleetDeploy = false
				m.accounts, _ = ui.GetAllActiveAccounts()
				m.state = auditStateSelectAccount
				m.accountCursor = 0
				m.status = ""
				return m, nil
			case 2: // Audit Tag
				m.wasFleetDeploy = true
				allAccounts, err := ui.GetAllAccounts()
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
		case "esc":
			if m.isFilteringAccount {
				m.isFilteringAccount = false
				return m, nil
			}
			if m.accountFilter != "" {
				m.accountFilter = ""
				return m, nil
			}
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
			m.status = i18n.T("audit.tui.auditing", m.selectedAccount.String())
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
		if core.ContainsIgnoreCase(acc.String(), m.accountFilter) {
			out = append(out, acc)
		}
	}
	return out
}

func (m auditModel) updateEnterPassphrase(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			passphrase := m.passphraseInput.Value()
			state.PasswordCache.Set([]byte(passphrase))
			if m.wasFleetDeploy {
				m.fleetResults = make(map[int]error) // Clear previous fleet results before retry
				m.state = auditStateFleetInProgress
			} else {
				m.state = auditStateInProgress // For single audit, go back to single in-progress
			}
			m.status = i18n.T("deploy.retrying")
			return m, m.pendingCmd // Re-run the original command
		case "esc":
			m.state = auditStateMenu
			m.err = nil
			m.status = i18n.T("deploy.passphrase_cancelled")
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.passphraseInput, cmd = m.passphraseInput.Update(msg)
	return m, cmd
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
			all, err := ui.GetAllActiveAccounts()
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
		case "esc", "enter":
			if m.wasFleetDeploy {
				m.fleetResults = make(map[int]error) // Clear results
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

// View renders the audit UI.
func (m auditModel) View() string {
	paneStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorSubtle).Padding(1, 2)
	helpFooterStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Background(lipgloss.Color("236")).Padding(0, 1).Italic(true)

	if m.err != nil {
		title := titleStyle.Render(i18n.T("audit.tui.failed"))
		help := helpFooterStyle.Render(i18n.T("audit.tui.help_failed"))
		content := fmt.Sprintf(i18n.T("account_form.error"), m.err)
		mainPane := paneStyle.Width(60).Render(lipgloss.JoinVertical(lipgloss.Left, title, "", content))
		return lipgloss.JoinVertical(lipgloss.Left, mainPane, "", help)
	}

	modeLabel := i18n.T("audit.tui.mode_strict")
	if m.mode == auditModeSerial {
		modeLabel = i18n.T("audit.tui.mode_serial")
	}

	switch m.state {
	case auditStateMenu:
		title := titleStyle.Render(i18n.T("audit.tui.title"))
		var listItems []string
		menuItems := []string{"audit.tui.menu.audit_fleet", "audit.tui.menu.audit_single", "audit.tui.menu.audit_tag", "audit.tui.menu.toggle_mode"}
		for i, key := range menuItems {
			label := i18n.T(key)
			if key == "audit.tui.menu.toggle_mode" {
				label = fmt.Sprintf("%s: %s", label, modeLabel)
			}
			if m.menuCursor == i {
				listItems = append(listItems, selectedItemStyle.Render("▸ "+label))
			} else {
				listItems = append(listItems, itemStyle.Render("  "+label))
			}
		}
		mainPane := paneStyle.Width(60).Render(lipgloss.JoinVertical(lipgloss.Left, title, "", lipgloss.JoinVertical(lipgloss.Left, listItems...)))
		help := helpFooterStyle.Render(i18n.T("audit.tui.help_menu"))
		if m.status != "" {
			mainPane += "\n" + helpFooterStyle.Render(m.status)
		}
		return lipgloss.JoinVertical(lipgloss.Left, mainPane, "", help)

	case auditStateSelectAccount:
		title := titleStyle.Render(i18n.T("audit.tui.select_account"))
		var listItems []string
		filtered := m.getFilteredAccounts()
		if m.accountCursor >= len(filtered) {
			m.accountCursor = 0
		}
		if len(filtered) == 0 {
			listItems = append(listItems, helpStyle.Render(i18n.T("audit.tui.no_accounts")))
		} else {
			for i, acc := range filtered {
				line := acc.String()
				if m.accountCursor == i {
					listItems = append(listItems, selectedItemStyle.Render("▸ "+line))
				} else {
					listItems = append(listItems, itemStyle.Render("  "+line))
				}
			}
		}
		mainPane := paneStyle.Width(60).Render(lipgloss.JoinVertical(lipgloss.Left, title, "", lipgloss.JoinVertical(lipgloss.Left, listItems...)))
		var filterStatus string
		if m.isFilteringAccount {
			filterStatus = i18n.T("audit.tui.filtering", m.accountFilter)
		} else if m.accountFilter != "" {
			filterStatus = i18n.T("audit.tui.filter_active", m.accountFilter)
		} else {
			filterStatus = i18n.T("audit.tui.filter_hint")
		}
		left := i18n.T("audit.tui.help_select")
		help := helpFooterStyle.Render(AlignFooter(left, filterStatus, m.width))
		return lipgloss.JoinVertical(lipgloss.Left, mainPane, "", help)

	case auditStateSelectTag:
		title := titleStyle.Render(i18n.T("audit.tui.select_tag"))
		var listItems []string
		if len(m.tags) == 0 {
			listItems = append(listItems, helpStyle.Render(i18n.T("audit.tui.no_tags")))
		} else {
			for i, tag := range m.tags {
				if m.tagCursor == i {
					listItems = append(listItems, selectedItemStyle.Render("▸ "+tag))
				} else {
					listItems = append(listItems, itemStyle.Render("  "+tag))
				}
			}
		}
		mainPane := paneStyle.Width(60).Render(lipgloss.JoinVertical(lipgloss.Left, title, "", lipgloss.JoinVertical(lipgloss.Left, listItems...)))
		help := helpFooterStyle.Render(i18n.T("audit.tui.help_select"))
		return lipgloss.JoinVertical(lipgloss.Left, mainPane, "", help)

	case auditStateFleetInProgress:
		title := titleStyle.Render(i18n.T("audit.tui.auditing_fleet"))
		var statusLines []string
		for _, acc := range m.accountsInFleet {
			res, ok := m.fleetResults[acc.ID]
			var status string
			if !ok {
				status = helpStyle.Render(i18n.T("audit.tui.pending"))
			} else if res != nil {
				status = "🚨 " + helpStyle.Render(i18n.T("audit.tui.failed_short"))
			} else {
				status = "✅ " + successStyle.Render(i18n.T("audit.tui.ok_short"))
			}
			statusLines = append(statusLines, fmt.Sprintf("  %s %s", acc.String(), status))
		}
		mainPane := paneStyle.Width(60).Render(lipgloss.JoinVertical(lipgloss.Left, title, "", lipgloss.JoinVertical(lipgloss.Left, statusLines...)))
		help := helpFooterStyle.Render(i18n.T("audit.tui.help_wait"))
		if m.status != "" {
			mainPane += "\n" + helpFooterStyle.Render(m.status)
		}
		return lipgloss.JoinVertical(lipgloss.Left, mainPane, "", help)

	case auditStateEnterPassphrase:
		var b strings.Builder
		b.WriteString(titleStyle.Render(i18n.T("deploy.passphrase_title")))
		b.WriteString("\n\n")
		b.WriteString(i18n.T("deploy.passphrase_prompt"))
		b.WriteString("\n\n")
		b.WriteString(m.passphraseInput.View())
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render(i18n.T("deploy.passphrase_help")))

		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			dialogBoxStyle.Render(b.String()),
		)

	case auditStateInProgress:
		title := titleStyle.Render(i18n.T("audit.tui.auditing"))
		mainPane := paneStyle.Width(60).Render(lipgloss.JoinVertical(lipgloss.Left, title, "", m.status))
		return lipgloss.JoinVertical(lipgloss.Left, mainPane)

	case auditStateComplete:
		title := titleStyle.Render(i18n.T("audit.tui.complete"))
		mainPane := paneStyle.Width(60).Render(lipgloss.JoinVertical(lipgloss.Left, title, "", m.status))
		if len(m.fleetResults) > 0 {
			okCount := 0
			var failed []string
			for _, acc := range m.accountsInFleet {
				if err, ok := m.fleetResults[acc.ID]; ok {
					if err == nil {
						okCount++
					} else {
						failed = append(failed, fmt.Sprintf("  - %s: %v", acc.String(), err))
					}
				}
			}
			mainPane += i18n.T("audit.tui.summary", okCount, len(failed))
			if len(failed) > 0 {
				mainPane += i18n.T("audit.tui.failed_accounts", strings.Join(failed, "\n"))
			}
		}
		help := helpFooterStyle.Render(i18n.T("audit.tui.help_complete"))
		return lipgloss.JoinVertical(lipgloss.Left, mainPane, "", help)
	}
	return ""
}

// performAuditCmd executes the selected audit mode for an account.
func performAuditCmd(account model.Account, mode auditModeType) tea.Cmd {
	return func() tea.Msg {
		var err error
		// Use core facade so TUI doesn't call deploy audit helpers directly.
		if mode == auditModeSerial {
			err = core.RunAuditForAccount(context.Background(), deployAdapter, account, "serial", nil)
		} else {
			err = core.RunAuditForAccount(context.Background(), deployAdapter, account, "strict", nil)
		}
		return auditResultMsg{account: account, err: err}
	}
}

