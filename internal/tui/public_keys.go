package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/model"
)

// A message to signal that we should go back to the keys list from the form.
type backToKeysMsg struct{}

type publicKeysViewState int

const (
	keysListView publicKeysViewState = iota
	keysFormView
)

// publicKeysModel is the model for the public key management view.
type publicKeysModel struct {
	state  publicKeysViewState
	form   publicKeyFormModel
	keys   []model.PublicKey
	cursor int
	status string
	err    error
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

	// Delegate updates to the form if it's active.
	if m.state == keysFormView {
		if _, ok := msg.(publicKeyCreatedMsg); ok {
			m.state = keysListView
			m.status = "Successfully added new public key."
			m.keys, m.err = db.GetAllPublicKeys()
			return m, nil
		}
		if _, ok := msg.(backToKeysMsg); ok {
			m.state = keysListView
			m.status = ""
			return m, nil
		}

		var newFormModel tea.Model
		newFormModel, cmd = m.form.Update(msg)
		m.form = newFormModel.(publicKeyFormModel)
		return m, cmd
	}

	// --- This is the list view update logic ---
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

		case "a":
			m.state = keysFormView
			m.form = newPublicKeyFormModel()
			m.status = ""
			return m, m.form.Init()
		}
	}
	return m, nil
}

func (m publicKeysModel) View() string {
	if m.state == keysFormView {
		return m.form.View()
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("🔑 Manage Public Keys"))
	b.WriteString("\n\n")

	if m.err != nil {
		b.WriteString(fmt.Sprintf("Error: %v\n", m.err))
		return b.String()
	}

	for i, key := range m.keys {
		// Displaying as "Comment (Algorithm)"
		line := fmt.Sprintf("%s (%s)", key.Comment, key.Algorithm)
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

	b.WriteString(helpStyle.Render("\n(a)dd, (d)elete, (q)uit to menu"))
	if m.status != "" {
		b.WriteString(helpStyle.Render("\n\n" + m.status))
	}

	return b.String()
}
