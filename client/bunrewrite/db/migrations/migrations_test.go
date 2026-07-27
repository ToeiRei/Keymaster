// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package migrations

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"encoding/pem"
	"slices"
	"strings"
	"testing"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/migrate"
	"golang.org/x/crypto/ssh"

	_ "modernc.org/sqlite"
)

// testHostKey renders a real host key the way the legacy known_hosts writer did,
// with the trailing newline MarshalAuthorizedKey appends. The backfill only
// salvages keys it can parse, so these seeds cannot be stand-in strings.
func testHostKey(t *testing.T) string {
	t.Helper()

	publicKey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate host key: %v", err)
	}
	hostKey, err := ssh.NewPublicKey(publicKey)
	if err != nil {
		t.Fatalf("wrap host key: %v", err)
	}
	return string(ssh.MarshalAuthorizedKey(hostKey))
}

// testPrivateKeyPEM is a real system key, so the backfill can derive the public
// key it stores alongside it.
func testPrivateKeyPEM(t *testing.T) string {
	t.Helper()

	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	block, err := ssh.MarshalPrivateKey(privateKey, "")
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}
	return string(pem.EncodeToMemory(block))
}

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

// TestBaseline_CreatesCutoverSchema pins the baseline to the schema as it stood
// at the SQL -> Go cutover, including the tables later migrations drop again.
// bridge.go marks this migration applied against legacy databases that do have
// all seven, so it must keep creating all seven.
func TestBaseline_CreatesCutoverSchema(t *testing.T) {
	db := openMemBunDB(t)

	if err := baselineUp(context.Background(), db); err != nil {
		t.Fatalf("baselineUp: %v", err)
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
		t.Error("account_keys should never be created by the baseline")
	}
}

func TestMigrate_LeavesOnlyTheModelTables(t *testing.T) {
	db := openMemBunDB(t)
	migrateAll(t, db)

	for _, table := range []string{"accounts", "public_keys", "links", "audit_log"} {
		if !tableExists(t, db, table) {
			t.Errorf("expected table %q to exist", table)
		}
	}
	for _, table := range []string{
		"account_keys", "system_keys", "known_hosts", "bootstrap_sessions",
	} {
		if tableExists(t, db, table) {
			t.Errorf("expected table %q to be dropped", table)
		}
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

// connectorSecretPrivateKey reads back the private key from an account's
// connector_secret JSON, so the tests assert on the value rather than on the
// exact encoding. Every caller runs after migrateAll, hence the renamed column.
func connectorSecretPrivateKey(t *testing.T, db *bun.DB, username string) string {
	t.Helper()

	var raw string
	if err := db.QueryRow("SELECT connector_secret FROM accounts WHERE username = ?", username).Scan(&raw); err != nil {
		t.Fatalf("query connector_secret for %s: %v", username, err)
	}
	if raw == "" {
		return ""
	}

	var secret sshSecret
	if err := json.Unmarshal([]byte(raw), &secret); err != nil {
		t.Fatalf("connector_secret for %s is not valid JSON (%q): %v", username, raw, err)
	}
	return secret.PrivateKey
}

func TestBackfill_CopiesActiveSystemKeyIntoConnectorSecret(t *testing.T) {
	db := openMemBunDB(t)
	seedCutoverSchema(t, db)

	// a PEM's newlines and slashes must survive the trip through JSON
	activeKey := "-----BEGIN OPENSSH PRIVATE KEY-----\nb3Blb/nNz\n-----END OPENSSH PRIVATE KEY-----\n"

	// higher serial but inactive: the active key must still win
	exec(t, db, "INSERT INTO system_keys (serial, public_key, private_key, is_active) VALUES (2, 'pub2', 'PRIV-STALE', false)")
	exec(t, db, "INSERT INTO system_keys (serial, public_key, private_key, is_active) VALUES (1, 'pub1', ?, true)", activeKey)
	exec(t, db, "INSERT INTO accounts (username, host) VALUES ('root', 'example.com')")
	// an account already configured by the new client must not be clobbered
	exec(t, db, `INSERT INTO accounts (username, host, port, deploy_method, deploy_secret) VALUES ('deploy', 'other.com', '2222', 'mock', '{"succeed":true}')`)

	migrateAll(t, db)

	if got := connectorSecretPrivateKey(t, db, "root"); got != activeKey {
		t.Fatalf("private key not backfilled: %q", got)
	}

	var con, port string
	if err := db.QueryRow("SELECT connector, port FROM accounts WHERE username = 'root'").
		Scan(&con, &port); err != nil {
		t.Fatalf("query backfilled account: %v", err)
	}
	if con != "ssh" || port != "22" {
		t.Fatalf("account not backfilled: connector=%q port=%q", con, port)
	}

	var secret string
	if err := db.QueryRow("SELECT connector_secret, connector, port FROM accounts WHERE username = 'deploy'").
		Scan(&secret, &con, &port); err != nil {
		t.Fatalf("query configured account: %v", err)
	}
	if secret != `{"succeed":true}` || con != "mock" || port != "2222" {
		t.Fatalf("configured account was clobbered: secret=%q connector=%q port=%q", secret, con, port)
	}
}

func TestBackfill_FallsBackToHighestSerialWhenNoneActive(t *testing.T) {
	db := openMemBunDB(t)
	seedCutoverSchema(t, db)

	exec(t, db, "INSERT INTO system_keys (serial, public_key, private_key, is_active) VALUES (1, 'pub1', 'PRIV-OLD', false)")
	exec(t, db, "INSERT INTO system_keys (serial, public_key, private_key, is_active) VALUES (3, 'pub3', 'PRIV-NEWEST', false)")
	exec(t, db, "INSERT INTO accounts (username, host) VALUES ('root', 'example.com')")

	migrateAll(t, db)

	if got := connectorSecretPrivateKey(t, db, "root"); got != "PRIV-NEWEST" {
		t.Fatalf("private key = %q, want PRIV-NEWEST", got)
	}
}

// connectorCache reads back an account's connector_cache as the ssh
// connector's shape.
func connectorCache(t *testing.T, db *bun.DB, username string) (string, sshCache) {
	t.Helper()

	var raw string
	if err := db.QueryRow("SELECT connector_cache FROM accounts WHERE username = ?", username).Scan(&raw); err != nil {
		t.Fatalf("query connector_cache for %s: %v", username, err)
	}
	if raw == "" {
		return raw, sshCache{}
	}

	var cache sshCache
	if err := json.Unmarshal([]byte(raw), &cache); err != nil {
		t.Fatalf("connector_cache for %s is not valid JSON (%q): %v", username, raw, err)
	}
	return raw, cache
}

func TestBackfill_SalvagesKnownHostsIntoConnectorCache(t *testing.T) {
	db := openMemBunDB(t)
	seedCutoverSchema(t, db)

	canonical, hostonly, defaultport := testHostKey(t), testHostKey(t), testHostKey(t)

	// ssh.MarshalAuthorizedKey leaves a trailing newline; the backfill trims it
	exec(t, db, "INSERT INTO known_hosts (hostname, key) VALUES ('canonical.com:2222', ?)", canonical)
	// the pinned-key path stored the host without a port
	exec(t, db, "INSERT INTO known_hosts (hostname, key) VALUES ('hostonly.com', ?)", strings.TrimSpace(hostonly))

	exec(t, db, "INSERT INTO accounts (username, host, port) VALUES ('a', 'canonical.com', '2222')")
	exec(t, db, "INSERT INTO accounts (username, host) VALUES ('b', 'hostonly.com')")
	// a legacy account with no port must still match a canonical :22 entry
	exec(t, db, "INSERT INTO known_hosts (hostname, key) VALUES ('defaultport.com:22', ?)", defaultport)
	exec(t, db, "INSERT INTO accounts (username, host) VALUES ('c', 'defaultport.com')")
	// no known host: the cache must stay empty so the account reports dirty
	exec(t, db, "INSERT INTO accounts (username, host, port) VALUES ('d', 'untrusted.com', '22')")
	// a key the connector could not parse is worse than none: it would make the
	// account unreadable rather than merely unverified, so it is skipped
	exec(t, db, "INSERT INTO known_hosts (hostname, key) VALUES ('corrupt.com:22', 'ssh-ed25519 AAAAcorrupt')")
	exec(t, db, "INSERT INTO accounts (username, host) VALUES ('e', 'corrupt.com')")

	migrateAll(t, db)

	for _, tc := range []struct{ username, wantKnownHost string }{
		{"a", strings.TrimSpace(canonical)},
		{"b", strings.TrimSpace(hostonly)},
		{"c", strings.TrimSpace(defaultport)},
	} {
		_, cache := connectorCache(t, db, tc.username)
		if cache.KnownHost != tc.wantKnownHost {
			t.Errorf("account %s known_host = %q, want %q", tc.username, cache.KnownHost, tc.wantKnownHost)
		}
	}

	if raw, _ := connectorCache(t, db, "d"); raw != "" {
		t.Errorf("account without a known host got a cache: %q", raw)
	}
	if raw, _ := connectorCache(t, db, "e"); raw != "" {
		t.Errorf("account with an unparsable known host got a cache: %q", raw)
	}
}

// TestBackfill_DerivesPublicKeyIntoConnectorSecret pins that the migration, not
// the connector, is what puts a public key in the secret: the connector never
// re-derives one, so a backfilled account would otherwise have nothing to write
// into authorized_keys.
func TestBackfill_DerivesPublicKeyIntoConnectorSecret(t *testing.T) {
	db := openMemBunDB(t)
	seedCutoverSchema(t, db)

	privateKey := testPrivateKeyPEM(t)
	exec(t, db, "INSERT INTO system_keys (serial, public_key, private_key, is_active) VALUES (1, 'pub1', ?, true)", privateKey)
	exec(t, db, "INSERT INTO accounts (username, host) VALUES ('root', 'example.com')")

	migrateAll(t, db)

	var raw string
	if err := db.QueryRow("SELECT connector_secret FROM accounts WHERE username = 'root'").Scan(&raw); err != nil {
		t.Fatalf("query connector_secret: %v", err)
	}
	var secret sshSecret
	if err := json.Unmarshal([]byte(raw), &secret); err != nil {
		t.Fatalf("connector_secret is not valid JSON (%q): %v", raw, err)
	}

	signer, err := ssh.ParsePrivateKey([]byte(privateKey))
	if err != nil {
		t.Fatalf("parse test key: %v", err)
	}
	if want := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(signer.PublicKey()))); secret.PublicKey != want {
		t.Fatalf("public key = %q, want %q", secret.PublicKey, want)
	}
}

func TestBackfill_LeavesExistingConnectorCacheAlone(t *testing.T) {
	db := openMemBunDB(t)
	seedCutoverSchema(t, db)

	exec(t, db, "INSERT INTO known_hosts (hostname, key) VALUES ('example.com:22', 'ssh-ed25519 AAAAlegacy')")
	exec(t, db, `INSERT INTO accounts (username, host, port, deploy_cache) VALUES ('root', 'example.com', '22', '{"authorized_keys_hash":"abc123"}')`)

	migrateAll(t, db)

	if raw, _ := connectorCache(t, db, "root"); raw != `{"authorized_keys_hash":"abc123"}` {
		t.Fatalf("existing cache was rewritten: %q", raw)
	}
}

func TestBackfill_NoSystemKeysLeavesConnectorSecretEmpty(t *testing.T) {
	db := openMemBunDB(t)
	seedCutoverSchema(t, db)

	exec(t, db, "INSERT INTO accounts (username, host) VALUES ('root', 'example.com')")

	migrateAll(t, db)

	var secret, con string
	if err := db.QueryRow("SELECT connector_secret, connector FROM accounts WHERE username = 'root'").
		Scan(&secret, &con); err != nil {
		t.Fatalf("query account: %v", err)
	}
	// Nothing to copy, so no JSON is written at all; the column stays at its
	// default and the account has no secret rather than an empty one.
	if secret != "" {
		t.Fatalf("connector_secret = %q, want empty", secret)
	}
	// the connector backfill runs regardless of whether a key was found
	if con != "ssh" {
		t.Fatalf("connector = %q, want ssh", con)
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

func accountColumns(t *testing.T, db *bun.DB) []string {
	t.Helper()
	rows, err := db.Query("SELECT name FROM pragma_table_info('accounts')")
	if err != nil {
		t.Fatalf("table_info(accounts): %v", err)
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var column string
		if err := rows.Scan(&column); err != nil {
			t.Fatalf("scan: %v", err)
		}
		columns = append(columns, column)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	return columns
}

// TestRenameConnectorColumns_RenamesAndKeepsValues covers both halves of the
// rename: no deploy_* column survives it, and what the earlier migrations wrote
// under the old names is still there under the new ones.
func TestRenameConnectorColumns_RenamesAndKeepsValues(t *testing.T) {
	db := openMemBunDB(t)
	seedCutoverSchema(t, db)

	exec(t, db, `INSERT INTO accounts (username, host, deploy_method, deploy_secret, deploy_cache) VALUES ('root', 'example.com', 'mock', '{"succeed":true}', '{"authorized_keys_hash":"abc123"}')`)

	migrateAll(t, db)

	columns := accountColumns(t, db)
	for _, column := range []string{
		"connector", "connector_secret", "connector_secret_rollback", "connector_cache",
	} {
		if !slices.Contains(columns, column) {
			t.Errorf("expected column %q on accounts, got %v", column, columns)
		}
	}
	for _, column := range columns {
		if strings.HasPrefix(column, "deploy_") {
			t.Errorf("column %q survived the rename", column)
		}
	}

	var con, secret, cache string
	if err := db.QueryRow("SELECT connector, connector_secret, connector_cache FROM accounts WHERE username = 'root'").
		Scan(&con, &secret, &cache); err != nil {
		t.Fatalf("query renamed account: %v", err)
	}
	if con != "mock" || secret != `{"succeed":true}` || cache != `{"authorized_keys_hash":"abc123"}` {
		t.Fatalf("values lost in the rename: connector=%q secret=%q cache=%q", con, secret, cache)
	}
}

func TestRenameConnectorColumns_DownRestoresTheOldNames(t *testing.T) {
	ctx := context.Background()
	db := openMemBunDB(t)
	seedCutoverSchema(t, db)

	if err := renameConnectorColumnsUp(ctx, db); err != nil {
		t.Fatalf("up: %v", err)
	}
	if err := renameConnectorColumnsDown(ctx, db); err != nil {
		t.Fatalf("down: %v", err)
	}

	columns := accountColumns(t, db)
	for _, column := range []string{
		"deploy_method", "deploy_secret", "deploy_secret_rollback", "deploy_cache",
	} {
		if !slices.Contains(columns, column) {
			t.Errorf("expected column %q back on accounts, got %v", column, columns)
		}
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
