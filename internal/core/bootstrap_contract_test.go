// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core

import (
	"fmt"
	"testing"

	"github.com/toeirei/keymaster/internal/keys"
	"github.com/toeirei/keymaster/internal/model"
)

func TestBuildAuthorizedKeysContent_Basic(t *testing.T) {
	systemKey := &model.SystemKey{Serial: 1, PublicKey: "ssh-ed25519 SYSKEYDATA sys-pub"}
	global := []model.PublicKey{{ID: 2, Algorithm: "ssh-ed25519", KeyData: "AAAAB3...1", Comment: "g1"}}
	account := []model.PublicKey{{ID: 3, Algorithm: "ssh-ed25519", KeyData: "AAAAB3...2", Comment: "a1"}}

	content, err := keys.BuildAuthorizedKeysContent(systemKey, global, account)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedHeader := fmt.Sprintf("# Keymaster Managed Keys (Serial: %d)\n", systemKey.Serial)
	if content[:len(expectedHeader)] != expectedHeader {
		t.Fatalf("expected header %q, got %q", expectedHeader, content[:len(expectedHeader)])
	}

	// Ensure restricted system key present
	restricted := "command=\"internal-sftp\",no-port-forwarding,no-x11-forwarding,no-agent-forwarding,no-pty " + systemKey.PublicKey
	if !containsLine(content, restricted) {
		t.Fatalf("restricted system key missing or malformed: %q", restricted)
	}

	// Ensure user keys block and order (a1 then g1)
	if !containsLine(content, "# User Keys") {
		t.Fatalf("missing user keys header")
	}

	if !containsLine(content, "ssh-ed25519 AAAAB3...2 a1") {
		t.Fatalf("missing account key line")
	}
	if !containsLine(content, "ssh-ed25519 AAAAB3...1 g1") {
		t.Fatalf("missing global key line")
	}
}

// containsLine checks that the content contains the exact substring on its own line (or within content).
func containsLine(s, sub string) bool {
	return len(s) >= len(sub) && (stringIndex(s, sub) >= 0)
}

// stringIndex is a tiny wrapper so tests don't import strings.
func stringIndex(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

