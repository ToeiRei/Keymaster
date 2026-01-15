// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/internal/i18n"
	"github.com/toeirei/keymaster/internal/ui"
)

// TestExpiryToggleEpochZero verifies that pressing 'e' toggles a key's expiry
// between cleared (zero time) and Unix epoch 1970-01-01 (deactivated sentinel).
func TestExpiryToggleEpochZero(t *testing.T) {
	initTestDBT(t)
	i18n.Init("en")

	km := ui.DefaultKeyManager()
	if km == nil {
		t.Fatal("no key manager")
	}

	k, err := km.AddPublicKeyAndGetModel("ssh-ed25519", "AAAAB3...T", "toggle-me", false, time.Time{})
	if err != nil {
		t.Fatalf("add key: %v", err)
	}

	m := newPublicKeysModelWithSearcher(nil)

	// Ensure list has at least one and cursor is at 0
	if len(m.displayedKeys) == 0 {
		t.Fatal("expected displayed keys")
	}

	// Move cursor to the added key so toggle targets it even when other keys exist
	for i, dk := range m.displayedKeys {
		if dk.ID == k.ID {
			m.cursor = i
			break
		}
	}

	// Press 'e' to deactivate (set to epoch 0)
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})

	// (no debug logs) ensure tests remain deterministic

	keys, err := km.GetAllPublicKeys()
	if err != nil {
		t.Fatalf("get keys: %v", err)
	}
	var found bool
	for _, kk := range keys {
		if kk.ID == k.ID {
			found = true
			if kk.ExpiresAt.IsZero() {
				t.Fatalf("expected expires_at to be set to epoch, got zero")
			}
			if !kk.ExpiresAt.Equal(time.Unix(0, 0).UTC()) {
				t.Fatalf("expected epoch 0, got %v", kk.ExpiresAt)
			}
		}
	}
	if !found {
		t.Fatalf("added key not found after toggle")
	}

	// Press 'e' again to reactivate (clear expiration)
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})

	keys, err = km.GetAllPublicKeys()
	if err != nil {
		t.Fatalf("get keys: %v", err)
	}
	for _, kk := range keys {
		if kk.ID == k.ID {
			if !kk.ExpiresAt.IsZero() {
				t.Fatalf("expected expires_at to be cleared, got %v", kk.ExpiresAt)
			}
		}
	}
}

// TestExpiryModalFlow ensures opening the expiry modal with 'x', entering a
// valid date, and pressing Enter updates the key's expiration in the DB.
func TestExpiryModalFlow(t *testing.T) {
	initTestDBT(t)
	i18n.Init("en")

	km := ui.DefaultKeyManager()
	if km == nil {
		t.Fatal("no key manager")
	}

	k, err := km.AddPublicKeyAndGetModel("ssh-ed25519", "AAAAB3...U", "expire-me", false, time.Time{})
	if err != nil {
		t.Fatalf("add key: %v", err)
	}

	m := newPublicKeysModelWithSearcher(nil)

	// Open expiry modal
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if !m.isSettingExpiry {
		t.Fatalf("expected expiry modal to be open")
	}

	// Type a date (YYYY-MM-DD)
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2', '0', '2', '6', '-', '0', '1', '-', '1', '0'}})

	// Submit with Enter
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	keys, err := km.GetAllPublicKeys()
	if err != nil {
		t.Fatalf("get keys: %v", err)
	}
	var found bool
	for _, kk := range keys {
		if kk.ID == k.ID {
			found = true
			want := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
			if kk.ExpiresAt.IsZero() {
				t.Fatalf("expected expires_at to be set, got zero")
			}
			if !kk.ExpiresAt.Equal(want) {
				t.Fatalf("expected expires_at %v, got %v", want, kk.ExpiresAt)
			}
		}
	}
	if !found {
		t.Fatalf("added key not found after expiry flow")
	}
}
