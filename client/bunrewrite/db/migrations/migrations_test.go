// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package migrations

import (
	"context"
	"database/sql"
	"testing"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/migrate"

	_ "modernc.org/sqlite"
)

func openMemBunDB(t *testing.T) *bun.DB {
	t.Helper()
	conn, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	conn.SetMaxOpenConns(1)
	conn.SetMaxIdleConns(1)
	t.Cleanup(func() { _ = conn.Close() })
	return bun.NewDB(conn, sqlitedialect.New())
}

func tableExists(t *testing.T, db *bun.DB, table string) bool {
	t.Helper()
	var exists int
	err := db.QueryRow("SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = ?", table).Scan(&exists)
	if err == sql.ErrNoRows {
		return false
	}
	if err != nil {
		t.Fatalf("checking table %s: %v", table, err)
	}
	return true
}

// TestBaselineName_MatchesFirstRegisteredMigration guards bridge.go's by-name
// lookup: it always targets whichever migration is registered first, so a
// future reorder or rename can't silently make it target the wrong one.
func TestBaselineName_MatchesFirstRegisteredMigration(t *testing.T) {
	sorted := Migrations.Sorted()
	if len(sorted) == 0 {
		t.Fatal("no migrations registered")
	}
	if sorted[0].Name != BaselineName {
		t.Fatalf("first registered migration is %q, want BaselineName %q", sorted[0].Name, BaselineName)
	}
}

func TestBaseline_CreatesFullSchema(t *testing.T) {
	ctx := context.Background()
	db := openMemBunDB(t)

	migrator := migrate.NewMigrator(db, Migrations)
	if err := migrator.Init(ctx); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if _, err := migrator.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	for _, table := range []string{
		"accounts", "public_keys", "links", "audit_log",
		"system_keys", "known_hosts", "bootstrap_sessions",
	} {
		if !tableExists(t, db, table) {
			t.Errorf("expected table %q to exist", table)
		}
	}
	if tableExists(t, db, "account_keys") {
		t.Error("account_keys should not exist on a fresh install")
	}
}

func TestMigrate_Idempotent(t *testing.T) {
	ctx := context.Background()
	db := openMemBunDB(t)

	migrator := migrate.NewMigrator(db, Migrations)
	if err := migrator.Init(ctx); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if _, err := migrator.Migrate(ctx); err != nil {
		t.Fatalf("first Migrate: %v", err)
	}
	group, err := migrator.Migrate(ctx)
	if err != nil {
		t.Fatalf("second Migrate: %v", err)
	}
	if !group.IsZero() {
		t.Fatalf("expected no-op migration group on second run, got %s", group)
	}
}
