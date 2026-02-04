// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/toeirei/keymaster/internal/core/db"
	"github.com/toeirei/keymaster/internal/model"
)

func TestRebuildDisplayedKeys_LocalFilter(t *testing.T) {
	initTestDBT(t)

	km := db.DefaultKeyManager()
	if km == nil {
		t.Fatal("no key manager")
	}

	if _, err := km.AddPublicKeyAndGetModel("ssh-ed25519", "AAAAB3...A", "alpha", false, time.Time{}); err != nil {
		t.Fatalf("add key alpha: %v", err)
	}
	if _, err := km.AddPublicKeyAndGetModel("ssh-ed25519", "AAAAB3...B", "beta", false, time.Time{}); err != nil {
		t.Fatalf("add key beta: %v", err)
	}

	m := newPublicKeysModelWithSearcher(nil)
	if len(m.displayedKeys) < 2 {
		t.Fatalf("expected at least 2 keys in displayedKeys, got %d", len(m.displayedKeys))
	}

	m.filter = "alpha"
	m.rebuildDisplayedKeys()
	if len(m.displayedKeys) != 1 {
		t.Fatalf("expected 1 key after filtering, got %d", len(m.displayedKeys))
	}
	out := m.listContentView()
	if !strings.Contains(out, "alpha") {
		t.Fatalf("listContentView did not contain filtered key 'alpha': %q", out)
	}
}

func TestEnsureCursorInView_Scrolls(t *testing.T) {
	var m publicKeysModel
	m.viewport = viewport.New(0, 0)
	// simulate 10 items
	m.displayedKeys = make([]model.PublicKey, 10)
	m.viewport.Height = 3
	m.cursor = 9
	m.viewport.YOffset = 0

	m.ensureCursorInView()
	want := 9 - m.viewport.Height + 1
	if m.viewport.YOffset != want {
		t.Fatalf("expected YOffset %d, got %d", want, m.viewport.YOffset)
	}
}

func TestViewConfirmation_IncludesComment(t *testing.T) {
	var m publicKeysModel
	m.width = 80
	m.height = 24
	m.isConfirmingDelete = true
	m.confirmCursor = 1
	m.keyToDelete = model.PublicKey{Comment: "to-delete"}

	out := m.viewConfirmation()
	if !strings.Contains(out, "to-delete") {
		t.Fatalf("expected confirmation view to include key comment, got: %s", out)
	}
}
