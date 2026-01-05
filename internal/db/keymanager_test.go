// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package db

import (
	"testing"
	"time"
)

// TestKeyManager_AssignUnassign_Audit verifies that the bunKeyManager correctly
// assigns and unassigns keys using the Bun helpers and records audit log
// entries via the Store's LogAction implementation.
func TestKeyManager_AssignUnassign_Audit(t *testing.T) {
	dsn := "file::memory:?cache=shared"
	s, err := NewStoreFromDSN("sqlite", dsn)
	if err != nil {
		t.Fatalf("NewStoreFromDSN error: %v", err)
	}
	// Ensure DB is closed after test
	defer func() { _ = s.BunDB().DB.Close() }()

	// Create an account
	acctID, err := s.AddAccount("user1", "host1", "label1", "")
	if err != nil {
		t.Fatalf("AddAccount error: %v", err)
	}

	// Use the bunKeyManager adapter directly and add a public key via KeyManager
	km := &bunKeyManager{bStore: s}
	pk, err := km.AddPublicKeyAndGetModel("ssh-ed25519", "ssh-ed25519 AAAAB3Nza... test-key", "k-one", false, time.Time{})
	if err != nil {
		t.Fatalf("AddPublicKeyAndGetModel error: %v", err)
	}

	// Assign key to account
	if err := km.AssignKeyToAccount(pk.ID, acctID); err != nil {
		t.Fatalf("AssignKeyToAccount failed: %v", err)
	}

	// Verify the key is visible for the account
	keys, err := km.GetKeysForAccount(acctID)
	if err != nil {
		t.Fatalf("GetKeysForAccount error: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected 1 key assigned; got %d", len(keys))
	}
	if keys[0].Comment != pk.Comment {
		t.Fatalf("expected key comment %q; got %q", pk.Comment, keys[0].Comment)
	}

	// Verify an audit entry was recorded for the assignment
	logs, err := s.GetAllAuditLogEntries()
	if err != nil {
		t.Fatalf("GetAllAuditLogEntries error: %v", err)
	}
	foundAssign := false
	for _, l := range logs {
		if l.Action == "ASSIGN_KEY" {
			foundAssign = true
			break
		}
	}
	if !foundAssign {
		t.Fatalf("expected at least one ASSIGN_KEY audit entry; got %v", logs)
	}

	// Unassign the key and verify removal and audit
	if err := km.UnassignKeyFromAccount(pk.ID, acctID); err != nil {
		t.Fatalf("UnassignKeyFromAccount failed: %v", err)
	}
	keysAfter, err := km.GetKeysForAccount(acctID)
	if err != nil {
		t.Fatalf("GetKeysForAccount after unassign error: %v", err)
	}
	if len(keysAfter) != 0 {
		t.Fatalf("expected 0 keys after unassign; got %d", len(keysAfter))
	}
	logs2, err := s.GetAllAuditLogEntries()
	if err != nil {
		t.Fatalf("GetAllAuditLogEntries after unassign error: %v", err)
	}
	foundUnassign := false
	for _, l := range logs2 {
		if l.Action == "UNASSIGN_KEY" {
			foundUnassign = true
			break
		}
	}
	if !foundUnassign {
		t.Fatalf("expected at least one UNASSIGN_KEY audit entry; got %v", logs2)
	}

	// Assigning the same key twice should not create duplicates. Call again
	// and verify the assigned set remains size 1.
	_ = km.AssignKeyToAccount(pk.ID, acctID)
	keysDup, err := km.GetKeysForAccount(acctID)
	if err != nil {
		t.Fatalf("GetKeysForAccount after duplicate assign error: %v", err)
	}
	if len(keysDup) != 1 {
		t.Fatalf("expected 1 key after duplicate assign; got %d", len(keysDup))
	}
}
