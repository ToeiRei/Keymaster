// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package db

import (
	"testing"
	"time"

	"github.com/toeirei/keymaster/core/model"
)

func TestBackupImport_RoundTrip(t *testing.T) {
	_ = newTestDB(t)

	// Create sample data
	mgr := DefaultAccountManager()
	if mgr == nil {
		t.Fatalf("no account manager available")
	}
	a1, err := mgr.AddAccount("bob", "web1.example", "web", "")
	if err != nil {
		t.Fatalf("AddAccount failed: %v", err)
	}
	_, err = mgr.AddAccount("carol", "db1.example", "db", "")
	if err != nil {
		t.Fatalf("AddAccount failed: %v", err)
	}

	// Add public keys
	km := DefaultKeyManager()
	if km == nil {
		t.Fatalf("no key manager available")
	}
	pk, err := km.AddPublicKeyAndGetModel("ed25519", "AAAAB3NzaC1lZDI1NTE5AAAAIbackupkey", "bk-1", false, time.Time{})
	if err != nil {
		t.Fatalf("AddPublicKeyAndGetModel failed: %v", err)
	}
	if pk == nil {
		t.Fatalf("expected public key model, got nil")
	}

	// Assign key to account
	if err := km.AssignKeyToAccount(pk.ID, a1); err != nil {
		t.Fatalf("AssignKeyToAccount failed: %v", err)
	}

	// System key
	if _, err := CreateSystemKey("sys-pub", "sys-priv"); err != nil {
		t.Fatalf("CreateSystemKey failed: %v", err)
	}

	// Known host
	if err := AddKnownHostKey("host.example", "ssh-ed25519 AAAA..."); err != nil {
		t.Fatalf("AddKnownHostKey failed: %v", err)
	}

	// Audit log
	if err := LogAction("TEST_ACTION", "details"); err != nil {
		t.Fatalf("LogAction failed: %v", err)
	}

	// Bootstrap session
	expires := time.Now().Add(5 * time.Minute)
	if err := SaveBootstrapSession("bs-1", "bob", "web1.example", "", "", "tempkey", expires, "active"); err != nil {
		t.Fatalf("SaveBootstrapSession failed: %v", err)
	}

	// Export
	backup, err := ExportDataForBackup()
	if err != nil {
		t.Fatalf("ExportDataForBackup failed: %v", err)
	}

	// Wipe DB by importing an empty backup (ImportDataFromBackup performs wipe-and-replace)
	empty := &model.BackupData{SchemaVersion: backup.SchemaVersion}
	if err := ImportDataFromBackup(empty); err != nil {
		t.Fatalf("ImportDataFromBackup(empty) failed: %v", err)
	}

	// Ensure DB is empty
	postEmpty, err := ExportDataForBackup()
	if err != nil {
		t.Fatalf("ExportDataForBackup after wipe failed: %v", err)
	}
	if len(postEmpty.Accounts) != 0 || len(postEmpty.PublicKeys) != 0 {
		t.Fatalf("expected DB to be empty after empty import, got accounts=%d keys=%d", len(postEmpty.Accounts), len(postEmpty.PublicKeys))
	}

	// Restore from backup
	if err := ImportDataFromBackup(backup); err != nil {
		t.Fatalf("ImportDataFromBackup failed: %v", err)
	}

	// Re-export and compare counts
	restored, err := ExportDataForBackup()
	if err != nil {
		t.Fatalf("ExportDataForBackup after restore failed: %v", err)
	}
	if len(restored.Accounts) != len(backup.Accounts) {
		t.Fatalf("account count mismatch after restore: want=%d got=%d", len(backup.Accounts), len(restored.Accounts))
	}
	if len(restored.PublicKeys) != len(backup.PublicKeys) {
		t.Fatalf("public key count mismatch after restore: want=%d got=%d", len(backup.PublicKeys), len(restored.PublicKeys))
	}
	if len(restored.AccountKeys) != len(backup.AccountKeys) {
		t.Fatalf("account_keys count mismatch after restore: want=%d got=%d", len(backup.AccountKeys), len(restored.AccountKeys))
	}
}
