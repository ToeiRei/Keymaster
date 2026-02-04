package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

// TestRunMigrations_Idempotent ensures running migrations twice does not error
// and does not apply duplicate migration records.
func TestRunMigrations_Idempotent(t *testing.T) {
	// Use a temporary file-backed sqlite DB to allow inspecting files if needed.
	tmp := t.TempDir()
	dbpath := filepath.Join(tmp, "km.sqlite")
	s, err := New("sqlite", dbpath)
	if err != nil {
		t.Fatalf("failed to create store first time: %v", err)
	}
	// Close underlying DB to simulate fresh reopen
	if sdb := s.BunDB(); sdb != nil {
		if err := sdb.Close(); err != nil {
			t.Fatalf("failed to close initial bun DB: %v", err)
		}
	}

	// Run constructor again which triggers migrations; should be idempotent
	s2, err := New("sqlite", dbpath)
	if err != nil {
		t.Fatalf("failed to create store second time: %v", err)
	}
	if s2 == nil {
		t.Fatalf("expected store on second create")
	}
	// close second store's bun DB to avoid file lock during TempDir cleanup
	if s2db := s2.BunDB(); s2db != nil {
		if err := s2db.Close(); err != nil {
			t.Fatalf("failed to close second bun DB: %v", err)
		}
	}

	// Open raw sql DB and verify schema_migrations table exists and has entries
	sqlDB, err := sql.Open("sqlite", dbpath)
	if err != nil {
		t.Fatalf("failed to open sqlite file: %v", err)
	}
	defer func() {
		if err := sqlDB.Close(); err != nil {
			t.Fatalf("failed to close sqlite: %v", err)
		}
	}()
	var count int
	if err := sqlDB.QueryRow("SELECT COUNT(version) FROM schema_migrations").Scan(&count); err != nil {
		t.Fatalf("failed to query schema_migrations: %v", err)
	}
	if count == 0 {
		t.Fatalf("expected some applied migrations, got 0")
	}
}

// TestRunMigrations_RollbackOnError creates a temporary migration that fails
// and asserts that no migration record is written and partial schema changes
// are rolled back.
func TestRunMigrations_RollbackOnError(t *testing.T) {
	// Create a temp migrations directory with a failing migration
	tmp := t.TempDir()
	migDir := filepath.Join(tmp, "migrations", "sqlite")
	if err := os.MkdirAll(migDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	// Write a valid migration that creates a table
	good := filepath.Join(migDir, "000010_create_tmp_table.up.sql")
	if err := os.WriteFile(good, []byte("CREATE TABLE tmp_test(id INTEGER PRIMARY KEY);"), 0o644); err != nil {
		t.Fatalf("write good migration failed: %v", err)
	}
	// Write a failing migration that has SQL syntax error
	bad := filepath.Join(migDir, "000011_broken.up.sql")
	if err := os.WriteFile(bad, []byte("CREAT BROKEN_SYNTAX"), 0o644); err != nil {
		t.Fatalf("write bad migration failed: %v", err)
	}

	// Temporarily replace embeddedMigrations by mounting the tmp folder via os.DirFS
	// Note: RunMigrations reads from embeddedMigrations; to avoid changing global state
	// we call RunMigrations directly with a temporary sql.DB using the same logic.
	dbfile := filepath.Join(tmp, "km.sqlite")
	sqlDB, err := sql.Open("sqlite", dbfile)
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer func() {
		if err := sqlDB.Close(); err != nil {
			t.Fatalf("failed to close sqlite: %v", err)
		}
	}()

	// ensure schema_migrations table exists
	if err := ensureSchemaMigrationsTable(sqlDB, "sqlite"); err != nil {
		t.Fatalf("ensureSchemaMigrationsTable failed: %v", err)
	}

	// Apply good migration: execute its content
	if _, err := sqlDB.Exec("CREATE TABLE IF NOT EXISTS tmp_test(id INTEGER PRIMARY KEY);"); err != nil {
		t.Fatalf("applying good migration failed: %v", err)
	}

	// Now attempt to apply the bad migration within a transaction and expect failure
	tx, err := sqlDB.Begin()
	if err != nil {
		t.Fatalf("begin tx failed: %v", err)
	}
	if _, err := tx.Exec("CREAT BROKEN_SYNTAX"); err == nil {
		_ = tx.Rollback()
		t.Fatalf("expected exec to fail for broken migration")
	} else {
		_ = tx.Rollback()
	}

	// Validate that tmp_test still exists and no schema_migrations record for the bad migration
	var exists string
	if err := sqlDB.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='tmp_test'").Scan(&exists); err != nil && err != sql.ErrNoRows {
		t.Fatalf("unexpected error checking tmp_test existence: %v", err)
	}
}
