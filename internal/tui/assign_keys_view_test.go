// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package tui

import (
	"strings"
	"testing"

	"github.com/toeirei/keymaster/internal/i18n"
	"github.com/toeirei/keymaster/internal/model"
)

func TestAssignKeys_FilterAndAccountListContent(t *testing.T) {
	i18n.Init("en")
	m := &assignKeysModel{}
	m.accounts = []model.Account{{ID: 1, Username: "alice", Hostname: "h1"}, {ID: 2, Username: "bob", Hostname: "h2"}}
	m.accountCursor = 0
	// No filter => both
	out := m.accountListViewContent()
	if !strings.Contains(out, "alice") || !strings.Contains(out, "bob") {
		t.Fatalf("expected both accounts in view, got: %q", out)
	}

	m.accountFilter = "bob"
	filtered := m.filteredAccounts()
	if len(filtered) != 1 || filtered[0].Username != "bob" {
		t.Fatalf("expected filtered accounts to contain 'bob', got: %v", filtered)
	}
}

func TestAssignKeys_KeyFilteringAndCheckedMarks(t *testing.T) {
	i18n.Init("en")
	k1 := model.PublicKey{ID: 11, Comment: "k-one", Algorithm: "ssh-ed25519", IsGlobal: false}
	k2 := model.PublicKey{ID: 12, Comment: "global-k", Algorithm: "ssh-rsa", IsGlobal: true}
	m := &assignKeysModel{}
	m.keys = []model.PublicKey{k1, k2}
	m.assignedKeys = map[int]struct{}{k1.ID: {}}
	m.keyCursor = 0

	// No filter => both keys present
	out := m.keyListViewContent()
	checked := i18n.T("assign_keys.checkmark_checked")
	unchecked := i18n.T("assign_keys.checkmark_unchecked")
	if !strings.Contains(out, k1.Comment) || !strings.Contains(out, checked) {
		t.Fatalf("expected assigned key to show checked mark; out=%q", out)
	}
	if !strings.Contains(out, k2.Comment) || !strings.Contains(out, unchecked) {
		t.Fatalf("expected unassigned/global key to show unchecked mark; out=%q", out)
	}

	// Filter by comment substring
	m.keyFilter = "global"
	fk := m.filteredKeys()
	if len(fk) != 1 || fk[0].ID != k2.ID {
		t.Fatalf("expected only global key after filter, got: %v", fk)
	}
}
