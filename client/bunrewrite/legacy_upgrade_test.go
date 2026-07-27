// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package bunrewrite

import (
	"context"
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/toeirei/keymaster/client"
	"github.com/toeirei/keymaster/client/bunrewrite/db"
	"github.com/toeirei/keymaster/config"

	_ "modernc.org/sqlite"
)

// seedLegacyDatabase writes an on-disk SQLite file in the original (000001)
// schema, populated the way a real pre-bunrewrite install would be, and
// records that migration so Open() takes the legacy bridge path. It returns the
// system key it seeded: the connector validates a secret when it parses one, so
// reading the migrated account back only works with a real key.
func seedLegacyDatabase(t *testing.T, path string) string {
	t.Helper()

	legacyPrivateKey := testPrivateKeyPEM(t)

	conn, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open seed connection: %v", err)
	}
	defer conn.Close()

	schema, err := os.ReadFile(filepath.Join("db", "legacymigrate", "migrations", "sqlite", "000001_create_initial_tables.up.sql"))
	if err != nil {
		t.Fatalf("read legacy schema: %v", err)
	}

	stmts := []string{
		`CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY, applied_at TIMESTAMP)`,
		string(schema),
		`INSERT INTO schema_migrations (version, applied_at) VALUES ('000001_create_initial_tables', CURRENT_TIMESTAMP)`,
		`INSERT INTO accounts (id, username, hostname, serial, is_active) VALUES (1, 'root', 'example.com', 3, 1)`,
		`INSERT INTO public_keys (id, algorithm, key_data, comment, is_global) VALUES (1, 'ssh-ed25519', 'AAAAC3NzaC1lZDI1NTE5alice', 'alice', 0)`,
		`INSERT INTO account_keys (account_id, key_id) VALUES (1, 1)`,
		`INSERT INTO system_keys (serial, public_key, private_key, is_active) VALUES (3, 'ssh-ed25519 AAAAsystem', '` + legacyPrivateKey + `', 1)`,
		`INSERT INTO audit_log (id, timestamp, username, action, details) VALUES (1, '2026-01-02 03:04:05+00:00', 'jannes', 'ADD_ACCOUNT', 'account: root@example.com')`,
	}
	for _, stmt := range stmts {
		if _, err := conn.Exec(stmt); err != nil {
			t.Fatalf("seed %.40q: %v", stmt, err)
		}
	}
	return legacyPrivateKey
}

// TestLegacyUpgrade_ReadableThroughClient is the end-to-end counterpart to the
// migration tests: it opens the real client against a legacy on-disk database
// and reads it back through the public API, which is where a missed data
// migration actually shows up (an unresolvable connector, a missing deploy
// secret, or audit details that no longer unmarshal).
func TestLegacyUpgrade_ReadableThroughClient(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "keymaster.db")
	legacyPrivateKey := seedLegacyDatabase(t, path)

	cfg := config.Config{
		Database: config.ConfigDatabase{Type: "sqlite", Dsn: path},
		Language: "de",
	}
	c, err := NewBunClient(cfg, log.Default(), "test.keymaster.legacy_upgrade")
	if err != nil {
		t.Fatalf("NewBunClient: %v", err)
	}
	defer c.Close(ctx)

	accounts, err := c.ListAccounts(ctx)
	if err != nil {
		t.Fatalf("ListAccounts: %v", err)
	}
	if len(accounts) != 1 {
		t.Fatalf("expected 1 account, got %d", len(accounts))
	}
	acc := accounts[0]
	if acc.Username != "root" || acc.Host != "example.com" || acc.Port != 22 {
		t.Fatalf("account not reshaped: %+v", acc)
	}
	if acc.Connector != "ssh" {
		t.Fatalf("connector %q would not resolve", acc.Connector)
	}
	// connector_secret holds the ssh connector's JSON shape, so read the key
	// back through the connector rather than comparing the raw column.
	secretFields := acc.ConnectorSecret.Fields()
	if len(secretFields) == 0 || secretFields[0].Key != "private_key" {
		t.Fatalf("unexpected secret fields: %+v", secretFields)
	}
	if secretFields[0].Value != legacyPrivateKey {
		t.Fatalf("system private key not carried into connector_secret: %q", secretFields[0].Value)
	}
	// The serial is no longer read by anything, so it cannot be asserted through
	// the client any more. The migration still has to carry it across, so check
	// the column directly rather than dropping the assertion.
	var serial int
	if err := c.bun.NewSelect().
		Model((*db.AccountModel)(nil)).
		Column("serial").
		Where("id = ?", int(acc.Id)).
		Scan(ctx, &serial); err != nil {
		t.Fatalf("read serial column: %v", err)
	}
	if serial != 3 {
		t.Fatalf("serial not preserved: %d", serial)
	}

	keys, err := c.ListPublicKeysLinkedToAccount(ctx, acc.Id, false)
	if err != nil {
		t.Fatalf("ListPublicKeysLinkedToAccount: %v", err)
	}
	if len(keys) != 1 || keys[0].Comment != "alice" {
		t.Fatalf("account_keys not carried into links: %+v", keys)
	}

	logs, err := c.ListAuditLogs(ctx, 0, 0)
	if err != nil {
		t.Fatalf("ListAuditLogs: %v", err)
	}
	var legacyEntry *client.AuditLog
	for i, entry := range logs {
		if entry.Action == "ADD_ACCOUNT" {
			legacyEntry = &logs[i]
			break
		}
	}
	if legacyEntry == nil {
		t.Fatalf("legacy audit entry missing from %+v", logs)
	}
	if got, want := legacyEntry.Details.String(), `legacy="account: root@example.com"`; got != want {
		t.Fatalf("legacy audit details = %q, want %q", got, want)
	}
	if legacyEntry.Timestamp.IsZero() {
		t.Error("legacy audit timestamp did not survive")
	}
}
