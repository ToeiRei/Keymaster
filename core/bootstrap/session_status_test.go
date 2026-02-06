// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package bootstrap

import (
	"testing"

	"github.com/toeirei/keymaster/core/db"
)

func TestBootstrapSession_UpdateStatusPersists(t *testing.T) {
	dsn := "file:test_" + t.Name() + "?mode=memory&cache=shared"
	if _, err := db.New("sqlite", dsn); err != nil {
		t.Fatalf("db.New failed: %v", err)
	}

	s, err := NewBootstrapSession("bob", "host.local", "lbl", "t")
	if err != nil {
		t.Fatalf("NewBootstrapSession returned error: %v", err)
	}

	if err := s.Save(); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	if err := s.UpdateStatus(StatusCommitting); err != nil {
		t.Fatalf("UpdateStatus returned error: %v", err)
	}

	bs, err := db.GetBootstrapSession(s.ID)
	if err != nil {
		t.Fatalf("GetBootstrapSession returned error: %v", err)
	}
	if bs == nil {
		t.Fatalf("expected bootstrap session in DB, got nil")
	}
	if bs.Status != string(StatusCommitting) {
		t.Fatalf("unexpected status in DB: %q", bs.Status)
	}
}
