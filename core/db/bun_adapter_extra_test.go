// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package db

import (
	"strings"
	"testing"
	"time"
)

// Test GetActiveSystemKeyBun / RotateSystemKeyBun behavior and account dirty marking.
func TestSystemKeyRotateAndActive(t *testing.T) {
	WithTestStore(t, func(s *BunStore) {
		bdb := s.BunDB()

		// Initially no system keys
		sk, err := GetActiveSystemKeyBun(bdb)
		if err != nil {
			t.Fatalf("GetActiveSystemKeyBun initial: %v", err)
		}
		if sk != nil {
			t.Fatalf("expected no active system key initially")
		}

		// Ensure we have an account to observe dirty flag
		aid, err := AddAccountBun(bdb, "u2", "h2", "lbl2", "")
		if err != nil {
			t.Fatalf("AddAccountBun: %v", err)
		}

		// Create initial system key
		sserial, err := CreateSystemKeyBun(bdb, "pub1", "priv1")
		if err != nil {
			t.Fatalf("CreateSystemKeyBun: %v", err)
		}
		if sserial <= 0 {
			t.Fatalf("invalid serial from CreateSystemKeyBun: %d", sserial)
		}

		// Active key should be present
		sk2, err := GetActiveSystemKeyBun(bdb)
		if err != nil {
			t.Fatalf("GetActiveSystemKeyBun after create: %v", err)
		}
		if sk2 == nil || sk2.Serial != sserial {
			t.Fatalf("unexpected active key: %+v", sk2)
		}

		// Rotate key
		newSerial, err := RotateSystemKeyBun(bdb, "pub2", "priv2")
		if err != nil {
			t.Fatalf("RotateSystemKeyBun: %v", err)
		}
		if newSerial == sserial {
			t.Fatalf("expected new serial different from previous")
		}

		// Active key should reflect new serial
		sk3, err := GetActiveSystemKeyBun(bdb)
		if err != nil {
			t.Fatalf("GetActiveSystemKeyBun after rotate: %v", err)
		}
		if sk3 == nil || sk3.Serial != newSerial {
			t.Fatalf("active key did not update: %+v", sk3)
		}

		// Accounts should have been marked dirty by rotation
		acc, err := GetAccountByIDBun(bdb, aid)
		if err != nil {
			t.Fatalf("GetAccountByIDBun: %v", err)
		}
		if acc == nil || !acc.IsDirty {
			t.Fatalf("expected account dirty after rotate: %+v", acc)
		}
	})
}

// Test AddPublicKeyAndGetModelBun returns nil on duplicate and marks accounts dirty when global.
func TestAddPublicKeyAndImportIntegrate(t *testing.T) {
	WithTestStore(t, func(s *BunStore) {
		bdb := s.BunDB()

		// Add a public key via AddPublicKeyAndGetModelBun
		pk, err := AddPublicKeyAndGetModelBun(bdb, "ssh-ed25519", "dataX", "dup-key", false, time.Time{})
		if err != nil {
			t.Fatalf("AddPublicKeyAndGetModelBun: %v", err)
		}
		if pk == nil {
			t.Fatalf("expected pk model on first insert")
		}

		// Duplicate insert should return (nil, nil)
		dup, err := AddPublicKeyAndGetModelBun(bdb, "ssh-ed25519", "dataX", "dup-key", false, time.Time{})
		if err != nil {
			t.Fatalf("duplicate AddPublicKeyAndGetModelBun error: %v", err)
		}
		if dup != nil {
			t.Fatalf("expected nil for duplicate insert")
		}

		// Make the key global and ensure accounts are marked dirty when toggled/expiry set
		// Add an account
		aid, err := AddAccountBun(bdb, "u3", "h3", "lbl3", "")
		if err != nil {
			t.Fatalf("AddAccountBun: %v", err)
		}

		// Mark the existing key global using TogglePublicKeyGlobalBun
		if err := TogglePublicKeyGlobalBun(bdb, pk.ID); err != nil {
			t.Fatalf("TogglePublicKeyGlobalBun: %v", err)
		}
		acc, _ := GetAccountByIDBun(bdb, aid)
		if acc == nil || !acc.IsDirty {
			t.Fatalf("expected account dirty after toggling global: %+v", acc)
		}

		// Export backup and then import into same DB (round-trip)
		backup, err := ExportDataForBackupBun(bdb)
		if err != nil {
			t.Fatalf("ExportDataForBackupBun: %v", err)
		}
		if err := ImportDataFromBackupBun(bdb, backup); err != nil {
			t.Fatalf("ImportDataForBackupBun: %v", err)
		}

		// Integrate should be idempotent (no error)
		if err := IntegrateDataFromBackupBun(bdb, backup); err != nil {
			t.Fatalf("IntegrateDataFromBackupBun: %v", err)
		}
	})
}

func TestGetAllAccounts_Delete_Update_Search(t *testing.T) {
	WithTestStore(t, func(s *BunStore) {
		bdb := s.bun

		// Create accounts
		id1, err := AddAccountBun(bdb, "alice", "host1", "label-a", "t1")
		if err != nil {
			t.Fatalf("AddAccountBun failed: %v", err)
		}
		id2, err := AddAccountBun(bdb, "bob", "host2", "label-b", "t2")
		if err != nil {
			t.Fatalf("AddAccountBun failed: %v", err)
		}

		all, err := GetAllAccountsBun(bdb)
		if err != nil {
			t.Fatalf("GetAllAccountsBun failed: %v", err)
		}
		if len(all) < 2 {
			t.Fatalf("expected at least 2 accounts, got %d", len(all))
		}

		// Update label/hostname/tags
		if err := UpdateAccountLabelBun(bdb, id1, "new-label"); err != nil {
			t.Fatalf("UpdateAccountLabelBun failed: %v", err)
		}
		if err := UpdateAccountHostnameBun(bdb, id1, "new-host"); err != nil {
			t.Fatalf("UpdateAccountHostnameBun failed: %v", err)
		}
		if err := UpdateAccountTagsBun(bdb, id1, "new:tag"); err != nil {
			t.Fatalf("UpdateAccountTagsBun failed: %v", err)
		}
		a, err := GetAccountByIDBun(bdb, id1)
		if err != nil {
			t.Fatalf("GetAccountByIDBun failed: %v", err)
		}
		if a == nil || a.Label != "new-label" || a.Hostname != "new-host" || a.Tags != "new:tag" {
			t.Fatalf("account update did not persist: %+v", a)
		}

		// Search
		res, err := SearchAccountsBun(bdb, "alice")
		if err != nil {
			t.Fatalf("SearchAccountsBun failed: %v", err)
		}
		if len(res) == 0 {
			t.Fatalf("expected search to find account alice")
		}

		// Delete one
		if err := DeleteAccountBun(bdb, id2); err != nil {
			t.Fatalf("DeleteAccountBun failed: %v", err)
		}
		got, err := GetAccountByIDBun(bdb, id2)
		if err != nil {
			t.Fatalf("GetAccountByIDBun after delete failed: %v", err)
		}
		if got != nil {
			t.Fatalf("expected account to be deleted, still present: %+v", got)
		}
	})
}

func TestPublicKeys_List_Toggle_Search_Delete_Assignments(t *testing.T) {
	WithTestStore(t, func(s *BunStore) {
		bdb := s.bun

		// Create accounts and keys
		accID, err := AddAccountBun(bdb, "deploy", "example", "lbl", "")
		if err != nil {
			t.Fatalf("AddAccountBun failed: %v", err)
		}

		pk1, err := AddPublicKeyAndGetModelBun(bdb, "ed25519", "AAAAB3NzaC1", "k1", false, time.Time{})
		if err != nil {
			t.Fatalf("AddPublicKeyAndGetModelBun failed: %v", err)
		}
		pk2, err := AddPublicKeyAndGetModelBun(bdb, "rsa", "AAAAB3NzaC2", "k2", true, time.Time{})
		if err != nil {
			t.Fatalf("AddPublicKeyAndGetModelBun failed: %v", err)
		}

		all, err := GetAllPublicKeysBun(bdb)
		if err != nil {
			t.Fatalf("GetAllPublicKeysBun failed: %v", err)
		}
		if len(all) < 2 {
			t.Fatalf("expected at least 2 public keys, got %d", len(all))
		}

		// Search public keys
		found, err := SearchPublicKeysBun(bdb, "k1")
		if err != nil {
			t.Fatalf("SearchPublicKeysBun failed: %v", err)
		}
		if len(found) == 0 {
			t.Fatalf("expected search to find k1")
		}

		// Toggle global flag for pk1
		if err := TogglePublicKeyGlobalBun(bdb, pk1.ID); err != nil {
			t.Fatalf("TogglePublicKeyGlobalBun failed: %v", err)
		}
		globals, err := GetGlobalPublicKeysBun(bdb)
		if err != nil {
			t.Fatalf("GetGlobalPublicKeysBun failed: %v", err)
		}
		// After toggling, pk1 should be global (pk2 was already global)
		if len(globals) < 2 {
			t.Fatalf("expected at least 2 global keys, got %d", len(globals))
		}

		// Try to assign pk1 (now global) to account - should fail
		err = AssignKeyToAccountBun(bdb, pk1.ID, accID)
		if err == nil {
			t.Fatal("AssignKeyToAccountBun should fail when trying to assign global key")
		}
		if !strings.Contains(err.Error(), "cannot assign global key") {
			t.Fatalf("expected error about global key assignment, got: %v", err)
		}

		// Delete public key pk2
		if err := DeletePublicKeyBun(bdb, pk2.ID); err != nil {
			t.Fatalf("DeletePublicKeyBun failed: %v", err)
		}
		got, err := GetPublicKeyByIDBun(bdb, pk2.ID)
		if err != nil {
			t.Fatalf("GetPublicKeyByIDBun after delete failed: %v", err)
		}
		if got != nil {
			t.Fatalf("expected public key to be deleted, still present: %+v", got)
		}
	})
}
