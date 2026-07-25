// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package db

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/toeirei/keymaster/client/bunrewrite/db/legacymigrate"
	"github.com/toeirei/keymaster/client/bunrewrite/db/migrations"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/migrate"

	_ "modernc.org/sqlite"
)

// openMemSQLite returns an in-memory SQLite DB pinned to a single connection
// so the schema is shared across queries.
func openMemSQLite(t *testing.T) *sql.DB {
	t.Helper()
	conn, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	conn.SetMaxOpenConns(1)
	conn.SetMaxIdleConns(1)
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

// applyLegacyMigrations applies sqlite up-migration files directly (bypassing
// the legacymigrate runner) and records them in schema_migrations, simulating
// a database created by the legacy migration chain up to (but excluding) the
// given prefix. Pass "" to apply every migration, simulating an install
// already fully at the old final (000005) shape.
func applyLegacyMigrations(t *testing.T, conn *sql.DB, excludePrefix string) {
	t.Helper()
	if _, err := conn.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY, applied_at TIMESTAMP)`); err != nil {
		t.Fatalf("ensure schema_migrations: %v", err)
	}

	dir := filepath.Join("legacymigrate", "migrations", "sqlite")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read migrations dir: %v", err)
	}
	var ups []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".up.sql") && (excludePrefix == "" || !strings.HasPrefix(e.Name(), excludePrefix)) {
			ups = append(ups, e.Name())
		}
	}
	sort.Strings(ups)

	for _, name := range ups {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if _, err := conn.Exec(string(data)); err != nil {
			t.Fatalf("apply %s: %v", name, err)
		}
		version := strings.TrimSuffix(name, ".up.sql")
		if _, err := conn.Exec("INSERT INTO schema_migrations(version, applied_at) VALUES(?, ?)", version, time.Now()); err != nil {
			t.Fatalf("record %s: %v", name, err)
		}
	}
}

func newBunDB(conn *sql.DB) *bun.DB {
	bunDB := bun.NewDB(conn, sqlitedialect.New())
	bunDB.RegisterModel((*LinkModel)(nil))
	return bunDB
}

func appliedMigrationNames(t *testing.T, bunDB *bun.DB) []string {
	t.Helper()
	migrator := migrate.NewMigrator(bunDB, migrations.Migrations)
	applied, err := migrator.AppliedMigrations(context.Background())
	if err != nil {
		t.Fatalf("AppliedMigrations: %v", err)
	}
	var names []string
	for _, m := range applied {
		names = append(names, m.Name)
	}
	sort.Strings(names)
	return names
}

func tableExistsSQLite(t *testing.T, conn *sql.DB, table string) bool {
	t.Helper()
	var exists int
	err := conn.QueryRow("SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = ?", table).Scan(&exists)
	if err == sql.ErrNoRows {
		return false
	}
	if err != nil {
		t.Fatalf("checking table %s: %v", table, err)
	}
	return true
}

func TestOpen_FreshDatabase(t *testing.T) {
	conn := openMemSQLite(t)
	bunDB := newBunDB(conn)
	ctx := context.Background()

	if err := runMigrations(ctx, conn, bunDB, "sqlite"); err != nil {
		t.Fatalf("runMigrations: %v", err)
	}

	if tableExistsSQLite(t, conn, "schema_migrations") {
		t.Error("schema_migrations should never be created for a fresh install")
	}
	if tableExistsSQLite(t, conn, "account_keys") {
		t.Error("account_keys should not exist on a fresh install")
	}
	names := appliedMigrationNames(t, bunDB)
	if len(names) != 2 || names[0] != migrations.BaselineName {
		t.Fatalf("unexpected applied migrations: %v", names)
	}

	// smoke CRUD round-trip through the registered models
	if _, err := bunDB.NewInsert().Model(&AccountModel{Username: "root", Host: "example.com", IsActive: true}).Exec(ctx); err != nil {
		t.Fatalf("insert account: %v", err)
	}
	var count int
	if err := conn.QueryRow("SELECT COUNT(*) FROM accounts").Scan(&count); err != nil {
		t.Fatalf("count accounts: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 account, got %d", count)
	}
}

func TestOpen_LegacyMidChain(t *testing.T) {
	conn := openMemSQLite(t)
	applyLegacyMigrations(t, conn, "000005") // 000001-000004 only

	if _, err := conn.Exec("INSERT INTO accounts (id, username, hostname, is_active, is_dirty) VALUES (1, 'root', 'example.com', 1, 1)"); err != nil {
		t.Fatalf("seed account: %v", err)
	}
	if _, err := conn.Exec("INSERT INTO public_keys (id, algorithm, key_data, comment, is_global) VALUES (1, 'ssh-ed25519', 'AAAAdata', 'alice', 0)"); err != nil {
		t.Fatalf("seed public_key: %v", err)
	}
	if _, err := conn.Exec("INSERT INTO account_keys (account_id, key_id) VALUES (1, 1)"); err != nil {
		t.Fatalf("seed account_keys: %v", err)
	}

	bunDB := newBunDB(conn)
	if err := runMigrations(context.Background(), conn, bunDB, "sqlite"); err != nil {
		t.Fatalf("runMigrations: %v", err)
	}

	// legacy runner completed 000005
	var version string
	if err := conn.QueryRow("SELECT version FROM schema_migrations WHERE version = '000005_bunrewrite_schema'").Scan(&version); err != nil {
		t.Fatalf("expected 000005 recorded in schema_migrations: %v", err)
	}

	// baseline + drop-account_keys recorded on the new system, without
	// baseline's Up() ever running (it would fail: accounts already exists)
	names := appliedMigrationNames(t, bunDB)
	if len(names) != 2 || names[0] != migrations.BaselineName {
		t.Fatalf("unexpected applied migrations: %v", names)
	}

	if tableExistsSQLite(t, conn, "account_keys") {
		t.Error("account_keys should have been dropped")
	}

	var host, deployMethod string
	if err := conn.QueryRow("SELECT host, deploy_method FROM accounts WHERE id = 1").Scan(&host, &deployMethod); err != nil {
		t.Fatalf("query accounts: %v", err)
	}
	if host != "example.com" || deployMethod != "" {
		t.Fatalf("account not reshaped: host=%q deploy_method=%q", host, deployMethod)
	}

	var accountID, publicKeyID int
	if err := conn.QueryRow("SELECT account_id, public_key_id FROM links").Scan(&accountID, &publicKeyID); err != nil {
		t.Fatalf("query links: %v", err)
	}
	if accountID != 1 || publicKeyID != 1 {
		t.Fatalf("link not backfilled from account_keys: account_id=%d public_key_id=%d", accountID, publicKeyID)
	}
}

func TestOpen_LegacyAtOldFinalShape(t *testing.T) {
	conn := openMemSQLite(t)
	applyLegacyMigrations(t, conn, "000005") // 000001-000004
	// simulate an already-shipped, already-upgraded bunrewrite install: the
	// legacy runner has already applied 000005 in a prior run of the app,
	// before this change ever existed.
	if err := legacymigrate.RunMigrations(conn, "sqlite"); err != nil {
		t.Fatalf("pre-seed legacy RunMigrations: %v", err)
	}

	bunDB := newBunDB(conn)
	if err := runMigrations(context.Background(), conn, bunDB, "sqlite"); err != nil {
		t.Fatalf("runMigrations: %v", err)
	}

	names := appliedMigrationNames(t, bunDB)
	if len(names) != 2 || names[0] != migrations.BaselineName {
		t.Fatalf("unexpected applied migrations: %v", names)
	}
	if tableExistsSQLite(t, conn, "account_keys") {
		t.Error("account_keys should have been dropped")
	}
}

func TestOpen_AlreadyOnNewSystem(t *testing.T) {
	conn := openMemSQLite(t)
	bunDB := newBunDB(conn)
	ctx := context.Background()

	if err := runMigrations(ctx, conn, bunDB, "sqlite"); err != nil {
		t.Fatalf("first runMigrations: %v", err)
	}
	before := appliedMigrationNames(t, bunDB)

	if err := runMigrations(ctx, conn, bunDB, "sqlite"); err != nil {
		t.Fatalf("second runMigrations: %v", err)
	}
	after := appliedMigrationNames(t, bunDB)

	if len(before) != len(after) {
		t.Fatalf("expected no-op second run: before=%v after=%v", before, after)
	}
}
