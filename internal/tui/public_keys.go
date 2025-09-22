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
	keys             []model.PublicKey
	cursor           int
	status           string
	err              error
	usageReportKey   model.PublicKey
	usageReportAccts []model.Account
}

func newPublicKeysModel() publicKeysModel {
	m := publicKeysModel{}
	var err error
	m.keys, err = db.GetAllPublicKeys()
	if err != nil {
		m.err = err
	}
	return m
}

func (m publicKeysModel) Init() tea.Cmd {
	return nil
}

func (m publicKeysModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if m.state == publicKeysFormView {
		if _, ok := msg.(publicKeyCreatedMsg); ok {
			m.state = publicKeysListView
			m.status = "Successfully added new public key."
			m.keys, m.err = db.GetAllPublicKeys()
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
		switch msg.String() {
		case "q", "esc":
			return m, func() tea.Msg { return backToMenuMsg{} }
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.keys)-1 {
				m.cursor++
			}
		case "a":
			m.state = publicKeysFormView
			m.form = newPublicKeyFormModel()
			m.status = ""
			return m, m.form.Init()
		case "d", "delete":
			if len(m.keys) > 0 {
				keyToDelete := m.keys[m.cursor]
				if err := db.DeletePublicKey(keyToDelete.ID); err != nil {
					m.err = err
				} else {
					m.status = fmt.Sprintf("Deleted key: %s", keyToDelete.Comment)
					m.keys, m.err = db.GetAllPublicKeys()
					if m.cursor >= len(m.keys) && len(m.keys) > 0 {
						m.cursor = len(m.keys) - 1
					}
				}
			}
			return m, nil
		case "g": // Toggle global status
			if len(m.keys) > 0 {
				keyToToggle := m.keys[m.cursor]
				if err := db.TogglePublicKeyGlobal(keyToToggle.ID); err != nil {
					m.err = err
				} else {
					m.status = fmt.Sprintf("Toggled global status for key: %s", keyToToggle.Comment)
					// Refresh the list to show the new status
					m.keys, m.err = db.GetAllPublicKeys()
				}
			}
			return m, nil
		case "u": // Usage report
			if len(m.keys) > 0 {
				m.usageReportKey = m.keys[m.cursor]
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

	for i, key := range m.keys {
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

	if len(m.keys) == 0 {
		b.WriteString(helpStyle.Render("No public keys found. Press 'a' to add one."))
	}

	b.WriteString(helpStyle.Render("\n(a)dd, (d)elete, (g)lobal toggle, (u)sage report, (q)uit"))
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
