package db

import (
	"testing"
)

func TestGetAllAccounts_Delete_Update_Search(t *testing.T) {
	s := initTestDB(t)
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
}

func TestPublicKeys_List_Toggle_Search_Delete_Assignments(t *testing.T) {
	s := initTestDB(t)
	bdb := s.bun

	// Create accounts and keys
	accID, err := AddAccountBun(bdb, "deploy", "example", "lbl", "")
	if err != nil {
		t.Fatalf("AddAccountBun failed: %v", err)
	}

	pk1, err := AddPublicKeyAndGetModelBun(bdb, "ed25519", "AAAAB3NzaC1", "k1", false)
	if err != nil {
		t.Fatalf("AddPublicKeyAndGetModelBun failed: %v", err)
	}
	pk2, err := AddPublicKeyAndGetModelBun(bdb, "rsa", "AAAAB3NzaC2", "k2", true)
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

	// Assign pk1 to account and verify accounts for key
	if err := AssignKeyToAccountBun(bdb, pk1.ID, accID); err != nil {
		t.Fatalf("AssignKeyToAccountBun failed: %v", err)
	}
	accs, err := GetAccountsForKeyBun(bdb, pk1.ID)
	if err != nil {
		t.Fatalf("GetAccountsForKeyBun failed: %v", err)
	}
	if len(accs) == 0 {
		t.Fatalf("expected account assignment to be visible for key")
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
}
