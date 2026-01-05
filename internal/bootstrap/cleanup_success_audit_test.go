// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package bootstrap

import (
	"testing"

	"github.com/toeirei/keymaster/internal/db"
)

// Test that the bootstrap package's audit wrapper records a BOOTSTRAP_HOST action
// when used with the package override and when falling back to DefaultAuditWriter.
func TestBootstrapLogAction_UsesWriters(t *testing.T) {
	// Package override path
	fake := &db.FakeAuditWriter{}
	SetAuditWriter(fake)
	defer ClearAuditWriter()

	db.ClearDefaultAuditWriter()

	if err := logAction("BOOTSTRAP_HOST", "account: alice@host, keys_deployed: 1"); err != nil {
		t.Fatalf("logAction returned error: %v", err)
	}
	if len(fake.Calls) != 1 {
		t.Fatalf("expected 1 audit call via package override, got %d", len(fake.Calls))
	}
	if fake.Calls[0][0] != "BOOTSTRAP_HOST" {
		t.Fatalf("unexpected action recorded: %s", fake.Calls[0][0])
	}

	// Default writer fallback path
	fake2 := &db.FakeAuditWriter{}
	db.SetDefaultAuditWriter(fake2)
	defer db.ClearDefaultAuditWriter()

	// Clear package override so fallback is used
	ClearAuditWriter()

	if err := logAction("BOOTSTRAP_HOST", "account: bob@host, keys_deployed: 2"); err != nil {
		t.Fatalf("logAction returned error: %v", err)
	}
	if len(fake2.Calls) != 1 {
		t.Fatalf("expected 1 audit call via default writer, got %d", len(fake2.Calls))
	}
	if fake2.Calls[0][0] != "BOOTSTRAP_HOST" {
		t.Fatalf("unexpected action recorded: %s", fake2.Calls[0][0])
	}
}

