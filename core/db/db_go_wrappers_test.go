// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package db

import (
	"database/sql"
	"errors"
	"strings"
	"testing"
)

// Test BunDB() returns nil when package-level store is nil and non-nil when set.
func TestBunDB_NilAndNonNil(t *testing.T) {
	// Save/restore globals
	prev := store
	defer func() { store = prev }()

	store = nil
	if BunDB() != nil {
		t.Fatalf("expected BunDB() to be nil when store is nil")
	}

	WithTestStore(t, func(s *BunStore) {
		if BunDB() == nil {
			t.Fatalf("expected BunDB() non-nil when store is initialized")
		}
	})
}

// Test NewStoreFromDSN returns error when sqlOpenFunc fails.
func TestNewStoreFromDSN_ErrorPath(t *testing.T) {
	// Save/restore sqlOpenFunc
	prev := sqlOpenFunc
	defer func() { sqlOpenFunc = prev }()

	sqlOpenFunc = func(driverName, dsn string) (*sql.DB, error) {
		return nil, errors.New("injected open failure")
	}

	s, err := NewStoreFromDSN("sqlite", ":memory:")
	if err == nil {
		// If it didn't error, ensure we clean up the store
		if s != nil {
			_ = s.BunDB()
		}
		t.Fatalf("expected NewStoreFromDSN to return error when sqlOpenFunc fails")
	}
	if !strings.Contains(err.Error(), "injected open failure") {
		t.Fatalf("unexpected error: %v", err)
	}
}
