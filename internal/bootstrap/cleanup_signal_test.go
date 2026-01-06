// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package bootstrap

import (
	"fmt"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"

	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/testutil"
)

// TestCleanupSession_LogsStructuredAudit ensures cleanupSession emits a
// structured BOOTSTRAP_FAILED audit entry (contains session=..., account=... and reason=interrupted_by_signal).
func TestCleanupSession_LogsStructuredAudit(t *testing.T) {
	if err := db.InitDB("sqlite", ":memory:"); err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}

	// Create an in-memory bootstrap session and persist it so UpdateStatus works.
	s, err := NewBootstrapSession("root", "192.168.10.15", "lbl", "")
	if err != nil {
		t.Fatalf("NewBootstrapSession failed: %v", err)
	}
	if err := s.Save(); err != nil {
		t.Fatalf("SaveBootstrapSession failed: %v", err)
	}

	fake := &testutil.FakeAuditWriter{}
	db.SetDefaultAuditWriter(fake)
	defer db.ClearDefaultAuditWriter()

	// Stub out network dial to avoid real SSH attempts and speed the test.
	prevDial := sshDialFunc
	sshDialFunc = func(network, addr string, config *ssh.ClientConfig) (*ssh.Client, error) {
		return nil, fmt.Errorf("dial disabled in test")
	}
	defer func() { sshDialFunc = prevDial }()

	if err := cleanupSession(s); err != nil {
		t.Fatalf("cleanupSession failed: %v", err)
	}

	if len(fake.Calls) == 0 {
		t.Fatalf("expected audit calls, got none")
	}
	if fake.Calls[0][0] != "BOOTSTRAP_FAILED" {
		t.Fatalf("unexpected audit action: %s", fake.Calls[0][0])
	}
	details := fake.Calls[0][1]
	if !strings.Contains(details, "session=") || !strings.Contains(details, "account=root@192.168.10.15") || !strings.Contains(details, "reason=interrupted_by_signal") {
		t.Fatalf("unexpected audit details: %s", details)
	}
}
