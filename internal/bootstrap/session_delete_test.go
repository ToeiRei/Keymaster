// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package bootstrap

import (
	"testing"

	"github.com/toeirei/keymaster/internal/db"
)

func TestBootstrapSession_SaveAndDelete(t *testing.T) {
	dsn := "file:test_" + t.Name() + "?mode=memory&cache=shared"
	if _, err := db.New("sqlite", dsn); err != nil {
		t.Fatalf("db.New failed: %v", err)
	}

	s, err := NewBootstrapSession("alice", "example.com", "lbl", "tags")
	if err != nil {
		t.Fatalf("NewBootstrapSession returned error: %v", err)
	}

	if err := s.Save(); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	// Ensure the session exists in DB
	bs, err := db.GetBootstrapSession(s.ID)
	if err != nil {
		t.Fatalf("GetBootstrapSession returned error: %v", err)
	}
	if bs == nil {
		t.Fatalf("expected saved session to exist, got nil")
	}

	// Delete via the method under test
	if err := s.Delete(); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}

	// Ensure it's removed
	bs, err = db.GetBootstrapSession(s.ID)
	if err != nil {
		t.Fatalf("GetBootstrapSession after delete returned error: %v", err)
	}
	if bs != nil {
		t.Fatalf("expected deleted session to be nil, got: %+v", bs)
	}
}
