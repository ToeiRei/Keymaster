package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/model"
)

// A simple style for focused text inputs.
var focusedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("170"))
var disabledStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

// A message to signal that an account was modified (created or updated).
type accountModifiedMsg struct {
	isNew    bool
	hostname string
}

type accountFormModel struct {
	focusIndex     int
	inputs         []textinput.Model // 0: user, 1: host, 2: label, 3: tags
	err            error
	editingAccount *model.Account // If not nil, we are in edit mode.

	// For tag autocompletion
	allTags          []string
	suggestions      []string
	suggestionCursor int
	isSuggesting     bool
}

func newAccountFormModel(accountToEdit *model.Account) accountFormModel {
	m := accountFormModel{
		inputs:       make([]textinput.Model, 4),
		isSuggesting: false,
	}

	var t textinput.Model
	for i := range m.inputs {
		t = textinput.New()
		t.Cursor.Style = focusedStyle
		t.CharLimit = 64
		t.Width = 40 // Give the input a fixed width

		switch i {
		case 0:
			t.Prompt = "Username:               "
			t.Placeholder = "user"
		case 1:
			t.Prompt = "Hostname:               "
			t.Placeholder = "www.example.com"
		case 2:
			t.Prompt = "Label (optional):       "
			t.Placeholder = "prod-web-01"
		case 3:
			t.Prompt = "Tags (comma-separated): "
			t.Placeholder = "role:db,dc:nyc"
		}
		m.inputs[i] = t
	}

	if accountToEdit != nil {
		m.editingAccount = accountToEdit
		m.inputs[0].SetValue(accountToEdit.Username)
		m.inputs[0].PromptStyle = disabledStyle
		m.inputs[0].TextStyle = disabledStyle
		m.inputs[1].SetValue(accountToEdit.Hostname)
		m.inputs[1].PromptStyle = disabledStyle
		m.inputs[1].TextStyle = disabledStyle
		m.inputs[2].SetValue(accountToEdit.Label)
		m.inputs[3].SetValue(accountToEdit.Tags)
		m.inputs[2].Focus() // Start focus on label
		m.inputs[2].TextStyle = focusedStyle
		m.focusIndex = 2 // Start focus on label
	} else {
		m.inputs[0].Focus()
		m.inputs[0].TextStyle = focusedStyle
	}

	// --- Populate tags for autocompletion ---
	allAccounts, err := db.GetAllAccounts()
	if err != nil {
		// Not a fatal error for the form, just no autocomplete.
		// We could log this or handle it more gracefully.
	}
	tagSet := make(map[string]struct{})
	for _, acc := range allAccounts {
		if acc.Tags != "" {
			tags := strings.Split(acc.Tags, ",")
			for _, tag := range tags {
				trimmedTag := strings.TrimSpace(tag)
				if trimmedTag != "" {
					tagSet[trimmedTag] = struct{}{}
				}
			}
		}
	}
	m.allTags = make([]string, 0, len(tagSet))
	for tag := range tagSet {
		m.allTags = append(m.allTags, tag)
	}
	sort.Strings(m.allTags) // Keep them sorted for predictable display

	return m
}

func (m accountFormModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m accountFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// If we are suggesting tags, handle that first.
		if m.isSuggesting {
			switch msg.String() {
			case "up":
				if m.suggestionCursor > 0 {
					m.suggestionCursor--
				}
				return m, nil
			case "down":
				if m.suggestionCursor < len(m.suggestions)-1 {
					m.suggestionCursor++
				}
				return m, nil
			case "tab", "enter":
				m.applySuggestion()
				m.updateSuggestions()
				return m, nil
			case "esc":
				m.isSuggesting = false
				m.suggestions = nil
				return m, nil
			}
		}

		switch msg.String() {
		// Go back to the accounts list.
		case "esc":
			return m, func() tea.Msg { return backToListMsg{} }

		// Set focus to next input
		case "tab", "shift+tab", "enter", "up", "down":
			s := msg.String()

			// If we are on the tags input and press down, and there are suggestions,
			// enter suggestion mode instead of cycling focus.
			if m.focusIndex == 3 && s == "down" && len(m.suggestions) > 0 && !m.isSuggesting {
				m.isSuggesting = true
				m.suggestionCursor = 0
				return m, nil
			}

			// Did the user press enter while the submit button was focused?
			// If so, create the account.
			if s == "enter" && m.focusIndex == len(m.inputs) {
				if m.editingAccount != nil {
					// Update existing account
					label := m.inputs[2].Value()
					tags := m.inputs[3].Value()
					if err := db.UpdateAccountLabel(m.editingAccount.ID, label); err != nil {
						m.err = err
						return m, nil
					}
					if err := db.UpdateAccountTags(m.editingAccount.ID, tags); err != nil {
						m.err = err
						return m, nil
					}
					// Signal that we're done.
					return m, func() tea.Msg { return accountModifiedMsg{isNew: false} }
				} else {
					// Add new account
					username := strings.TrimSpace(m.inputs[0].Value())
					hostname := strings.TrimSpace(m.inputs[1].Value())
					label := strings.TrimSpace(m.inputs[2].Value())
					tags := strings.TrimSpace(m.inputs[3].Value())

					if username == "" || hostname == "" {
						m.err = fmt.Errorf("username and hostname cannot be empty")
						return m, nil
					}

					if err := db.AddAccount(username, hostname, label, tags); err != nil {
						m.err = err
						return m, nil
					}
					// Signal that we're done.
					return m, func() tea.Msg { return accountModifiedMsg{isNew: true, hostname: hostname} }
				}
			}

			// Cycle focus
			if m.editingAccount != nil { // In edit mode
				// Cycle between label, tags, and submit button
				if s == "up" || s == "shift+tab" {
					m.focusIndex--
					if m.focusIndex < 2 { // 2 is the first editable field (label)
						m.focusIndex = len(m.inputs)
					}
				} else {
					m.focusIndex++
					if m.focusIndex > len(m.inputs) { // len(m.inputs) is the submit button
						m.focusIndex = 2
					}
				}
			} else {
				// In add mode, cycle through all fields
				if s == "up" || s == "shift+tab" {
					m.focusIndex--
				} else {
					m.focusIndex++
				}
				if m.focusIndex > len(m.inputs) {
					m.focusIndex = 0
				} else if m.focusIndex < 0 {
					m.focusIndex = len(m.inputs)
				}
			}

			cmds := make([]tea.Cmd, len(m.inputs))
			for i := 0; i <= len(m.inputs)-1; i++ {
				if i == m.focusIndex {
					cmds[i] = m.inputs[i].Focus()
					m.inputs[i].TextStyle = focusedStyle
					if m.editingAccount != nil && i < 2 {
						m.inputs[i].TextStyle = disabledStyle
					}
					continue
				}
				m.inputs[i].Blur()
				m.inputs[i].TextStyle = lipgloss.NewStyle()
			}

			return m, tea.Batch(cmds...)
		}
	}

	// Handle character input and blinking
	oldVal := ""
	if m.focusIndex == 3 {
		oldVal = m.inputs[3].Value()
	}

	var cmd tea.Cmd
	m, cmd = m.updateInputs(msg)

	// If tag input changed, update suggestions
	if m.focusIndex == 3 && m.inputs[3].Value() != oldVal {
		m.updateSuggestions()
	}

	return m, cmd
}

func (m accountFormModel) updateInputs(msg tea.Msg) (accountFormModel, tea.Cmd) {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return m, tea.Batch(cmds...)
}

func (m accountFormModel) View() string {
	var viewItems []string

	if m.editingAccount != nil {
		viewItems = append(viewItems, titleStyle.Render("✏️ Edit Account"))
	} else {
		viewItems = append(viewItems, titleStyle.Render("✨ Add New Account"))
	}

	// The title's padding adds a newline, so we add one more for a blank line.
	viewItems = append(viewItems, "")
	for i := range m.inputs {
		viewItems = append(viewItems, m.inputs[i].View())
		// If this is the tags input, add suggestions right after it.
		if i == 3 && len(m.suggestions) > 0 {
			var suggestionLines []string
			for i, s := range m.suggestions {
				if i == m.suggestionCursor {
					suggestionLines = append(suggestionLines, selectedItemStyle.Render("▸ "+s))
				} else {
					suggestionLines = append(suggestionLines, "  "+s)
				}
			}
			// Align the suggestions box with the start of the text input area.
			suggestionsBox := lipgloss.NewStyle().
				PaddingLeft(len(m.inputs[i].Prompt) + 1).
				Render(lipgloss.JoinVertical(lipgloss.Left, suggestionLines...))
			viewItems = append(viewItems, suggestionsBox)
		}
	}

	button := formItemStyle.Render("[ Submit ]")
	if m.focusIndex == len(m.inputs) {
		button = formSelectedItemStyle.Render("[ Submit ]")
	}
	viewItems = append(viewItems, "", button) // Blank line before button

	if m.err != nil {
		viewItems = append(viewItems, "", helpStyle.Render(fmt.Sprintf("Error: %v", m.err)))
	}

	viewItems = append(viewItems, "", helpStyle.Render("(tab to navigate, enter to submit, esc to cancel)"))

	return lipgloss.JoinVertical(lipgloss.Left, viewItems...)
}

// updateSuggestions calculates a new list of suggestions based on the current input.
func (m *accountFormModel) updateSuggestions() {
	if m.focusIndex != 3 {
		m.suggestions = nil
		m.isSuggesting = false
		return
	}

	currentVal := m.inputs[3].Value()
	parts := strings.Split(currentVal, ",")
	if len(parts) == 0 {
		m.suggestions = nil
		return
	}

	lastPart := strings.TrimSpace(parts[len(parts)-1])
	if lastPart == "" {
		m.suggestions = nil
		m.isSuggesting = false
		return
	}

	m.suggestions = []string{}
	for _, tag := range m.allTags {
		if strings.HasPrefix(strings.ToLower(tag), strings.ToLower(lastPart)) {
			// Also check if the tag is already in the input
			isAlreadyPresent := false
			for i := 0; i < len(parts)-1; i++ {
				if strings.TrimSpace(parts[i]) == tag {
					isAlreadyPresent = true
					break
				}
			}
			if !isAlreadyPresent {
				m.suggestions = append(m.suggestions, tag)
			}
		}
	}

	m.suggestionCursor = 0
}

// applySuggestion replaces the last typed tag with the selected suggestion.
func (m *accountFormModel) applySuggestion() {
	if !m.isSuggesting || m.suggestionCursor >= len(m.suggestions) {
		return
	}
	selectedSuggestion := m.suggestions[m.suggestionCursor]
	currentVal := m.inputs[3].Value()
	parts := strings.Split(currentVal, ",")
	parts[len(parts)-1] = selectedSuggestion
	newValue := strings.Join(parts, ", ") + ", "
	m.inputs[3].SetValue(newValue)
	m.inputs[3].SetCursor(len(newValue))
}
