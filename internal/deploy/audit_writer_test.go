// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package deploy

import (
	"testing"

	"github.com/toeirei/keymaster/internal/db"
)

func TestDeployLogAction_UsesDefaultWriter(t *testing.T) {
	fake := &db.FakeAuditWriter{}
	db.SetDefaultAuditWriter(fake)
	defer db.ClearDefaultAuditWriter()

	ClearAuditWriter()

	if err := logAction("DEP_TEST", "deploy details"); err != nil {
		t.Fatalf("logAction returned error: %v", err)
	}

	if len(fake.Calls) != 1 {
		t.Fatalf("expected 1 audit call, got %d", len(fake.Calls))
	}
	if fake.Calls[0][0] != "DEP_TEST" || fake.Calls[0][1] != "deploy details" {
		t.Fatalf("unexpected audit call: %#v", fake.Calls[0])
	}
}

func TestDeployLogAction_UsesPackageOverride(t *testing.T) {
	fake := &db.FakeAuditWriter{}
	SetAuditWriter(fake)
	defer ClearAuditWriter()

	db.ClearDefaultAuditWriter()

	if err := logAction("DEP_PKG", "pkg deploy"); err != nil {
		t.Fatalf("logAction returned error: %v", err)
	}

	if len(fake.Calls) != 1 {
		t.Fatalf("expected 1 audit call, got %d", len(fake.Calls))
	}
	if fake.Calls[0][0] != "DEP_PKG" || fake.Calls[0][1] != "pkg deploy" {
		t.Fatalf("unexpected audit call: %#v", fake.Calls[0])
	}
}
