// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package db

import (
	"testing"
	"time"
)

func TestBunDefaultManagers_KeyAndAccountFlow(t *testing.T) {
	WithTestStore(t, func(s *SqliteStore) {
		bdb := s.bun

		// Account add/delete via DefaultAccountManager
		id, err := AddAccountBun(bdb, "carol", "host3", "label-c", "")
		if err != nil {
			t.Fatalf("AddAccountBun failed: %v", err)
		}
		am := DefaultAccountManager()
		if am == nil {
			t.Fatalf("DefaultAccountManager returned nil")
		}
		if err := am.DeleteAccount(id); err != nil {
			t.Fatalf("DeleteAccount (manager) failed: %v", err)
		}
		if acc, _ := GetAccountByIDBun(bdb, id); acc != nil {
			t.Fatalf("expected account deleted, still present: %+v", acc)
		}

		// Public key and assignments via DefaultKeyManager
		pk, err := AddPublicKeyAndGetModelBun(bdb, "ssh-ed25519", "DATA", "tkey", false, time.Time{})
		if err != nil {
			t.Fatalf("AddPublicKeyAndGetModelBun failed: %v", err)
		}

		// create an account and assign
		aid, err := AddAccountBun(bdb, "dave", "host4", "label-d", "")
		if err != nil {
			t.Fatalf("AddAccountBun failed: %v", err)
		}
		if err := AssignKeyToAccountBun(bdb, pk.ID, aid); err != nil {
			t.Fatalf("AssignKeyToAccountBun failed: %v", err)
		}

		km := DefaultKeyManager()
		if km == nil {
			t.Fatalf("DefaultKeyManager returned nil")
		}

		// Get accounts for key
		accs, err := km.GetAccountsForKey(pk.ID)
		if err != nil {
			t.Fatalf("GetAccountsForKey failed: %v", err)
		}
		if len(accs) != 1 || accs[0].ID != aid {
			t.Fatalf("unexpected accounts for key: %+v", accs)
		}

		// Toggle global flag and verify via GetGlobalPublicKeysBun
		if err := km.TogglePublicKeyGlobal(pk.ID); err != nil {
			t.Fatalf("TogglePublicKeyGlobal failed: %v", err)
		}
		globals, err := GetGlobalPublicKeysBun(bdb)
		if err != nil {
			t.Fatalf("GetGlobalPublicKeysBun failed: %v", err)
		}
		found := false
		for _, g := range globals {
			if g.Comment == pk.Comment {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected key comment %q in global keys, globals: %+v", pk.Comment, globals)
		}

		// Delete public key via manager
		if err := km.DeletePublicKey(pk.ID); err != nil {
			t.Fatalf("DeletePublicKey (manager) failed: %v", err)
		}
		if pk2, _ := GetPublicKeyByIDBun(bdb, pk.ID); pk2 != nil {
			t.Fatalf("expected public key deleted, still present: %+v", pk2)
		}
	})
}
