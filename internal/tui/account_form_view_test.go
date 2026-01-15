// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/internal/i18n"
	"github.com/toeirei/keymaster/internal/model"
)

func TestAccountForm_View_AddAndEditModes(t *testing.T) {
	i18n.Init("en")
	_ = initTestDB()

	// Add mode
	m := newAccountFormModel(nil)
	out := m.View()
	if out == "" {
		t.Fatalf("expected non-empty view for add mode")
	}
	if !strings.Contains(out, "✨") {
		t.Fatalf("expected add-mode emoji present in view, got: %q", out)
	}

	// Edit mode
	acc := &model.Account{ID: 5, Username: "u5", Hostname: "h5", Label: "L"}
	me := newAccountFormModel(acc)
	out2 := me.View()
	if out2 == "" {
		t.Fatalf("expected non-empty view for edit mode")
	}
	if !strings.Contains(out2, "✏️") {
		t.Fatalf("expected edit-mode emoji present in view, got: %q", out2)
	}
}

func TestAccountForm_Validation_EmptyRequiredFields(t *testing.T) {
	i18n.Init("en")
	_ = initTestDB()

	m := newAccountFormModel(nil)
	// Ensure username/hostname empty
	for i := range m.inputs {
		m.inputs[i] = textinput.New()
	}
	// Place focus on submit button index
	submitIndex := len(m.inputs) + 1 // add mode: inputs + bootstrap checkbox + submit
	m.focusIndex = submitIndex

	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	res := mm.(accountFormModel)
	if res.err == nil {
		t.Fatalf("expected validation error when username/hostname empty")
	}
	if !strings.Contains(res.err.Error(), "username and hostname cannot be empty") {
		t.Fatalf("unexpected validation error message: %v", res.err)
	}
}

func TestAccountForm_View_ShowsSuggestions(t *testing.T) {
	i18n.Init("en")
	_ = initTestDB()

	m := newAccountFormModel(nil)
	for i := range m.inputs {
		m.inputs[i] = textinput.New()
	}
	m.allTags = []string{"alpha", "beta", "vault"}
	m.focusIndex = 3
	m.inputs[3].SetValue("v")
	m.updateSuggestions()
	if len(m.suggestions) == 0 {
		t.Fatalf("expected suggestions for 'v'")
	}
	out := m.View()
	// suggestion should appear in view output
	if !strings.Contains(out, "vault") {
		t.Fatalf("expected suggestion 'vault' in view output, got: %q", out)
	}
}
