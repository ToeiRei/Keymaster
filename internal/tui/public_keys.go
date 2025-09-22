package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
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
				keyToDelete := m.displayedKeys[m.cursor]
				if err := db.DeletePublicKey(keyToDelete.ID); err != nil {
					m.err = err
				} else {
					m.status = fmt.Sprintf("Deleted key: %s", keyToDelete.Comment)
					m.keys, m.err = db.GetAllPublicKeys()
					m.rebuildDisplayedKeys()
				}
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

	switch m.state {
	case publicKeysFormView:
		return m.form.View()
	case publicKeysUsageView:
		return m.viewUsageReport()
	default: // publicKeysListView
		return m.viewKeyList()
	}
}

func (m publicKeysModel) viewKeyList() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("🔑 Manage Public Keys"))
	b.WriteString("\n\n")

	for i, key := range m.displayedKeys {
		var globalMarker string
		if key.IsGlobal {
			globalMarker = "🌐 "
		}
		line := fmt.Sprintf("%s%s (%s)", globalMarker, key.Comment, key.Algorithm)
		if m.cursor == i {
			b.WriteString(selectedItemStyle.Render("» " + line))
		} else {
			b.WriteString(itemStyle.Render(line))
		}
		b.WriteString("\n")
	}

	if len(m.displayedKeys) == 0 && m.filter == "" {
		b.WriteString(helpStyle.Render("No public keys found. Press 'a' to add one."))
	} else if len(m.displayedKeys) == 0 && m.filter != "" {
		b.WriteString(helpStyle.Render("No keys match your filter."))
	}

	var filterStatus string
	if m.isFiltering {
		filterStatus = fmt.Sprintf("Filter: %s█", m.filter)
	} else if m.filter != "" {
		filterStatus = fmt.Sprintf("Filter: %s (press 'esc' to clear)", m.filter)
	} else {
		filterStatus = "Press / to filter..."
	}

	b.WriteString(helpStyle.Render(fmt.Sprintf("\n(a)dd, (d)elete, (g)lobal toggle, (u)sage report, (q)uit\n%s", filterStatus)))
	if m.status != "" {
		b.WriteString(helpStyle.Render("\n\n" + m.status))
	}

	return b.String()
}

func (m publicKeysModel) viewUsageReport() string {
	var b strings.Builder
	title := fmt.Sprintf("📜 Key Usage Report for: %s", m.usageReportKey.Comment)
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")

	if len(m.usageReportAccts) == 0 {
		b.WriteString(helpStyle.Render("This key is not assigned to any accounts."))
	} else {
		b.WriteString("This key is assigned to the following accounts:\n\n")
		for _, acc := range m.usageReportAccts {
			b.WriteString(itemStyle.Render("- " + acc.String()))
			b.WriteString("\n")
		}
	}

	b.WriteString(helpStyle.Render("\n(esc or q to go back)"))
	return b.String()
}
