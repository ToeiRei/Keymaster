// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package tui

import (
	"testing"

	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/testutil"
)

func TestTuiLogAction_UsesDefaultWriter(t *testing.T) {
	fake := &testutil.FakeAuditWriter{}
	db.SetDefaultAuditWriter(fake)
	defer db.ClearDefaultAuditWriter()

	// Ensure package override is cleared to force fallback to DefaultAuditWriter
	ClearAuditWriter()

	if err := logAction("TEST_ACTION", "details"); err != nil {
		t.Fatalf("logAction returned error: %v", err)
	}

	if len(fake.Calls) != 1 {
		t.Fatalf("expected 1 audit call, got %d", len(fake.Calls))
	}
	if fake.Calls[0][0] != "TEST_ACTION" || fake.Calls[0][1] != "details" {
		t.Fatalf("unexpected audit call: %#v", fake.Calls[0])
	}
}

func TestTuiLogAction_UsesPackageOverride(t *testing.T) {
	fake := &testutil.FakeAuditWriter{}
	SetAuditWriter(fake)
	defer ClearAuditWriter()

	// Ensure global default is cleared so we exercise the package override path
	db.ClearDefaultAuditWriter()

	if err := logAction("PKG_ACTION", "pkg details"); err != nil {
		t.Fatalf("logAction returned error: %v", err)
	}

	if len(fake.Calls) != 1 {
		t.Fatalf("expected 1 audit call, got %d", len(fake.Calls))
	}
	if fake.Calls[0][0] != "PKG_ACTION" || fake.Calls[0][1] != "pkg details" {
		t.Fatalf("unexpected audit call: %#v", fake.Calls[0])
	}
}

