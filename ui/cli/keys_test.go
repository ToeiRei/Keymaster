// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package cli

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/toeirei/keymaster/core/db"
)

// TestKeyCommands_BasicFlow tests the key management workflow: add → list → show → set-expiry → enable-global → delete.
func TestKeyCommands_BasicFlow(t *testing.T) {
	setupTestDB(t)

	// Add a new key
	out := executeCommand(t, nil, "key", "add",
		"--algorithm", "ssh-ed25519",
		"--key-data", "AAAAC3NzaC1lZDI1NTE5AAAAIFakeKeyDataForTestingPurposes",
		"--comment", "test-key@example.com")
	if !strings.Contains(out, "Key added successfully") {
		t.Fatalf("add command failed, output: %s", out)
	}

	// List keys
	out = executeCommand(t, nil, "key", "list")
	if !strings.Contains(out, "ssh-ed25519") || !strings.Contains(out, "test-key@example.com") {
		t.Fatalf("expected key in list output, got: %s", out)
	}

	// Get the key ID from the list output
	km := db.DefaultKeyManager()
	keys, err := km.GetAllPublicKeys()
	if err != nil || len(keys) == 0 {
		t.Fatalf("failed to retrieve keys: %v", err)
	}
	keyID := fmt.Sprintf("%d", keys[0].ID)

	// Show key details
	out = executeCommand(t, nil, "key", "show", keyID)
	if !strings.Contains(out, "test-key@example.com") {
		t.Fatalf("expected key comment in show output, got: %s", out)
	}

	// Set expiry
	futureDate := time.Now().Add(30 * 24 * time.Hour).Format("2006-01-02")
	out = executeCommand(t, nil, "key", "set-expiry", keyID, futureDate)
	if !strings.Contains(out, "expiration set to") {
		t.Fatalf("expected expiry success message, got: %s", out)
	}

	// Enable global deployment
	out = executeCommand(t, nil, "key", "enable-global", keyID)
	if !strings.Contains(out, "enabled for global deployment") {
		t.Fatalf("expected global enable message, got: %s", out)
	}

	// Verify global status in list
	out = executeCommand(t, nil, "key", "list")
	if !strings.Contains(out, "yes") {
		t.Fatalf("expected 'yes' for global status in list, got: %s", out)
	}

	// Delete key with --force
	out = executeCommand(t, nil, "key", "delete", keyID, "--force")
	if !strings.Contains(out, "Key deleted") {
		t.Fatalf("expected deletion success message, got: %s", out)
	}

	// Verify key is gone
	keys, _ = km.GetAllPublicKeys()
	if len(keys) != 0 {
		t.Errorf("expected 0 keys after deletion, got %d", len(keys))
	}
}

// TestKeyEnableGlobalCmd_Idempotent verifies that enabling global twice doesn't error.
func TestKeyEnableGlobalCmd_Idempotent(t *testing.T) {
	setupTestDB(t)

	// Add a key
	km := db.DefaultKeyManager()
	addedKey, err := km.AddPublicKeyAndGetModel(
		"ssh-ed25519",
		"AAAAC3NzaC1lZDI1NTE5AAAAIFakeKeyDataForTesting",
		"idempotent-test@example.com",
		false,
		time.Time{},
	)
	if err != nil {
		t.Fatalf("failed to add key: %v", err)
	}

	keyID := fmt.Sprintf("%d", addedKey.ID)

	// Enable global first time
	out := executeCommand(t, nil, "key", "enable-global", keyID)
	if !strings.Contains(out, "enabled for global deployment") {
		t.Fatalf("expected enable success message, got: %s", out)
	}

	// Enable global second time (should be idempotent)
	out = executeCommand(t, nil, "key", "enable-global", keyID)
	if !strings.Contains(out, "already global") {
		t.Fatalf("expected 'already global' message, got: %s", out)
	}
}

// TestKeyDisableGlobalCmd_Idempotent verifies that disabling global twice doesn't error.
func TestKeyDisableGlobalCmd_Idempotent(t *testing.T) {
	setupTestDB(t)

	// Add a global key
	km := db.DefaultKeyManager()
	addedKey, err := km.AddPublicKeyAndGetModel(
		"ssh-ed25519",
		"AAAAC3NzaC1lZDI1NTE5AAAAIFakeKeyDataForTesting",
		"idempotent-disable@example.com",
		true, // Start as global
		time.Time{},
	)
	if err != nil {
		t.Fatalf("failed to add key: %v", err)
	}

	keyID := fmt.Sprintf("%d", addedKey.ID)

	// Disable global first time
	out := executeCommand(t, nil, "key", "disable-global", keyID)
	if !strings.Contains(out, "disabled from global deployment") {
		t.Fatalf("expected disable success message, got: %s", out)
	}

	// Disable global second time (should be idempotent)
	out = executeCommand(t, nil, "key", "disable-global", keyID)
	if !strings.Contains(out, "already non-global") {
		t.Fatalf("expected 'already non-global' message, got: %s", out)
	}
}

// TestKeyListCmd_Filtering tests filtering keys by global status and search terms.
func TestKeyListCmd_Filtering(t *testing.T) {
	setupTestDB(t)

	// Add keys using CLI commands instead of direct DB calls
	out1 := executeCommand(t, nil, "key", "add",
		"--algorithm", "ssh-ed25519",
		"--key-data", "AAAAC3NzaC1lZDI1NTE5AAAAIGlobalKey",
		"--comment", "global-key@example.com",
		"--global")
	if !strings.Contains(out1, "Key added successfully") {
		t.Fatalf("failed to add global key: %s", out1)
	}

	out2 := executeCommand(t, nil, "key", "add",
		"--algorithm", "ssh-rsa",
		"--key-data", "AAAAB3NzaC1yc2EAAAADAQABNonGlobalKey",
		"--comment", "local-key@example.com",
		"--global=false")
	if !strings.Contains(out2, "Key added successfully") {
		t.Fatalf("failed to add local key: %s", out2)
	}

	// List all keys
	out := executeCommand(t, nil, "key", "list")
	if !strings.Contains(out, "global-key@example.com") && !strings.Contains(out, "local-key@example.com") {
		// Debug: print what we got
		t.Logf("List output: %s", out)
		// Try to see if keys exist in DB directly
		km := db.DefaultKeyManager()
		keys, err := km.GetAllPublicKeys()
		t.Logf("Direct DB query: %d keys, err=%v", len(keys), err)
		t.Fatalf("expected both keys in list")
	}

	// Filter by global=yes
	out = executeCommand(t, nil, "key", "list", "--global", "yes")
	if !strings.Contains(out, "global-key@example.com") {
		t.Fatalf("expected global key in filtered list, got: %s", out)
	}

	// Filter by global=no
	out = executeCommand(t, nil, "key", "list", "--global", "no")
	if !strings.Contains(out, "local-key@example.com") {
		t.Fatalf("expected non-global key in filtered list, got: %s", out)
	}
}

// TestKeySetExpiryCmd_ClearExpiry tests setting and clearing expiration dates.
func TestKeySetExpiryCmd_ClearExpiry(t *testing.T) {
	setupTestDB(t)

	// Add a key using CLI
	executeCommand(t, nil, "key", "add",
		"--algorithm", "ssh-ed25519",
		"--key-data", "AAAAC3NzaC1lZDI1NTE5AAAAIExpiryTestKey",
		"--comment", "expiry-test@example.com")

	// Get the key ID (should be 1 in fresh DB)
	keyID := "1"

	// Set expiry date
	futureDate := time.Now().Add(60 * 24 * time.Hour).Format("2006-01-02")
	out := executeCommand(t, nil, "key", "set-expiry", keyID, futureDate)
	if !strings.Contains(out, "expiration set to") {
		t.Fatalf("expected expiry set message, got: %s", out)
	}

	// Clear expiry with 'never'
	out = executeCommand(t, nil, "key", "set-expiry", keyID, "never")
	if !strings.Contains(out, "expiration cleared") {
		t.Fatalf("expected expiry cleared message, got: %s", out)
	}

	// Verify key shows 'never' in list
	out = executeCommand(t, nil, "key", "list")
	if !strings.Contains(out, "never") {
		t.Fatalf("expected 'never' in expiry column, got: %s", out)
	}
}
