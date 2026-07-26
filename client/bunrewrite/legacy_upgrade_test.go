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
	"github.com/toeirei/keymaster/config"

	_ "modernc.org/sqlite"
)

const legacyPrivateKey = "-----BEGIN OPENSSH PRIVATE KEY-----\nnot-a-real-key\n-----END OPENSSH PRIVATE KEY-----\n"

// seedLegacyDatabase writes an on-disk SQLite file in the original (000001)
// schema, populated the way a real pre-bunrewrite install would be, and
// records that migration so Open() takes the legacy bridge path.
func seedLegacyDatabase(t *testing.T, path string) {
	t.Helper()

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
}

// TestLegacyUpgrade_ReadableThroughClient is the end-to-end counterpart to the
// migration tests: it opens the real client against a legacy on-disk database
// and reads it back through the public API, which is where a missed data
// migration actually shows up (an unresolvable connector, a missing deploy
// secret, or audit details that no longer unmarshal).
func TestLegacyUpgrade_ReadableThroughClient(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "keymaster.db")
	seedLegacyDatabase(t, path)

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
	if acc.DeployMethod != "ssh" {
		t.Fatalf("deploy method %q would not resolve to a connector", acc.DeployMethod)
	}
	if acc.DeploySecret != legacyPrivateKey {
		t.Fatalf("system private key not carried into deploy_secret: %q", acc.DeploySecret)
	}
	if acc.Serial != 3 {
		t.Fatalf("serial not preserved: %d", acc.Serial)
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
