package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/model"
)

type publicKeysViewState int

const (
	publicKeysListView publicKeysViewState = iota
	publicKeysFormView
	publicKeysUsageView
)

type publicKeysModel struct {
	state            publicKeysViewState
	form             publicKeyFormModel
	keys             []model.PublicKey // Master list
	displayedKeys    []model.PublicKey // Filtered list
	cursor           int
	status           string
	err              error
	usageReportKey   model.PublicKey
	usageReportAccts []model.Account
	filter           string
	isFiltering      bool
	// For delete confirmation
	isConfirmingDelete bool
	keyToDelete        model.PublicKey
	confirmCursor      int // 0 for No, 1 for Yes
	width, height      int
}

func newPublicKeysModel() publicKeysModel {
	m := publicKeysModel{}
	var err error
	m.keys, err = db.GetAllPublicKeys()
	if err != nil {
		m.err = err
	}
	m.rebuildDisplayedKeys()
	return m
}

func (m publicKeysModel) Init() tea.Cmd {
	return nil
}

func (m *publicKeysModel) rebuildDisplayedKeys() {
	if m.filter == "" {
		m.displayedKeys = m.keys
	} else {
		m.displayedKeys = []model.PublicKey{}
		lowerFilter := strings.ToLower(m.filter)
		for _, key := range m.keys {
			if strings.Contains(strings.ToLower(key.Comment), lowerFilter) ||
				strings.Contains(strings.ToLower(key.Algorithm), lowerFilter) {
				m.displayedKeys = append(m.displayedKeys, key)
			}
		}
	}

	// Reset cursor if it's out of bounds
	if m.cursor >= len(m.displayedKeys) {
		if len(m.displayedKeys) > 0 {
			m.cursor = len(m.displayedKeys) - 1
		} else {
			m.cursor = 0
		}
	}
}

func (m publicKeysModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Handle window size messages first, as they affect layout.
	if sizeMsg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = sizeMsg.Width
		m.height = sizeMsg.Height
	}

	if m.state == publicKeysFormView {
		if _, ok := msg.(publicKeyCreatedMsg); ok {
			m.state = publicKeysListView
			m.status = "Successfully added new public key."
			m.keys, m.err = db.GetAllPublicKeys()
			m.rebuildDisplayedKeys()
			return m, nil
		}
		if _, ok := msg.(backToListMsg); ok {
			m.state = publicKeysListView
			m.status = ""
			return m, nil
		}
		var newFormModel tea.Model
		newFormModel, cmd = m.form.Update(msg)
		m.form = newFormModel.(publicKeyFormModel)
		return m, cmd
	}

	if m.state == publicKeysUsageView {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "q", "esc":
				m.state = publicKeysListView
				m.status = ""
				return m, nil
			}
		}
		return m, nil
	}

	// List view logic
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle delete confirmation
		if m.isConfirmingDelete {
			switch msg.String() {
			case "y":
				// Fallthrough to confirm
			case "n", "q", "esc":
				m.isConfirmingDelete = false
				m.status = "Deletion cancelled."
				return m, nil
			case "right", "tab", "l":
				m.confirmCursor = 1 // Yes
				return m, nil
			case "left", "shift+tab", "h":
				m.confirmCursor = 0 // No
				return m, nil
			case "enter":
				if m.confirmCursor == 1 { // Yes is selected
					if err := db.DeletePublicKey(m.keyToDelete.ID); err != nil {
						m.err = err
					} else {
						m.status = fmt.Sprintf("Deleted key: %s", m.keyToDelete.Comment)
						m.keys, m.err = db.GetAllPublicKeys()
						m.rebuildDisplayedKeys()
					}
				}
				m.isConfirmingDelete = false
				return m, nil
			}
			return m, nil
		}

		// If we are in filtering mode, capture all input for the filter.
		if m.isFiltering {
			switch msg.Type {
			case tea.KeyEsc:
				m.isFiltering = false
				m.filter = ""
				m.rebuildDisplayedKeys()
			case tea.KeyEnter:
				m.isFiltering = false
			case tea.KeyBackspace:
				if len(m.filter) > 0 {
					m.filter = m.filter[:len(m.filter)-1]
					m.rebuildDisplayedKeys()
				}
			case tea.KeyRunes:
				m.filter += string(msg.Runes)
				m.rebuildDisplayedKeys()
			}
			return m, nil
		}

		// Not in filtering mode, handle commands.
		switch msg.String() {
		case "/":
			m.isFiltering = true
			m.filter = "" // Start with a fresh filter
			m.rebuildDisplayedKeys()
			return m, nil
		case "q", "esc":
			if m.filter != "" && !m.isFiltering {
				m.filter = ""
				m.rebuildDisplayedKeys()
				return m, nil
			}
			return m, func() tea.Msg { return backToMenuMsg{} }
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.displayedKeys)-1 {
				m.cursor++
			}
		case "a":
			m.state = publicKeysFormView
			m.form = newPublicKeyFormModel()
			m.status = ""
			return m, m.form.Init()
		case "d", "delete":
			if len(m.displayedKeys) > 0 {
				m.keyToDelete = m.displayedKeys[m.cursor]
				m.isConfirmingDelete = true
				m.confirmCursor = 0 // Default to No
			}
			return m, nil
		case "g": // Toggle global status
			if len(m.displayedKeys) > 0 {
				keyToToggle := m.displayedKeys[m.cursor]
				if err := db.TogglePublicKeyGlobal(keyToToggle.ID); err != nil {
					m.err = err
				} else {
					m.status = fmt.Sprintf("Toggled global status for key: %s", keyToToggle.Comment)
					// Refresh the list to show the new status
					m.keys, m.err = db.GetAllPublicKeys()
					m.rebuildDisplayedKeys()
				}
			}
			return m, nil
		case "u": // Usage report
			if len(m.displayedKeys) > 0 {
				m.usageReportKey = m.displayedKeys[m.cursor]
				accounts, err := db.GetAccountsForKey(m.usageReportKey.ID)
				if err != nil {
					m.err = err
					return m, nil
				}
				m.usageReportAccts = accounts
				m.state = publicKeysUsageView
				m.status = ""
			}
			return m, nil
		}
	}
	return m, nil
}

func (m publicKeysModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	if m.isConfirmingDelete {
		return m.viewConfirmation()
	}

	switch m.state {
	case publicKeysFormView:
		return m.form.View()
	case publicKeysUsageView:
		return m.viewUsageReport()
	default: // publicKeysListView
		return m.viewKeyList()
	}
}

func (m publicKeysModel) viewConfirmation() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("🗑️ Confirm Deletion"))

	question := fmt.Sprintf("Are you sure you want to delete the public key\n\n%s?", m.keyToDelete.Comment)
	b.WriteString(question)
	b.WriteString("\n\nThis will remove it from all accounts it is assigned to.")
	b.WriteString("\n\n")

	var yesButton, noButton string
	if m.confirmCursor == 1 { // Yes
		yesButton = activeButtonStyle.Render("Yes, Delete")
		noButton = buttonStyle.Render("No, Cancel")
	} else { // No
		yesButton = buttonStyle.Render("Yes, Delete")
		noButton = activeButtonStyle.Render("No, Cancel")
	}

	buttons := lipgloss.JoinHorizontal(lipgloss.Top, noButton, "  ", yesButton)
	b.WriteString(buttons)

	b.WriteString("\n" + helpStyle.Render("\n(left/right to navigate, enter to confirm, esc to cancel)"))

	// Center the whole dialog
	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		dialogBoxStyle.Render(b.String()),
	)
}

func (m publicKeysModel) viewKeyList() string {
	var viewItems []string
	viewItems = append(viewItems, titleStyle.Render("🔑 Manage Public Keys"))

	var listItems []string
	for i, key := range m.displayedKeys {
		var globalMarker string
		if key.IsGlobal {
			globalMarker = "🌐 "
		}
		line := fmt.Sprintf("%s%s (%s)", globalMarker, key.Comment, key.Algorithm)
		if m.cursor == i {
			listItems = append(listItems, selectedItemStyle.Render("▸ "+line))
		} else {
			listItems = append(listItems, itemStyle.Render("  "+line))
		}
	}
	viewItems = append(viewItems, lipgloss.JoinVertical(lipgloss.Left, listItems...))

	if len(m.displayedKeys) == 0 && m.filter == "" {
		viewItems = append(viewItems, helpStyle.Render("No public keys found. Press 'a' to add one."))
	} else if len(m.displayedKeys) == 0 && m.filter != "" {
		viewItems = append(viewItems, helpStyle.Render("No keys match your filter."))
	}
	viewItems = append(viewItems, "") // Spacer

	var filterStatus string
	if m.isFiltering {
		filterStatus = fmt.Sprintf("Filter: %s█", m.filter)
	} else if m.filter != "" {
		filterStatus = fmt.Sprintf("Filter: %s (press 'esc' to clear)", m.filter)
	} else {
		filterStatus = "Press / to filter..."
	}

	viewItems = append(viewItems, helpStyle.Render(fmt.Sprintf("(a)dd (d)elete (g)lobal toggle (u)sage (q)uit\n%s", filterStatus)))
	if m.status != "" {
		viewItems = append(viewItems, "", statusMessageStyle.Render(m.status))
	}

	return lipgloss.JoinVertical(lipgloss.Left, viewItems...)
}

func (m publicKeysModel) viewUsageReport() string {
	var b strings.Builder
	title := fmt.Sprintf("📜 Key Usage Report for: %s", m.usageReportKey.Comment)
	b.WriteString(titleStyle.Render(title))

	if len(m.usageReportAccts) == 0 {
		b.WriteString(helpStyle.Render("This key is not assigned to any accounts."))
	} else {
		b.WriteString("This key is assigned to the following accounts:\n\n")
		for _, acc := range m.usageReportAccts {
			b.WriteString(itemStyle.Render("- "+acc.String()) + "\n")
		}
	}

	b.WriteString(helpStyle.Render("\n(esc or q to go back)"))
	return b.String()
}
