// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package db

import (
	"testing"
	"time"

	"github.com/toeirei/keymaster/core/model"
)

// Test KnownHost get/add/replace behavior and Import/Integrate audit timestamp handling.
func TestKnownHostAndImportIntegrateTimestamps(t *testing.T) {
	WithTestStore(t, func(s *BunStore) {
		bdb := s.BunDB()

		// Get on missing host returns empty string
		v, err := GetKnownHostKeyBun(bdb, "no-such-host")
		if err != nil {
			t.Fatalf("GetKnownHostKeyBun missing: %v", err)
		}
		if v != "" {
			t.Fatalf("expected empty for missing host, got %q", v)
		}

		// Add known host and retrieve
		if err := AddKnownHostKeyBun(bdb, "example.local", "ssh-rsa AAA"); err != nil {
			t.Fatalf("AddKnownHostKeyBun: %v", err)
		}
		got, err := GetKnownHostKeyBun(bdb, "example.local")
		if err != nil {
			t.Fatalf("GetKnownHostKeyBun: %v", err)
		}
		if got != "ssh-rsa AAA" {
			t.Fatalf("unexpected known host value: %q", got)
		}

		// Prepare backup data with two audit entries: RFC3339 and non-parseable timestamp.
		now := time.Now().UTC()
		backup := &model.BackupData{SchemaVersion: 1}
		backup.Accounts = []model.Account{}
		backup.PublicKeys = []model.PublicKey{}
		backup.AccountKeys = []model.AccountKey{}
		backup.SystemKeys = []model.SystemKey{}
		backup.KnownHosts = []model.KnownHost{{Hostname: "k1", Key: "KVAL"}}
		backup.AuditLogEntries = []model.AuditLogEntry{
			{ID: 1, Timestamp: now.Format(time.RFC3339), Username: "u", Action: "a", Details: "d"},
			{ID: 2, Timestamp: "not-a-time", Username: "u2", Action: "a2", Details: "d2"},
		}
		backup.BootstrapSessions = []model.BootstrapSession{}

		// Import should succeed
		if err := ImportDataFromBackupBun(bdb, backup); err != nil {
			t.Fatalf("ImportDataFromBackupBun: %v", err)
		}

		// Verify audit entries present
		aentries, err := GetAllAuditLogEntriesBun(bdb)
		if err != nil {
			t.Fatalf("GetAllAuditLogEntriesBun: %v", err)
		}
		if len(aentries) < 2 {
			t.Fatalf("expected audit entries after import, got %d", len(aentries))
		}

		// Integrate should be idempotent (no error) when called twice
		if err := IntegrateDataFromBackupBun(bdb, backup); err != nil {
			t.Fatalf("IntegrateDataFromBackupBun first: %v", err)
		}
		if err := IntegrateDataFromBackupBun(bdb, backup); err != nil {
			t.Fatalf("IntegrateDataFromBackupBun second: %v", err)
		}
	})
}
