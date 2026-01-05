// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package db

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/toeirei/keymaster/internal/model"
)

// TestIntegration_Smoke runs a minimal integration test against a real DB.
// It requires two env vars to be set by CI: INTEGRATION_DB ("postgres" or "mysql")
// and INTEGRATION_DSN (the driver DSN). If not present the test is skipped.
func TestIntegration_Smoke(t *testing.T) {
	dbType := os.Getenv("INTEGRATION_DB")
	dsn := os.Getenv("INTEGRATION_DSN")
	if dbType == "" || dsn == "" {
		t.Skip("integration DB env not set; skipping")
	}

	// Retry connecting for a short while to allow service startup in CI.
	var storeInst Store
	var err error
	for i := 0; i < 30; i++ {
		storeInst, err = NewStoreFromDSN(dbType, dsn)
		if err == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		t.Fatalf("failed to initialize store for integration DB (%s): %v", dbType, err)
	}

	// Basic operations: add account, add public key and verify duplicate detection
	id, err := storeInst.AddAccount("intuser", "int.example", "", "")
	if err != nil {
		t.Fatalf("AddAccount failed on %s: %v", dbType, err)
	}
	_ = id

	// Use KeyManager adapter for public-key CRUD operations in integration tests.
	km := &bunKeyManager{bStore: storeInst}
	if _, err := km.AddPublicKeyAndGetModel("ed25519", "intkeydata", "int-comment", false, time.Time{}); err != nil {
		t.Fatalf("AddPublicKey failed on %s: %v", dbType, err)
	}
	// duplicate should return ErrDuplicate from the Bun helper mapping
	if _, err := km.AddPublicKeyAndGetModel("ed25519", "intkeydata", "int-comment", false, time.Time{}); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("expected ErrDuplicate on duplicate AddPublicKey for %s, got: %v", dbType, err)
	}

	// Export backup
	backup, err := storeInst.ExportDataForBackup()
	if err != nil {
		t.Fatalf("ExportDataForBackup failed on %s: %v", dbType, err)
	}

	// Wipe DB via importing an empty backup (wipe-and-replace semantics)
	empty := &model.BackupData{SchemaVersion: backup.SchemaVersion}
	if err := storeInst.ImportDataFromBackup(empty); err != nil {
		t.Fatalf("ImportDataFromBackup(empty) failed on %s: %v", dbType, err)
	}
	postEmpty, err := storeInst.ExportDataForBackup()
	if err != nil {
		t.Fatalf("ExportDataForBackup after wipe failed on %s: %v", dbType, err)
	}
	if len(postEmpty.Accounts) != 0 || len(postEmpty.PublicKeys) != 0 {
		t.Fatalf("expected DB to be empty after empty import on %s, got accounts=%d keys=%d", dbType, len(postEmpty.Accounts), len(postEmpty.PublicKeys))
	}

	// Restore from original backup
	if err := storeInst.ImportDataFromBackup(backup); err != nil {
		t.Fatalf("ImportDataFromBackup restore failed on %s: %v", dbType, err)
	}
	restored, err := storeInst.ExportDataForBackup()
	if err != nil {
		t.Fatalf("ExportDataForBackup after restore failed on %s: %v", dbType, err)
	}
	if len(restored.Accounts) != len(backup.Accounts) || len(restored.PublicKeys) != len(backup.PublicKeys) {
		t.Fatalf("restore mismatch on %s: want accounts=%d keys=%d got accounts=%d keys=%d", dbType, len(backup.Accounts), len(backup.PublicKeys), len(restored.Accounts), len(restored.PublicKeys))
	}
}
