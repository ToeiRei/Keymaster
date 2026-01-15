// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package db

import (
	"context"
	"testing"

	"github.com/uptrace/bun"
)

func TestBeginTx_WithTx_IsInitialized_GetAllAuditLogEntries(t *testing.T) {
	// Preserve original store and restore at end
	orig := store
	defer func() { store = orig }()

	// Ensure uninitialized state is reported when store is nil
	store = nil
	if IsInitialized() {
		t.Fatal("expected IsInitialized to be false when store is nil")
	}

	// Initialize a test DB
	WithTestStore(t, func(s *SqliteStore) {
		bdb := s.bun

		if !IsInitialized() {
			t.Fatal("expected IsInitialized to be true after InitDB")
		}

		// Test BeginTx returns a usable transaction
		ctx := context.Background()
		tx, err := BeginTx(ctx, bdb, nil)
		if err != nil {
			t.Fatalf("BeginTx failed: %v", err)
		}
		// tx value is returned; commit above verifies usability
		if err := tx.Commit(); err != nil {
			t.Fatalf("tx.Commit failed: %v", err)
		}

		// Test WithTx commits on success and allows ExecRaw usage
		if err := WithTx(ctx, bdb, func(ctx context.Context, tx bun.Tx) error {
			_, err := ExecRaw(ctx, tx, "INSERT INTO audit_log (username, action, details) VALUES (?, ?, ?)", "tester", "act", "d")
			return err
		}); err != nil {
			t.Fatalf("WithTx failed: %v", err)
		}

		// Verify wrapper GetAllAuditLogEntries returns the inserted row
		entries, err := GetAllAuditLogEntries()
		if err != nil {
			t.Fatalf("GetAllAuditLogEntries failed: %v", err)
		}
		if len(entries) == 0 {
			t.Fatalf("expected at least one audit log entry, got 0")
		}
	})
}
