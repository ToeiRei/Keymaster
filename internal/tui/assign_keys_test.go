package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/i18n"
	"github.com/toeirei/keymaster/internal/model"
)

func TestAssignKeys_FilteringAndAccountListView(t *testing.T) {
	i18n.Init("en")
	m := &assignKeysModel{}
	m.accounts = []model.Account{{ID: 1, Username: "alice", Hostname: "h1"}, {ID: 2, Username: "bob", Hostname: "h2"}}
	m.accountCursor = 0
	content := m.accountListViewContent()
	if !strings.Contains(content, "alice") {
		t.Fatalf("expected 'alice' in account list content, got: %q", content)
	}

	// filtering
	m.accountFilter = "bob"
	filtered := m.filteredAccounts()
	if len(filtered) != 1 || filtered[0].Username != "bob" {
		t.Fatalf("expected filtered accounts to contain 'bob', got: %v", filtered)
	}
}

func TestAssignKeys_AssignAndUnassign_DB(t *testing.T) {
	i18n.Init("en")
	_ = initTestDB()

	// Create account and keys
	acctID, err := db.AddAccount("akuser", "akhost", "lbl", "")
	if err != nil {
		t.Fatalf("AddAccount failed: %v", err)
	}
	k1, err := db.AddPublicKeyAndGetModel("ssh-ed25519", "AAAAB3Nza...1", "k-one", false)
	if err != nil || k1 == nil {
		t.Fatalf("AddPublicKeyAndGetModel k1 failed: %v %v", err, k1)
	}
	k2, err := db.AddPublicKeyAndGetModel("ssh-rsa", "AAAAB3Nza...2", "k-two", false)
	if err != nil || k2 == nil {
		t.Fatalf("AddPublicKeyAndGetModel k2 failed: %v %v", err, k2)
	}

	// Assign k1 to account via DB to simulate pre-existing assignment
	if err := db.AssignKeyToAccount(k1.ID, acctID); err != nil {
		t.Fatalf("AssignKeyToAccount failed: %v", err)
	}

	// Build model from DB
	m := newAssignKeysModel()
	if m.err != nil {
		t.Fatalf("newAssignKeysModel error: %v", m.err)
	}

	// Find the account index in m.accounts
	idx := -1
	for i, a := range m.accounts {
		if a.ID == acctID {
			idx = i
			break
		}
	}
	if idx < 0 {
		t.Fatalf("account %d not found in model.accounts", acctID)
	}

	// Select the account (simulate pressing Enter)
	m.accountCursor = idx
	mm, _ := m.updateAccountSelection(tea.KeyMsg{Type: tea.KeyEnter})
	m2 := mm.(*assignKeysModel)
	if m2.state != assignStateSelectKeys {
		t.Fatalf("expected state assignStateSelectKeys after selecting account")
	}

	// Ensure assignedKeys includes k1
	if _, ok := m2.assignedKeys[k1.ID]; !ok {
		t.Fatalf("expected k1 to be in assignedKeys map")
	}
	// Diagnostic: ensure k2 is not already assigned
	if _, ok := m2.assignedKeys[k2.ID]; ok {
		t.Fatalf("did not expect k2 to be pre-assigned to account; assignedKeys=%v", m2.assignedKeys)
	}

	// Find index of k2 in filteredKeys
	fk := m2.filteredKeys()
	k2idx := -1
	for i, k := range fk {
		if k.ID == k2.ID {
			k2idx = i
			break
		}
	}
	if k2idx < 0 {
		t.Fatalf("k2 not present in filtered keys")
	}

	// Move cursor to k2 and press space to assign
	m2.keyCursor = k2idx
	if fk[k2idx].ID != k2.ID {
		t.Fatalf("expected filtered key at index to be k2 (id %d), got id %d", k2.ID, fk[k2idx].ID)
	}
	mm2, _ := m2.updateKeySelection(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	m3 := mm2.(*assignKeysModel)
	if _, ok := m3.assignedKeys[k2.ID]; !ok {
		t.Fatalf("expected k2 to be assigned after space; status=%q err=%v", m3.status, m3.err)
	}

	// Press space again to unassign k2
	mm3, _ := m3.updateKeySelection(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	m4 := mm3.(*assignKeysModel)
	if _, ok := m4.assignedKeys[k2.ID]; ok {
		t.Fatalf("expected k2 to be unassigned after second space; status=%q err=%v", m4.status, m4.err)
	}
}
