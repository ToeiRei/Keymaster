package db

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestRunMigrationsSqlite(t *testing.T) {
	dsn := "file:test_migrations?mode=memory&cache=shared"
	dbConn, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer func() { _ = dbConn.Close() }()

	if err := RunMigrations(dbConn, "sqlite"); err != nil {
		t.Fatalf("RunMigrations failed: %v", err)
	}

	rows, err := dbConn.Query("SELECT version FROM schema_migrations")
	if err != nil {
		t.Fatalf("query schema_migrations failed: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var versions []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			t.Fatalf("scan version failed: %v", err)
		}
		versions = append(versions, v)
	}

	if len(versions) < 2 {
		t.Fatalf("expected at least 2 migrations applied, got %d", len(versions))
	}

	want := map[string]bool{
		"000001_create_initial_tables":     true,
		"000002_create_bootstrap_sessions": true,
	}
	for _, v := range versions {
		if _, ok := want[v]; ok {
			delete(want, v)
		}
	}
	if len(want) != 0 {
		t.Fatalf("missing expected migrations: %v", want)
	}
}

func TestRunDBMaintenanceSqlite_Smoke(t *testing.T) {
	dsn := "file:test_maint?mode=memory&cache=shared"
	if err := RunDBMaintenance("sqlite", dsn); err != nil {
		t.Fatalf("RunDBMaintenance failed: %v", err)
	}
}
