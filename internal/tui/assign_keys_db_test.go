// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/i18n"
)

// TestAssignKeys_AssignAndUnassign performs a DB-driven integration-style test
// that creates an account and keys, builds the assignKeysModel, selects the
// account, assigns a key, verifies DB state, then unassigns and verifies again.
func TestAssignKeys_AssignAndUnassign(t *testing.T) {
	i18n.Init("en")
	_ = initTestDB()

	// Create account and keys
	mgr := db.DefaultAccountManager()
	if mgr == nil {
		t.Fatalf("no account manager available")
	}
	acctID, err := mgr.AddAccount("akuser", "akhost", "lbl", "")
	if err != nil {
		t.Fatalf("AddAccount failed: %v", err)
	}
	km := db.DefaultKeyManager()
	if km == nil {
		t.Fatalf("no key manager available")
	}
	k1, err := km.AddPublicKeyAndGetModel("ssh-ed25519", "AAAAB3Nza...1", "k-one", false, time.Time{})
	if err != nil || k1 == nil {
		t.Fatalf("AddPublicKeyAndGetModel k1 failed: %v %v", err, k1)
	}

	// Build model and simulate selecting the account (enter)
	m := newAssignKeysModel()
	if m == nil || m.err != nil {
		t.Fatalf("newAssignKeysModel failed: %v", m.err)
	}

	// Ensure accounts include the created one
	found := false
	for _, a := range m.accounts {
		if a.ID == acctID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("created account not present in model accounts: %v", m.accounts)
	}

	// Simulate pressing enter on the account to select it. We need to ensure the
	// accountCursor is at the correct index in filteredAccounts.
	for i, a := range m.filteredAccounts() {
		if a.ID == acctID {
			m.accountCursor = i
			break
		}
	}

	// Trigger selection
	m2, _ := m.updateAccountSelection(tea.KeyMsg{Type: tea.KeyEnter})
	m = m2.(*assignKeysModel)
	_ = m

	if m.selectedAccount.ID != acctID {
		t.Fatalf("expected selected account %d, got %d", acctID, m.selectedAccount.ID)
	}

	// Now simulate toggling assignment (space) on the first key in filteredKeys
	if len(m.filteredKeys()) == 0 {
		t.Fatalf("no keys available to assign")
	}
	// ensure keyCursor points to first key
	m.keyCursor = 0
	// assign
	m2, _ = m.updateKeySelection(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	m = m2.(*assignKeysModel)
	_ = m

	// Verify in DB that the key is assigned
	km2 := db.DefaultKeyManager()
	if km2 == nil {
		t.Fatalf("no key manager available")
	}
	assigned, err := km2.GetKeysForAccount(acctID)
	if err != nil {
		t.Fatalf("GetKeysForAccount failed: %v", err)
	}
	if len(assigned) == 0 {
		t.Fatalf("expected key to be assigned in DB after assignment")
	}

	// Now unassign using space again (on the same key)
	m2, _ = m.updateKeySelection(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	m = m2.(*assignKeysModel)
	_ = m

	assigned, err = km2.GetKeysForAccount(acctID)
	if err != nil {
		t.Fatalf("GetKeysForAccount failed: %v", err)
	}
	if len(assigned) != 0 {
		t.Fatalf("expected no keys assigned after unassign, got: %v", assigned)
	}
}
