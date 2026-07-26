// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package migrations

import (
	"context"
	"database/sql"
	"encoding/json"
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

// migrateAll runs the whole registry against db.
func migrateAll(t *testing.T, db *bun.DB) {
	t.Helper()
	ctx := context.Background()
	migrator := migrate.NewMigrator(db, Migrations)
	if err := migrator.Init(ctx); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if _, err := migrator.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
}

func TestBaseline_CreatesFullSchema(t *testing.T) {
	db := openMemBunDB(t)
	migrateAll(t, db)

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

// seedCutoverSchema creates the baseline schema without recording it, so a
// test can plant legacy rows before the later migrations see them. baselineUp
// is idempotent (every statement is IF NOT EXISTS), so migrateAll can still
// run it for real afterwards.
func seedCutoverSchema(t *testing.T, db *bun.DB) {
	t.Helper()
	if err := baselineUp(context.Background(), db); err != nil {
		t.Fatalf("baselineUp: %v", err)
	}
}

func exec(t *testing.T, db *bun.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
}

func TestBackfill_CopiesActiveSystemKeyIntoDeploySecret(t *testing.T) {
	db := openMemBunDB(t)
	seedCutoverSchema(t, db)

	// higher serial but inactive: the active key must still win
	exec(t, db, "INSERT INTO system_keys (serial, public_key, private_key, is_active) VALUES (2, 'pub2', 'PRIV-STALE', false)")
	exec(t, db, "INSERT INTO system_keys (serial, public_key, private_key, is_active) VALUES (1, 'pub1', 'PRIV-ACTIVE', true)")
	exec(t, db, "INSERT INTO accounts (username, host) VALUES ('root', 'example.com')")
	// an account already configured by the new client must not be clobbered
	exec(t, db, "INSERT INTO accounts (username, host, port, deploy_method, deploy_secret) VALUES ('deploy', 'other.com', '2222', 'mock', 'true')")

	migrateAll(t, db)

	var secret, method, port string
	if err := db.QueryRow("SELECT deploy_secret, deploy_method, port FROM accounts WHERE username = 'root'").
		Scan(&secret, &method, &port); err != nil {
		t.Fatalf("query backfilled account: %v", err)
	}
	if secret != "PRIV-ACTIVE" || method != "ssh" || port != "22" {
		t.Fatalf("account not backfilled: secret=%q method=%q port=%q", secret, method, port)
	}

	if err := db.QueryRow("SELECT deploy_secret, deploy_method, port FROM accounts WHERE username = 'deploy'").
		Scan(&secret, &method, &port); err != nil {
		t.Fatalf("query configured account: %v", err)
	}
	if secret != "true" || method != "mock" || port != "2222" {
		t.Fatalf("configured account was clobbered: secret=%q method=%q port=%q", secret, method, port)
	}
}

func TestBackfill_FallsBackToHighestSerialWhenNoneActive(t *testing.T) {
	db := openMemBunDB(t)
	seedCutoverSchema(t, db)

	exec(t, db, "INSERT INTO system_keys (serial, public_key, private_key, is_active) VALUES (1, 'pub1', 'PRIV-OLD', false)")
	exec(t, db, "INSERT INTO system_keys (serial, public_key, private_key, is_active) VALUES (3, 'pub3', 'PRIV-NEWEST', false)")
	exec(t, db, "INSERT INTO accounts (username, host) VALUES ('root', 'example.com')")

	migrateAll(t, db)

	var secret string
	if err := db.QueryRow("SELECT deploy_secret FROM accounts WHERE username = 'root'").Scan(&secret); err != nil {
		t.Fatalf("query account: %v", err)
	}
	if secret != "PRIV-NEWEST" {
		t.Fatalf("deploy_secret = %q, want PRIV-NEWEST", secret)
	}
}

func TestBackfill_NoSystemKeysLeavesDeploySecretEmpty(t *testing.T) {
	db := openMemBunDB(t)
	seedCutoverSchema(t, db)

	exec(t, db, "INSERT INTO accounts (username, host) VALUES ('root', 'example.com')")

	migrateAll(t, db)

	var secret, method string
	if err := db.QueryRow("SELECT deploy_secret, deploy_method FROM accounts WHERE username = 'root'").
		Scan(&secret, &method); err != nil {
		t.Fatalf("query account: %v", err)
	}
	if secret != "" {
		t.Fatalf("deploy_secret = %q, want empty", secret)
	}
	// the method backfill runs regardless of whether a key was found
	if method != "ssh" {
		t.Fatalf("deploy_method = %q, want ssh", method)
	}
}

func TestBackfill_ConvertsLegacyAuditDetailsToJSON(t *testing.T) {
	db := openMemBunDB(t)
	seedCutoverSchema(t, db)

	// quotes and a backslash: the conversion must escape them, not corrupt them
	exec(t, db, `INSERT INTO audit_log (id, username, action, details) VALUES (1, 'root', 'ADD_ACCOUNT', 'account: "root"\srv')`)
	exec(t, db, `INSERT INTO audit_log (id, username, action, details) VALUES (2, 'root', 'account.create', '[{"key":"id","value":"7"}]')`)
	exec(t, db, `INSERT INTO audit_log (id, username, action, details) VALUES (3, 'root', 'account.delete', 'null')`)
	exec(t, db, `INSERT INTO audit_log (id, username, action, details) VALUES (4, 'root', 'account.verify', NULL)`)

	migrateAll(t, db)

	var converted string
	if err := db.QueryRow("SELECT details FROM audit_log WHERE id = 1").Scan(&converted); err != nil {
		t.Fatalf("query converted row: %v", err)
	}
	var details []auditDetail
	if err := json.Unmarshal([]byte(converted), &details); err != nil {
		t.Fatalf("converted details are not valid JSON (%q): %v", converted, err)
	}
	if len(details) != 1 || details[0].Key != "legacy" || details[0].Value != `account: "root"\srv` {
		t.Fatalf("unexpected converted details: %+v", details)
	}

	var untouched string
	if err := db.QueryRow("SELECT details FROM audit_log WHERE id = 2").Scan(&untouched); err != nil {
		t.Fatalf("query new-format row: %v", err)
	}
	if untouched != `[{"key":"id","value":"7"}]` {
		t.Fatalf("new-format row was rewritten: %q", untouched)
	}

	if err := db.QueryRow("SELECT details FROM audit_log WHERE id = 3").Scan(&untouched); err != nil {
		t.Fatalf("query null row: %v", err)
	}
	if untouched != "null" {
		t.Fatalf("bun-written null details was rewritten: %q", untouched)
	}

	var isNull bool
	if err := db.QueryRow("SELECT details IS NULL FROM audit_log WHERE id = 4").Scan(&isNull); err != nil {
		t.Fatalf("query NULL row: %v", err)
	}
	if !isNull {
		t.Error("NULL details should have been left alone")
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
