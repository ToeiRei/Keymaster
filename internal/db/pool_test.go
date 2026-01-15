// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package db

import (
	"testing"
)

// TestDBPoolDefaultsSQLite verifies that NewStoreFromDSN sets a sensible
// default for MaxOpenConns for SQLite. We assert the default value is applied
// and that the returned Store is the SQLite concrete type.
func TestDBPoolDefaultsSQLite(t *testing.T) {
	// Ensure CI env overrides do not change the expectation for this unit test.
	t.Setenv("KEYMASTER_DB_MAX_OPEN_CONNS", "")
	t.Setenv("KEYMASTER_DB_MAX_IDLE_CONNS", "")

	dsn := "file::memory:?cache=shared"
	s, err := NewStoreFromDSN("sqlite", dsn)
	if err != nil {
		t.Fatalf("NewStoreFromDSN returned error: %v", err)
	}
	ss, ok := s.(*SqliteStore)
	if !ok {
		t.Fatalf("expected *SqliteStore, got %T", s)
	}
	// The default in NewStoreFromDSN is 25. Check that the sql.DB Stats reflects that.
	stats := ss.BunDB().DB.Stats()
	want := 25
	if stats.MaxOpenConnections != want {
		t.Fatalf("MaxOpenConnections = %d; want %d", stats.MaxOpenConnections, want)
	}
	_ = ss.BunDB().DB.Close()
}

