// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// package tui provides the terminal user interface for Keymaster.
// This file contains the logic for the account creation and editing form,
// including input handling, validation, and tag autocompletion.
package tui // import "github.com/toeirei/keymaster/internal/tui"

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/internal/core"
	"github.com/toeirei/keymaster/internal/core/db"
	"github.com/toeirei/keymaster/internal/core/logging"
	"github.com/toeirei/keymaster/internal/core/model"
	"github.com/toeirei/keymaster/internal/i18n"
	"github.com/toeirei/keymaster/internal/uiadapters"
)

// focusedStyle is a simple style for focused text inputs.
var focusedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("170"))

// disabledStyle is a simple style for disabled text inputs.
var disabledStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

// accountModifiedMsg is a message to signal that an account was modified (created or updated).
type accountModifiedMsg struct {
	isNew     bool
	username  string
	hostname  string
	accountID int
}

// bootstrapRequestedMsg is a message to signal that bootstrap workflow should be started.
type bootstrapRequestedMsg struct {
	username string
	hostname string
	label    string
	tags     string
}

type accountFormModel struct {
	// focusIndex determines which input or button is currently active.
	// 0: user, 1: host, 2: label, 3: tags, 4: bootstrap checkbox, 5: submit button.
	focusIndex     int
	inputs         []textinput.Model // 0: user, 1: host, 2: label, 3: tags
	err            error
	editingAccount *model.Account // If not nil, we are in edit mode.

	// Bootstrap functionality
	bootstrapEnabled bool // Whether bootstrap mode is enabled

	// For tag autocompletion
	allTags          []string
	suggestions      []string
	suggestionCursor int
	isSuggesting     bool
	tagSuggester     TagSuggester
}

// newAccountFormModelWithSuggester creates a new form model with an injected
// TagSuggester. Pass `nil` to use the package default suggester.
// TagSuggester defines a minimal suggester used by the TUI form.
type TagSuggester interface {
	AllTags() ([]string, error)
	Suggest(currentVal string) []string
}

// dbTagSuggester implements TagSuggester by scanning the DB.
type dbTagSuggester struct{}

func (d *dbTagSuggester) AllTags() ([]string, error) {
	allAccounts, err := db.GetAllAccounts()
	if err != nil {
		return nil, err
	}
	tagSet := make(map[string]struct{})
	for _, acc := range allAccounts {
		if acc.Tags == "" {
			continue
		}
		for _, tag := range strings.Split(acc.Tags, ",") {
			t := strings.TrimSpace(tag)
			if t != "" {
				tagSet[t] = struct{}{}
			}
		}
	}
	tags := make([]string, 0, len(tagSet))
	for t := range tagSet {
		tags = append(tags, t)
	}
	sort.Strings(tags)
	return tags, nil
}

func (d *dbTagSuggester) Suggest(currentVal string) []string {
	tags, err := d.AllTags()
	if err != nil {
		return nil
	}
	return SuggestTags(tags, currentVal)
}

func newAccountFormModelWithSuggester(accountToEdit *model.Account, ts TagSuggester) accountFormModel {
	m := accountFormModel{
		inputs:       make([]textinput.Model, 4),
		isSuggesting: false,
		tagSuggester: ts,
	}

	var t textinput.Model
	for i := range m.inputs {
		t = textinput.New()
		t.Cursor.Style = focusedStyle
		t.CharLimit = 64
		t.Width = 40 // Give the input a fixed width

		switch i {
		case 0:
			t.Prompt = i18n.T("account_form.username_label")
			t.Placeholder = i18n.T("account_form.username_placeholder")
		case 1:
			t.Prompt = i18n.T("account_form.hostname_label")
			t.Placeholder = i18n.T("account_form.hostname_placeholder")
		case 2:
			t.Prompt = i18n.T("account_form.label_label")
			t.Placeholder = i18n.T("account_form.label_placeholder")
		case 3:
			t.Prompt = i18n.T("account_form.tags_label")
			t.Placeholder = i18n.T("account_form.tags_placeholder")
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
	// If an explicit TagSuggester was provided, prefer it.
	if ts != nil {
		m.tagSuggester = ts
		if tags, err := ts.AllTags(); err == nil {
			m.allTags = tags
			sort.Strings(m.allTags)
			return m
		}
		// Fall through to fallback DB scan on error
	}
	// If no suggester provided, try the canonical store adapter first.
	if ts == nil {
		if s := uiadapters.NewStoreAdapter(); s != nil {
			if accts, err := s.GetAllAccounts(); err == nil {
				tagSet := make(map[string]struct{})
				for _, acc := range accts {
					for _, tag := range SplitTags(acc.Tags) {
						tagSet[tag] = struct{}{}
					}
				}
				m.allTags = make([]string, 0, len(tagSet))
				for tag := range tagSet {
					m.allTags = append(m.allTags, tag)
				}
				sort.Strings(m.allTags)
				return m
			}
		}
		// Fallback to DB-backed suggester if adapter not available
		d := dbTagSuggester{}
		if tags, err := d.AllTags(); err == nil {
			m.allTags = tags
			sort.Strings(m.allTags)
			return m
		}
	}

	// Final fallback: scan DB directly to preserve legacy behavior
	allAccounts, err := db.GetAllAccounts()
	if err != nil {
		logging.Warnf("failed to load accounts for tag autocomplete: %v", err)
	}
	tagSet := make(map[string]struct{})
	for _, acc := range allAccounts {
		for _, tag := range SplitTags(acc.Tags) {
			tagSet[tag] = struct{}{}
		}
	}
	m.allTags = make([]string, 0, len(tagSet))
	for tag := range tagSet {
		m.allTags = append(m.allTags, tag)
	}
	sort.Strings(m.allTags)

	return m
}

// newAccountFormModel is the original convenience constructor that uses the
// default `TagSuggester`.
func newAccountFormModel(accountToEdit *model.Account) accountFormModel {
	// Preserve legacy behavior for callers that expect to set `allTags` manually
	// in tests: do not inject a suggester by default.
	return newAccountFormModelWithSuggester(accountToEdit, nil)
}

// Init initializes the form model, returning a command to start the cursor blinking.
func (m accountFormModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages and updates the form model's state.
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

		// --- Handle character input for the focused field ---
		// This needs to happen before the navigation switch case to allow typing.
		oldVal := ""
		if m.focusIndex == 3 { // Tags input
			oldVal = m.inputs[3].Value()
		}
		var inputCmd tea.Cmd
		m, inputCmd = m.updateInputs(msg)
		if m.focusIndex == 3 && m.inputs[3].Value() != oldVal {
			m.updateSuggestions()
		}
		// --- End character input handling ---

		switch msg.String() {
		// Go back to the accounts list.
		case "esc":
			return m, func() tea.Msg { return backToListMsg{} }

		// Handle spacebar for bootstrap checkbox toggle
		case " ":
			if m.focusIndex == len(m.inputs) && m.editingAccount == nil {
				m.bootstrapEnabled = !m.bootstrapEnabled
				return m, nil
			}
			// If not on checkbox, pass through to input
			if m.focusIndex < len(m.inputs) {
				var spaceCmd tea.Cmd
				m.inputs[m.focusIndex], spaceCmd = m.inputs[m.focusIndex].Update(msg)
				return m, spaceCmd
			}
			return m, nil

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

			// Handle checkbox toggle for bootstrap (enter key)
			if s == "enter" && m.focusIndex == len(m.inputs) && m.editingAccount == nil {
				m.bootstrapEnabled = !m.bootstrapEnabled
				return m, nil
			}

			submitButtonIndex := len(m.inputs)
			if m.editingAccount == nil {
				submitButtonIndex++ // Account for bootstrap checkbox in add mode
			}

			// Did the user press enter while the submit button was focused?
			// If so, create the account or start bootstrap.
			if s == "enter" && m.focusIndex == submitButtonIndex {
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
					return m, func() tea.Msg {
						return accountModifiedMsg{isNew: false, username: m.editingAccount.Username, hostname: m.editingAccount.Hostname, accountID: m.editingAccount.ID}
					}
				} else {
					// Validate inputs first
					username := strings.TrimSpace(m.inputs[0].Value())
					hostname := strings.TrimSpace(m.inputs[1].Value())
					label := strings.TrimSpace(m.inputs[2].Value())
					tags := strings.TrimSpace(m.inputs[3].Value())

					if username == "" || hostname == "" {
						m.err = fmt.Errorf("username and hostname cannot be empty")
						return m, nil
					}

					if m.bootstrapEnabled {
						// Start bootstrap workflow
						return m, func() tea.Msg {
							return bootstrapRequestedMsg{
								username: username,
								hostname: hostname,
								label:    label,
								tags:     tags,
							}
						}
					} else {
						// Add new account via injected AccountManager when available.
						mgr := db.DefaultAccountManager()
						if mgr == nil {
							m.err = fmt.Errorf("no account manager configured")
							return m, nil
						}
						newID, err := mgr.AddAccount(username, hostname, label, tags)
						if err != nil {
							m.err = err
							return m, nil
						}
						// Signal that we're done.
						return m, func() tea.Msg {
							return accountModifiedMsg{isNew: true, username: username, hostname: hostname, accountID: newID}
						}
					}
				}
			}

			// Cycle focus
			if m.editingAccount != nil { // In edit mode
				// Cycle between label, tags, and submit button (no bootstrap checkbox in edit mode)
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
				// In add mode, cycle through all fields including bootstrap checkbox
				// 0-3: inputs, 4: bootstrap checkbox, 5: submit button
				if s == "up" || s == "shift+tab" {
					m.focusIndex--
				} else {
					m.focusIndex++
				}
				if m.focusIndex > len(m.inputs)+1 { // +1 for bootstrap checkbox + submit button
					m.focusIndex = 0
				} else if m.focusIndex < 0 {
					m.focusIndex = len(m.inputs) + 1 // Start at submit button
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

			return m, tea.Batch(append(cmds, inputCmd)...)
		}
	}
	return m, nil
}

// --- Local tag helpers (copied from ui/helpers.go) ---
// SuggestTags returns a list of suggested tags based on the current input value.
func SuggestTags(allTags []string, currentVal string) []string {
	parts := strings.Split(currentVal, ",")
	if len(parts) == 0 {
		return nil
	}
	lastPart := strings.TrimSpace(parts[len(parts)-1])
	if lastPart == "" {
		return nil
	}
	lowerLast := strings.ToLower(lastPart)

	present := make(map[string]struct{})
	for i := 0; i < len(parts)-1; i++ {
		p := strings.ToLower(strings.TrimSpace(parts[i]))
		if p != "" {
			present[p] = struct{}{}
		}
	}

	var suggestions []string
	for _, tag := range allTags {
		lowerTag := strings.ToLower(tag)
		if strings.HasPrefix(lowerTag, lowerLast) {
			if _, ok := present[lowerTag]; !ok {
				suggestions = append(suggestions, tag)
			}
		}
	}
	return suggestions
}

// SplitTags splits a comma-separated tag string into trimmed, non-empty tags.
func SplitTags(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		tp := strings.TrimSpace(p)
		if tp != "" {
			out = append(out, tp)
		}
	}
	return out
}

// SplitTagsPreserveTrailing splits tags like SplitTags but preserves an empty
// trailing element when the input ends with a comma. Each part is trimmed.
func SplitTagsPreserveTrailing(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, strings.TrimSpace(p))
	}
	return out
}

// updateInputs passes messages to the underlying text input models.
func (m accountFormModel) updateInputs(msg tea.Msg) (accountFormModel, tea.Cmd) {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return m, tea.Batch(cmds...)
}

// View renders the account form UI based on the current model state.
func (m accountFormModel) View() string {
	var contentItems []string

	// Title section
	if m.editingAccount != nil {
		contentItems = append(contentItems, titleStyle.Render("✏️ "+i18n.T("account_form.edit_title")))
	} else {
		contentItems = append(contentItems, titleStyle.Render("✨ "+i18n.T("account_form.add_title")))
	}

	contentItems = append(contentItems, "")

	// Input fields
	for i := range m.inputs {
		contentItems = append(contentItems, m.inputs[i].View())
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
			contentItems = append(contentItems, suggestionsBox)
		}
	}

	// Add bootstrap checkbox (only in add mode, not edit mode)
	if m.editingAccount == nil {
		checkbox := "☐ " + i18n.T("account_form.bootstrap_label")
		if m.bootstrapEnabled {
			checkbox = "☑ " + i18n.T("account_form.bootstrap_label")
		}

		if m.focusIndex == len(m.inputs) {
			checkbox = formSelectedItemStyle.Render(checkbox)
		} else {
			checkbox = formItemStyle.Render(checkbox)
		}
		contentItems = append(contentItems, "", checkbox)
	}

	// Submit button using modern button style
	submitButtonIndex := len(m.inputs)
	if m.editingAccount == nil {
		submitButtonIndex = len(m.inputs) + 1 // Account for bootstrap checkbox
	}

	buttonText := i18n.T("account_form.submit")
	if m.editingAccount == nil && m.bootstrapEnabled {
		buttonText = i18n.T("account_form.bootstrap_submit")
	}

	var button string
	if m.focusIndex == submitButtonIndex {
		button = activeButtonStyle.Render(buttonText)
	} else {
		button = buttonStyle.Render(buttonText)
	}
	contentItems = append(contentItems, "", button)

	// Error message
	if m.err != nil {
		contentItems = append(contentItems, "", errorStyle.Render(fmt.Sprintf(i18n.T("account_form.error"), m.err)))
	}

	// Main pane with border (matching other pages)
	paneStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorSubtle).
		Padding(1, 2).
		Width(60)

	mainContent := paneStyle.Render(lipgloss.JoinVertical(lipgloss.Left, contentItems...))

	// Help footer with background (matching other pages)
	helpFooterStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Background(lipgloss.Color("236")).
		Padding(0, 1).
		Italic(true)

	helpFooter := helpFooterStyle.Render(i18n.T("account_form.help"))

	return lipgloss.JoinVertical(lipgloss.Left, mainContent, "", helpFooter)
}

// updateSuggestions calculates a new list of suggestions based on the current input.
func (m *accountFormModel) updateSuggestions() {
	if m.focusIndex != 3 {
		m.suggestions = nil
		m.isSuggesting = false
		return
	}

	currentVal := m.inputs[3].Value()
	parts := SplitTagsPreserveTrailing(currentVal)
	if len(parts) == 0 {
		m.suggestions = nil
		return
	}

	lastPart := parts[len(parts)-1]
	if lastPart == "" {
		m.suggestions = nil
		m.isSuggesting = false
		return
	}

	// Prefer using the injected TagSuggester when available so tests and
	// alternative UIs can provide custom suggestion behavior.
	if m.tagSuggester != nil {
		m.suggestions = m.tagSuggester.Suggest(currentVal)
	} else {
		m.suggestions = core.SuggestTags(m.allTags, currentVal)
	}
	m.suggestionCursor = 0
	m.isSuggesting = len(m.suggestions) > 0
}

// applySuggestion replaces the last typed tag with the selected suggestion.
func (m *accountFormModel) applySuggestion() {
	if !m.isSuggesting || m.suggestionCursor >= len(m.suggestions) {
		return
	}
	selectedSuggestion := m.suggestions[m.suggestionCursor]
	currentVal := m.inputs[3].Value()
	newValue := core.ApplySuggestion(currentVal, selectedSuggestion)
	m.inputs[3].SetValue(newValue)
	m.inputs[3].SetCursor(len(newValue))
}
