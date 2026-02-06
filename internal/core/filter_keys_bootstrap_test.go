// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core

import (
	"testing"

	"github.com/toeirei/keymaster/internal/core/model"
)

func TestFilterKeysForBootstrap(t *testing.T) {
	all := []model.PublicKey{
		{ID: 1, KeyData: "sysdata-abc", Comment: "Keymaster System Key"},
		{ID: 2, KeyData: "user1", Comment: "alice", IsGlobal: false},
		{ID: 3, KeyData: "user2", Comment: "bob", IsGlobal: true},
		{ID: 4, KeyData: "other", Comment: "Keymaster System Key old", IsGlobal: false},
	}

	user, global := FilterKeysForBootstrap(all, "sysdata-abc")
	if len(user) != 1 || user[0].ID != 2 {
		t.Fatalf("expected user-selectable id 2 when system key data present, got %v", user)
	}
	if len(global) != 1 || global[0].ID != 3 {
		t.Fatalf("expected only global key id 3 in global list, got %v", global)
	}

	// When systemKeyData empty, comment filter still excludes system-key comments
	user2, global2 := FilterKeysForBootstrap(all, "")
	if len(user2) != 1 || user2[0].ID != 2 {
		t.Fatalf("expected user-selectable id 2, got %v", user2)
	}
	if len(global2) != 1 || global2[0].ID != 3 {
		t.Fatalf("expected global id 3, got %v", global2)
	}
}
