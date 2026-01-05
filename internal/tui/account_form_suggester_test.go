// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package tui

import (
	"reflect"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/toeirei/keymaster/internal/i18n"
)

type mockSuggester struct {
	all     []string
	suggest func(string) []string
}

func (m *mockSuggester) AllTags() ([]string, error)  { return m.all, nil }
func (m *mockSuggester) Suggest(val string) []string { return m.suggest(val) }

func TestFormUsesTagSuggester_AllTagsPopulated(t *testing.T) {
	i18n.Init("en")
	_ = initTestDB()

	ms := &mockSuggester{all: []string{"z", "a"}, suggest: func(string) []string { return nil }}
	m := newAccountFormModelWithSuggester(nil, ms)
	// allTags should be populated from suggester and sorted
	if !reflect.DeepEqual(m.allTags, []string{"a", "z"}) {
		t.Fatalf("unexpected allTags: %v", m.allTags)
	}
}

func TestFormUsesTagSuggester_SuggestUsed(t *testing.T) {
	i18n.Init("en")
	_ = initTestDB()

	ms := &mockSuggester{
		all: []string{"x"},
		suggest: func(val string) []string {
			if val == "v" {
				return []string{"vault-mock"}
			}
			return nil
		},
	}
	m := newAccountFormModelWithSuggester(nil, ms)
	// prepare tags input
	for i := range m.inputs {
		m.inputs[i] = textinput.New()
	}
	m.focusIndex = 3
	m.inputs[3].SetValue("v")
	mp := &m
	mp.updateSuggestions()
	if len(mp.suggestions) != 1 || mp.suggestions[0] != "vault-mock" {
		t.Fatalf("expected suggestion from mock suggester, got: %v", mp.suggestions)
	}
}

