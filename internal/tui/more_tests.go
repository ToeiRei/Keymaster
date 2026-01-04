package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/internal/i18n"
	"github.com/toeirei/keymaster/internal/model"
)

func TestApplySuggestion_AlreadyPresent_NoDuplicate(t *testing.T) {
	i18n.Init("en")
	var m accountFormModel
	m.inputs = make([]textinput.Model, 4)
	for i := range m.inputs {
		m.inputs[i] = textinput.New()
	}
	m.allTags = []string{"ops", "dev"}
	m.focusIndex = 3
	// input already contains 'ops'
	m.inputs[3].SetValue("ops, ")
	m.updateSuggestions()
	// suggestions should not include 'ops' because it's already present
	for _, s := range m.suggestions {
		if s == "ops" {
			t.Fatalf("did not expect 'ops' in suggestions when already present: %v", m.suggestions)
		}
	}
}

func TestUpdateInputs_DoesNotPanic_OnKeyMsg(t *testing.T) {
	var m accountFormModel
	m.inputs = make([]textinput.Model, 4)
	for i := range m.inputs {
		m.inputs[i] = textinput.New()
	}
	// send a rune key to updateInputs and ensure it returns without panic
	_, _ = m.updateInputs(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
}

func TestEnsureCursorInView_TopAndBottom(t *testing.T) {
	m := accountsModel{}
	m.viewport = viewport.New(0, 0)
	m.viewport.Height = 3
	// Use a few displayed accounts
	m.displayedAccounts = []model.Account{{ID: 1}, {ID: 2}, {ID: 3}, {ID: 4}, {ID: 5}}

	// Cursor above top -> set YOffset
	m.viewport.YOffset = 2
	m.cursor = 0
	m.ensureCursorInView()
	if m.viewport.YOffset != 0 {
		t.Fatalf("expected YOffset 0 when cursor at 0, got %d", m.viewport.YOffset)
	}

	// Cursor below bottom -> set YOffset accordingly
	m.cursor = 4
	m.ensureCursorInView()
	if m.viewport.YOffset == 0 {
		t.Fatalf("expected YOffset > 0 when cursor at bottom, got %d", m.viewport.YOffset)
	}
}

// Note: viewportStub previously existed here but was unused; removed to
// satisfy linter unused checks. Tests use the real `viewport` in current code.

func TestViewConfirmationAndKeySelection_Render(t *testing.T) {
	i18n.Init("en")
	m := accountsModel{}
	m.width = 80
	m.height = 24
	m.accountToDelete = model.Account{Username: "x", Hostname: "h"}
	m.withDecommission = true
	m.confirmCursor = 1 // Yes focused
	out := m.viewConfirmation()
	if !strings.Contains(out, "Yes") && !strings.Contains(out, "No") {
		t.Fatalf("expected confirmation buttons in output, got: %q", out)
	}

	// key selection view
	m.availableKeys = []model.PublicKey{{ID: 1, Comment: "k1", KeyData: "0123456789abcdef"}}
	m.keySelectionCursor = 0
	m.keySelectionInButtonMode = false
	ks := m.viewKeySelection()
	if !strings.Contains(ks, "key_selection") && !strings.Contains(ks, "cancel") {
		// not strict; ensure view renders strings
		if ks == "" {
			t.Fatalf("expected non-empty key selection view")
		}
	}
}
