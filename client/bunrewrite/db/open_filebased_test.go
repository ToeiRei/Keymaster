// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/toeirei/keymaster/client/bunrewrite/db/migrations"
	"github.com/uptrace/bun"
)

// legacyStages describes real historical snapshots of a user's on-disk
// database: how far the legacy SQL-file migration chain had progressed the
// last time they upgraded, before ever seeing the new bun/migrate system.
var legacyStages = []struct {
	name string
	// throughVersion is the last legacy migration (filename minus .up.sql)
	// to apply; "" applies every legacy migration, i.e. the old bunrewrite
	// final (000005) shape.
	throughVersion string
	// oldShape is true when the schema at this stage predates the 000005
	// reshape (hostname/key_data/account_keys instead of host/data/links).
	oldShape bool
}{
	{"only 000001 applied", "000001_create_initial_tables", true},
	{"through the 000002 batch", "000002_create_bootstrap_sessions", true},
	{"through 000003", "000003_add_account_key_hash", true},
	{"through 000004 (full pre-bunrewrite legacy)", "000004_add_audit_log_context_columns", true},
	{"fully at 000005 (old bunrewrite final shape)", "", false},
}

// applyLegacyMigrationsThrough applies every sqlite up-migration file in
// order up to and including throughVersion (or all of them if throughVersion
// is ""), recording each in schema_migrations exactly like a real install
// that stopped upgrading at that point.
func applyLegacyMigrationsThrough(t *testing.T, conn *sql.DB, throughVersion string) {
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
		if strings.HasSuffix(e.Name(), ".up.sql") {
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
		if version == throughVersion {
			break
		}
	}
}

// seedLegacyShapeData seeds an account, public key, and their join row using
// only columns that exist from 000001 onward, so it's valid at every
// oldShape stage regardless of how far the legacy chain had progressed.
func seedLegacyShapeData(t *testing.T, conn *sql.DB) {
	t.Helper()
	if _, err := conn.Exec("INSERT INTO accounts (id, username, hostname) VALUES (1, 'root', 'example.com')"); err != nil {
		t.Fatalf("seed account: %v", err)
	}
	if _, err := conn.Exec("INSERT INTO public_keys (id, algorithm, key_data, comment) VALUES (1, 'ssh-ed25519', 'AAAAdata', 'alice')"); err != nil {
		t.Fatalf("seed public_key: %v", err)
	}
	if _, err := conn.Exec("INSERT INTO account_keys (account_id, key_id) VALUES (1, 1)"); err != nil {
		t.Fatalf("seed account_keys: %v", err)
	}
	seedSystemKey(t, conn)
}

// seedSystemKey plants the shared legacy SSH identity that the backfill fans
// out into accounts.deploy_secret. system_keys exists from 000001 onward, so
// every stage can seed it.
func seedSystemKey(t *testing.T, conn *sql.DB) {
	t.Helper()
	if _, err := conn.Exec("INSERT INTO system_keys (serial, public_key, private_key, is_active) VALUES (1, 'pub', 'PRIV-ACTIVE', 1)"); err != nil {
		t.Fatalf("seed system_key: %v", err)
	}
}

// seedNewShapeData seeds the same logical row directly in the post-000005
// shape, for the stage that's already fully upgraded before Open() ever
// runs.
func seedNewShapeData(t *testing.T, conn *sql.DB) {
	t.Helper()
	if _, err := conn.Exec("INSERT INTO accounts (id, username, host) VALUES (1, 'root', 'example.com')"); err != nil {
		t.Fatalf("seed account: %v", err)
	}
	if _, err := conn.Exec("INSERT INTO public_keys (id, algorithm, data, comment) VALUES (1, 'ssh-ed25519', 'AAAAdata', 'alice')"); err != nil {
		t.Fatalf("seed public_key: %v", err)
	}
	if _, err := conn.Exec("INSERT INTO links (account_id, public_key_id) VALUES (1, 1)"); err != nil {
		t.Fatalf("seed link: %v", err)
	}
	seedSystemKey(t, conn)
}

// assertBridgedCorrectly holds for every stage: every migration is recorded,
// the legacy-only tables are gone, and the seeded account/key/link survived
// (or was reshaped and backfilled) correctly.
func assertBridgedCorrectly(t *testing.T, bunDB *bun.DB) {
	t.Helper()

	names := appliedMigrationNames(t, bunDB)
	if len(names) != migrationCount || names[0] != migrations.BaselineName {
		t.Fatalf("unexpected applied migrations: %v", names)
	}

	for _, table := range []string{
		"account_keys", "system_keys", "known_hosts", "bootstrap_sessions", "schema_migrations",
	} {
		var exists int
		err := bunDB.QueryRow("SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = ?", table).Scan(&exists)
		if err != sql.ErrNoRows {
			t.Errorf("expected %s to be dropped, got err=%v", table, err)
		}
	}

	var host, data, deployMethod, deploySecret string
	err := bunDB.QueryRow(`
		SELECT a.host, a.deploy_method, a.deploy_secret, pk.data
		FROM accounts a
		JOIN links l ON l.account_id = a.id
		JOIN public_keys pk ON pk.id = l.public_key_id
		WHERE a.id = 1
	`).Scan(&host, &deployMethod, &deploySecret, &data)
	if err != nil {
		t.Fatalf("query reshaped/linked data: %v", err)
	}
	if host != "example.com" || data != "AAAAdata" {
		t.Fatalf("unexpected data after bridging: host=%q data=%q", host, data)
	}
	if deployMethod != "ssh" || deploySecret != "PRIV-ACTIVE" {
		t.Fatalf("account not backfilled: deploy_method=%q deploy_secret=%q", deployMethod, deploySecret)
	}
}

// TestOpen_FileBackedLegacyStages runs the real, public Open() entrypoint
// against genuine on-disk SQLite files frozen at each legacy migration
// stage - unlike the in-memory bridge tests in open_test.go, this exercises
// actual file I/O and the full driver/connection-pool setup in Open(), not
// just the internal runMigrations helper.
func TestOpen_FileBackedLegacyStages(t *testing.T) {
	for _, stage := range legacyStages {
		t.Run(stage.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "keymaster.db")

			seedConn, err := sql.Open("sqlite", path)
			if err != nil {
				t.Fatalf("open seed connection: %v", err)
			}
			applyLegacyMigrationsThrough(t, seedConn, stage.throughVersion)
			if stage.oldShape {
				seedLegacyShapeData(t, seedConn)
			} else {
				seedNewShapeData(t, seedConn)
			}
			if err := seedConn.Close(); err != nil {
				t.Fatalf("close seed connection: %v", err)
			}

			bunDB, err := Open("sqlite", path)
			if err != nil {
				t.Fatalf("Open: %v", err)
			}
			assertBridgedCorrectly(t, bunDB)
			if err := bunDB.Close(); err != nil {
				t.Fatalf("close bunDB: %v", err)
			}

			// simulate an app restart against the same on-disk file
			bunDB2, err := Open("sqlite", path)
			if err != nil {
				t.Fatalf("second Open (restart): %v", err)
			}
			defer bunDB2.Close()
			assertBridgedCorrectly(t, bunDB2)
		})
	}
}
