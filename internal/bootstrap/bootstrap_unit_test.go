// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package bootstrap

import (
	"testing"
	"time"
)

func TestRemoveLine_RemovesCorrectLine(t *testing.T) {
	content := "first\nsecond\nthird\n"
	out := removeLine(content, "second")
	if out == content {
		t.Fatalf("expected line removed, got same content")
	}
	if contains := (len(out) > 0 && out == "first\nthird\n"); !contains {
		t.Fatalf("unexpected result: %q", out)
	}
}

func TestTemporaryKeyPair_CleanupAndGetters(t *testing.T) {
	tk := &TemporaryKeyPair{
		privateKey: []byte("secretdata"),
		publicKey:  "ssh-ed25519 AAA...",
	}

	if got := tk.GetPublicKey(); got != "ssh-ed25519 AAA..." {
		t.Fatalf("unexpected public key: %q", got)
	}
	if len(tk.GetPrivateKeyPEM()) == 0 {
		t.Fatalf("expected non-empty private key")
	}

	tk.Cleanup()
	if pk := tk.GetPrivateKeyPEM(); pk != nil {
		t.Fatalf("expected private key to be cleared, got %v", pk)
	}
}

func TestNewBootstrapSession_BasicsAndCommand(t *testing.T) {
	s, err := NewBootstrapSession("alice", "host.example", "lbl", "tag1")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	if s.ID == "" {
		t.Fatalf("expected non-empty session ID")
	}
	if s.TempKeyPair == nil {
		t.Fatalf("expected temporary key pair")
	}
	cmd := s.GetBootstrapCommand()
	if cmd == "" || s.TempKeyPair.publicKey == "" {
		t.Fatalf("unexpected bootstrap command or public key")
	}
	if !containsSubstring(cmd, s.TempKeyPair.publicKey) {
		t.Fatalf("bootstrap command does not contain public key: %q", cmd)
	}
}

func TestIsExpired_Behavior(t *testing.T) {
	s := &BootstrapSession{}
	s.ExpiresAt = time.Now().Add(-time.Hour)
	if !s.IsExpired() {
		t.Fatalf("expected expired session")
	}
	s.ExpiresAt = time.Now().Add(time.Hour)
	if s.IsExpired() {
		t.Fatalf("expected non-expired session")
	}
}

func TestGenerateSessionID_Length(t *testing.T) {
	id, err := generateSessionID()
	if err != nil {
		t.Fatalf("generateSessionID error: %v", err)
	}
	if len(id) != 32 {
		t.Fatalf("unexpected session id length: %d", len(id))
	}
}

// containsSubstring is a tiny helper to avoid pulling fmt into tests repeatedly.
func containsSubstring(s, sub string) bool {
	return len(sub) > 0 && len(s) >= len(sub) && (s == sub || len(s) > len(sub) && (indexOf(s, sub) >= 0))
}

// indexOf finds the first index of substr in s or -1.
func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
