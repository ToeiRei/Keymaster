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

	_ "modernc.org/sqlite"
)

// openMemSQLite returns an in-memory SQLite DB pinned to a single connection so
// the schema is shared across queries.
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

// applyLegacyMigrations applies every sqlite up-migration EXCEPT the new
// 000005 one and records them in schema_migrations, simulating a database
// created by the legacy core/db migration chain.
func applyLegacyMigrations(t *testing.T, conn *sql.DB) {
	t.Helper()
	if err := ensureSchemaMigrationsTable(conn, "sqlite"); err != nil {
		t.Fatalf("ensure schema_migrations: %v", err)
	}

	entries, err := os.ReadDir(filepath.Join("migrations", "sqlite"))
	if err != nil {
		t.Fatalf("read migrations dir: %v", err)
	}
	var ups []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".up.sql") && !strings.HasPrefix(e.Name(), "000005") {
			ups = append(ups, e.Name())
		}
	}
	sort.Strings(ups)

	for _, name := range ups {
		data, err := os.ReadFile(filepath.Join("migrations", "sqlite", name))
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

// TestLegacyUpgrade verifies that a legacy database (schema at 000004) upgrades
// via 000005: columns are reshaped, data is preserved, and account_keys rows are
// migrated into the new links table.
func TestLegacyUpgrade(t *testing.T) {
	conn := openMemSQLite(t)
	applyLegacyMigrations(t, conn)

	// seed legacy data in the OLD shape (hostname, key_data, account_keys)
	if _, err := conn.Exec("INSERT INTO accounts (id, username, hostname, is_active, is_dirty) VALUES (1, 'root', 'example.com', 1, 1)"); err != nil {
		t.Fatalf("seed account: %v", err)
	}
	if _, err := conn.Exec("INSERT INTO public_keys (id, algorithm, key_data, comment, is_global) VALUES (1, 'ssh-ed25519', 'AAAAdata', 'alice', 0)"); err != nil {
		t.Fatalf("seed public_key: %v", err)
	}
	if _, err := conn.Exec("INSERT INTO account_keys (account_id, key_id) VALUES (1, 1)"); err != nil {
		t.Fatalf("seed account_keys: %v", err)
	}

	// run the (new) migration set: 000001-000004 are already applied, so only 000005 runs
	if err := RunMigrations(conn, "sqlite"); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}

	// accounts reshaped: host backfilled from hostname, new columns present
	var host, deployMethod string
	if err := conn.QueryRow("SELECT host, deploy_method FROM accounts WHERE id = 1").Scan(&host, &deployMethod); err != nil {
		t.Fatalf("query accounts: %v", err)
	}
	if host != "example.com" || deployMethod != "" {
		t.Fatalf("account not reshaped: host=%q deploy_method=%q", host, deployMethod)
	}

	// public_keys reshaped: data backfilled from key_data
	var data string
	if err := conn.QueryRow("SELECT data FROM public_keys WHERE id = 1").Scan(&data); err != nil {
		t.Fatalf("query public_keys: %v", err)
	}
	if data != "AAAAdata" {
		t.Fatalf("public key data not backfilled: %q", data)
	}

	// account_keys migrated into links
	var accountID, publicKeyID int
	if err := conn.QueryRow("SELECT account_id, public_key_id FROM links").Scan(&accountID, &publicKeyID); err != nil {
		t.Fatalf("query links: %v", err)
	}
	if accountID != 1 || publicKeyID != 1 {
		t.Fatalf("link not backfilled from account_keys: account_id=%d public_key_id=%d", accountID, publicKeyID)
	}

	// applying again is a no-op (idempotent version tracking)
	if err := RunMigrations(conn, "sqlite"); err != nil {
		t.Fatalf("RunMigrations second run: %v", err)
	}
}
