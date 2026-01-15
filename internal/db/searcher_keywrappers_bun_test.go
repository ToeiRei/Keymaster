// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package db

import (
	"testing"
	"time"
)

func TestBunKeyWrappers_GetAllAndQuery(t *testing.T) {
	WithTestStore(t, func(s *SqliteStore) {
		bdb := s.bun

		_, err := AddPublicKeyAndGetModelBun(bdb, "ssh-ed25519", "D1", "k1", false, time.Time{})
		if err != nil {
			t.Fatalf("AddPublicKeyAndGetModelBun failed: %v", err)
		}
		_, err = AddPublicKeyAndGetModelBun(bdb, "ssh-rsa", "D2", "k2", true, time.Time{})
		if err != nil {
			t.Fatalf("AddPublicKeyAndGetModelBun failed: %v", err)
		}

		km := DefaultKeyManager()
		if km == nil {
			t.Fatalf("DefaultKeyManager returned nil")
		}

		all, err := km.GetAllPublicKeys()
		if err != nil {
			t.Fatalf("GetAllPublicKeys failed: %v", err)
		}
		if len(all) < 2 {
			t.Fatalf("expected at least 2 public keys, got %d", len(all))
		}

		p, err := km.GetPublicKeyByComment("k1")
		if err != nil {
			t.Fatalf("GetPublicKeyByComment failed: %v", err)
		}
		if p == nil || p.Comment != "k1" {
			t.Fatalf("unexpected GetPublicKeyByComment: %+v", p)
		}

		globals, err := km.GetGlobalPublicKeys()
		if err != nil {
			t.Fatalf("GetGlobalPublicKeys failed: %v", err)
		}
		found := false
		for _, g := range globals {
			if g.Comment == "k2" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected global key k2 present, globals: %+v", globals)
		}

		ks := DefaultKeySearcher()
		if ks == nil {
			t.Fatalf("DefaultKeySearcher returned nil")
		}
		res, err := ks.SearchPublicKeys("k1")
		if err != nil {
			t.Fatalf("SearchPublicKeys failed: %v", err)
		}
		if len(res) == 0 || res[0].Comment != "k1" {
			t.Fatalf("unexpected SearchPublicKeys result: %+v", res)
		}
	})
}

