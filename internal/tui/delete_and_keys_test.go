package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/i18n"
	"github.com/toeirei/keymaster/internal/model"
)

func TestDeleteConfirm_NoDecommission_DeletesAccount(t *testing.T) {
	i18n.Init("en")
	_ = initTestDB()

	// Add an account to DB
	id, err := db.AddAccount("deluser", "delhost", "lbl", "")
	if err != nil {
		t.Fatalf("failed to add account: %v", err)
	}

	m := accountsModel{}
	m.accountToDelete = model.Account{ID: id}
	m.isConfirmingDelete = true
	m.withDecommission = false
	m.confirmCursor = 1 // Yes focused

	mi, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m1 := mi.(*accountsModel)
	if m1.isConfirmingDelete {
		t.Fatalf("expected isConfirmingDelete false after confirming delete")
	}

	// Ensure account removed from DB
	accts, err := db.GetAllAccounts()
	if err != nil {
		t.Fatalf("GetAllAccounts failed: %v", err)
	}
	for _, a := range accts {
		if a.ID == id {
			t.Fatalf("expected account %d to be deleted", id)
		}
	}
}

func TestKeySelection_NavigationAndButtons(t *testing.T) {
	i18n.Init("en")

	m := accountsModel{}
	m.isConfirmingKeySelection = true
	m.availableKeys = []model.PublicKey{{ID: 1, Comment: "k1", KeyData: "0123456789abcdef"}, {ID: 2, Comment: "k2", KeyData: "fedcba9876543210"}}
	m.selectedKeysToKeep = map[int]bool{1: true, 2: true}
	m.keySelectionCursor = 0
	m.keySelectionInButtonMode = false

	// Move down
	mi, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m1 := mi.(*accountsModel)
	if m1.keySelectionCursor != 1 {
		t.Fatalf("expected keySelectionCursor 1 after down, got %d", m1.keySelectionCursor)
	}

	// Toggle selection (space)
	mi, _ = m1.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	m2 := mi.(*accountsModel)
	if m2.selectedKeysToKeep[2] != false {
		t.Fatalf("expected key ID 2 to be toggled to false")
	}

	// Tab to buttons
	mi, _ = m2.Update(tea.KeyMsg{Type: tea.KeyTab})
	m3 := mi.(*accountsModel)
	if !m3.keySelectionInButtonMode {
		t.Fatalf("expected keySelectionInButtonMode true after tab")
	}

	// Press enter on Cancel (button cursor default 0)
	mi, _ = m3.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m4 := mi.(*accountsModel)
	if m4.isConfirmingKeySelection {
		t.Fatalf("expected key selection dialog closed after pressing enter on Cancel")
	}
	if m4.isConfirmingDelete {
		t.Fatalf("expected isConfirmingDelete false after cancel")
	}
}
