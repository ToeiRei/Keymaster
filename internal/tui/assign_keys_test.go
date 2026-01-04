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
		func TestAssignKeys_ViewReflectsAssignedState(t *testing.T) {
			i18n.Init("en")

			// Prepare a model with two keys; mark one assigned
			k1 := model.PublicKey{ID: 101, Comment: "k-one", Algorithm: "ssh-ed25519"}
			k2 := model.PublicKey{ID: 102, Comment: "k-two", Algorithm: "ssh-rsa"}
			m := &assignKeysModel{}
			m.keys = []model.PublicKey{k1, k2}
			m.assignedKeys = map[int]struct{}{k1.ID: {}}
			m.keyCursor = 0

			out := m.keyListViewContent()
			checked := i18n.T("assign_keys.checkmark_checked")
			unchecked := i18n.T("assign_keys.checkmark_unchecked")

			if !strings.Contains(out, k1.Comment) || !strings.Contains(out, checked) {
				t.Fatalf("expected assigned key to show checked mark; out=%q", out)
			}
			if !strings.Contains(out, k2.Comment) || !strings.Contains(out, unchecked) {
				t.Fatalf("expected unassigned key to show unchecked mark; out=%q", out)
			}
		}

