// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package db

import (
	"testing"
	"time"

	"github.com/toeirei/keymaster/core/model"
)

func TestIntegrateDataFromBackup_NonDestructive(t *testing.T) {
	_ = newTestDB(t)

	// initial data
	mgr := DefaultAccountManager()
	if mgr == nil {
		t.Fatalf("no account manager available")
	}
	_, err := mgr.AddAccount("alice", "host1.example", "", "")
	if err != nil {
		t.Fatalf("AddAccount alice failed: %v", err)
	}
	km := DefaultKeyManager()
	if km == nil {
		t.Fatalf("no key manager available")
	}
	if err := km.AddPublicKey("ed25519", "AAAAB3NzaC1lZDI1NTE5AAAAIkey1", "c1", false, time.Time{}); err != nil {
		t.Fatalf("AddPublicKey c1 failed: %v", err)
	}

	before, err := ExportDataForBackup()
	if err != nil {
		t.Fatalf("ExportDataForBackup failed: %v", err)
	}

	// prepare integration backup: one duplicate account, one new account; one duplicate key, one new key
	bk := &model.BackupData{SchemaVersion: before.SchemaVersion}
	bk.Accounts = []model.Account{
		// duplicate by username+hostname should be ignored
		{ID: 999, Username: "alice", Hostname: "host1.example", Label: "", Tags: "", Serial: 0, IsActive: true},
		{ID: 1000, Username: "bob", Hostname: "host2.example", Label: "", Tags: "", Serial: 0, IsActive: true},
	}
	bk.PublicKeys = []model.PublicKey{
		{ID: 999, Algorithm: "ed25519", KeyData: "AAAAB3NzaC1lZDI1NTE5AAAAIkey1", Comment: "c1", IsGlobal: false},
		{ID: 1000, Algorithm: "ed25519", KeyData: "AAAAB3NzaC1lZDI1NTE5AAAAIkey2", Comment: "c2", IsGlobal: false},
	}

	if err := IntegrateDataFromBackup(bk); err != nil {
		t.Fatalf("IntegrateDataFromBackup failed: %v", err)
	}

	after, err := ExportDataForBackup()
	if err != nil {
		t.Fatalf("ExportDataForBackup after integrate failed: %v", err)
	}

	// Expect accounts to increase by 1 (bob added), public keys increase by 1 (c2 added)
	if len(after.Accounts) != len(before.Accounts)+1 {
		t.Fatalf("expected accounts to increase by 1: before=%d after=%d", len(before.Accounts), len(after.Accounts))
	}
	if len(after.PublicKeys) != len(before.PublicKeys)+1 {
		t.Fatalf("expected public keys to increase by 1: before=%d after=%d", len(before.PublicKeys), len(after.PublicKeys))
	}
}
