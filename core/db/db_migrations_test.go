// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package db

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

// ensureSchemaMigrationsTable adds the applied_at column when missing.
func TestEnsureSchemaMigrationsTable_AddsColumn(t *testing.T) {
	db, err := sql.Open("sqlite", "file:test_ensure?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer db.Close()

	// Create schema_migrations without applied_at
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY)`); err != nil {
		t.Fatalf("create table: %v", err)
	}

	if err := ensureSchemaMigrationsTable(db, "sqlite"); err != nil {
		t.Fatalf("ensureSchemaMigrationsTable failed: %v", err)
	}

	// Check PRAGMA table_info to ensure applied_at exists
	rows, err := db.Query("PRAGMA table_info(schema_migrations)")
	if err != nil {
		t.Fatalf("pragma query: %v", err)
	}
	defer rows.Close()
	found := false
	for rows.Next() {
		var cid int
		var name string
		var typ string
		var notnull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if name == "applied_at" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("applied_at column not found")
	}
}

// RunMigrations should return nil when no embedded migrations exist for the type.
func TestRunMigrations_NoMigrations(t *testing.T) {
	db, err := sql.Open("sqlite", "file:test_nomig?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer db.Close()
	// Use a dbType that has no embedded migrations to exercise the ErrNotExist path.
	if err := RunMigrations(db, "no-such-type"); err != nil {
		t.Fatalf("RunMigrations (no migrations) returned error: %v", err)
	}
}

// RunMigrations should apply sqlite embedded migrations without error.
func TestRunMigrations_SQLite(t *testing.T) {
	db, err := sql.Open("sqlite", "file:test_mig?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer db.Close()
	if err := RunMigrations(db, "sqlite"); err != nil {
		t.Fatalf("RunMigrations sqlite failed: %v", err)
	}
}
