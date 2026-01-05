// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/toeirei/keymaster/internal/i18n"
)

func TestAccountForm_UpdateSuggestionsAndApply(t *testing.T) {
	i18n.Init("en")

	var m accountFormModel
	m.inputs = make([]textinput.Model, 4)
	for i := range m.inputs {
		m.inputs[i] = textinput.New()
	}

	// populate tags and simulate typing
	m.allTags = []string{"dev", "prod", "vault", "test"}
	m.focusIndex = 3
	m.inputs[3].SetValue("v")

	// should populate suggestions with tags starting with 'v'
	m.updateSuggestions()
	if len(m.suggestions) == 0 {
		t.Fatalf("expected suggestions for 'v', got none")
	}
	found := false
	for _, s := range m.suggestions {
		if s == "vault" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected 'vault' in suggestions, got: %v", m.suggestions)
	}

	// apply suggestion
	m.isSuggesting = true
	m.suggestionCursor = 0
	m.applySuggestion()
	if got := m.inputs[3].Value(); got == "" || got[len(got)-2:] != ", " {
		t.Fatalf("expected input to be replaced with suggestion and trailing comma, got: %q", got)
	}
}

