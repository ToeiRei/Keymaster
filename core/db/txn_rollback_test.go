// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package db

import (
	"context"
	"errors"
	"testing"

	"github.com/toeirei/keymaster/internal/core/model"
	"github.com/uptrace/bun"
)

// TestWithTxRollback ensures that a returned error from the WithTx callback causes a rollback.
func TestWithTxRollback(t *testing.T) {
	WithTestStore(t, func(s *BunStore) {
		bdb := s.BunDB()
		ctx := context.Background()

		// Ensure baseline: zero or known count
		var before int
		if err := QueryRawInto(ctx, bdb, &before, "SELECT COUNT(id) FROM accounts"); err != nil {
			t.Fatalf("QueryRawInto baseline: %v", err)
		}

		// Run a transaction that inserts and then returns an error
		err := WithTx(ctx, bdb, func(ctx context.Context, tx bun.Tx) error {
			if _, err := ExecRaw(ctx, tx, "INSERT INTO accounts (username, hostname, label, tags, serial, is_active, is_dirty) VALUES (?, ?, ?, ?, ?, ?, ?)", "txuser", "txhost", "lbl", "", 1, true, false); err != nil {
				return err
			}
			return errors.New("forced rollback")
		})
		if err == nil {
			t.Fatalf("expected error from WithTx callback")
		}

		// Verify count unchanged
		var after int
		if err := QueryRawInto(ctx, bdb, &after, "SELECT COUNT(id) FROM accounts"); err != nil {
			t.Fatalf("QueryRawInto after: %v", err)
		}
		if after != before {
			t.Fatalf("expected rollback, account count changed %d -> %d", before, after)
		}
	})
}

// TestImportDataFromBackupRollback forces a failure during ImportDataFromBackupBun
// by providing duplicate primary keys in KnownHosts so the middle of the import
// will error; we verify the pre-existing data remains present (rollback occurred).
func TestImportDataFromBackupRollback(t *testing.T) {
	WithTestStore(t, func(s *BunStore) {
		bdb := s.BunDB()

		// Create a pre-existing account and known host that should remain after failed import
		aid, err := AddAccountBun(bdb, "pre", "prehost", "pl", "")
		if err != nil {
			t.Fatalf("AddAccountBun setup: %v", err)
		}
		if err := AddKnownHostKeyBun(bdb, "prehost.example", "origkey"); err != nil {
			t.Fatalf("AddKnownHostKeyBun setup: %v", err)
		}

		// Prepare backup with duplicate KnownHosts to trigger primary-key constraint error
		backup := &model.BackupData{SchemaVersion: 1}
		backup.Accounts = []model.Account{}
		backup.PublicKeys = []model.PublicKey{}
		backup.AccountKeys = []model.AccountKey{}
		backup.SystemKeys = []model.SystemKey{}
		// duplicate hostname keys should cause INSERT UNIQUE constraint failure
		backup.KnownHosts = []model.KnownHost{{Hostname: "dup", Key: "k1"}, {Hostname: "dup", Key: "k2"}}
		backup.AuditLogEntries = []model.AuditLogEntry{}
		backup.BootstrapSessions = []model.BootstrapSession{}

		// Import should return an error due to duplicate PK in KnownHosts
		if err := ImportDataFromBackupBun(bdb, backup); err == nil {
			t.Fatalf("expected ImportDataFromBackupBun to fail due to duplicate known_hosts, but it succeeded")
		}

		// Verify the pre-existing account still exists
		acc, err := GetAccountByIDBun(bdb, aid)
		if err != nil {
			t.Fatalf("GetAccountByIDBun after failed import: %v", err)
		}
		if acc == nil {
			t.Fatalf("expected pre-existing account to remain after failed import")
		}
		// Verify pre-existing known host still present
		kh, err := GetKnownHostKeyBun(bdb, "prehost.example")
		if err != nil {
			t.Fatalf("GetKnownHostKeyBun after failed import: %v", err)
		}
		if kh != "origkey" {
			t.Fatalf("expected pre-existing known host to remain, got %q", kh)
		}
	})
}
