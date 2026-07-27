// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package migrations

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net"
	"strings"

	"github.com/uptrace/bun"
	"golang.org/x/crypto/ssh"
)

// legacyAuditRow is the audit_log row as this migration reads it: id plus the
// raw details text, before it is rewritten into the JSON shape the live model
// expects.
type legacyAuditRow struct {
	bun.BaseModel `bun:"table:audit_log"`

	ID      int    `bun:"id"`
	Details string `bun:"details"`
}

// auditDetail mirrors client.AuditLogDetail's JSON encoding. It is duplicated
// here rather than imported so the migration keeps producing this exact shape
// however the client type evolves.
type auditDetail struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// legacyKnownHost is the known_hosts row this migration reads. The table is
// dropped by 00000000000004, so this is the last chance to salvage it.
type legacyKnownHost struct {
	bun.BaseModel `bun:"table:known_hosts"`

	Hostname string `bun:"hostname"`
	Key      string `bun:"key"`
}

// legacyAccountHost identifies an account by the address its host key would
// have been stored under.
type legacyAccountHost struct {
	bun.BaseModel `bun:"table:accounts"`

	ID   int    `bun:"id"`
	Host string `bun:"host"`
	Port string `bun:"port"`
}

// sshSecret and sshCache mirror the ssh connector's JSON encodings. They are
// duplicated here for the same reason as auditDetail: the migration must keep
// producing these exact shapes however the connector types evolve.
//
// The public key is derived and stored here rather than left to the connector:
// it is what ends up in authorized_keys, and the connector deliberately never
// re-derives it so that file stays byte-stable. The authorized_keys hash is
// still deliberately absent: nothing has been deployed through the new stack
// yet, so every account should read as dirty.
type sshSecret struct {
	PrivateKey string `json:"private_key"`
	PublicKey  string `json:"public_key"`
}

type sshCache struct {
	KnownHost string `json:"known_host,omitempty"`
}

func init() {
	Migrations.MustRegister(backfillLegacyDataUp, backfillLegacyDataDown)
}

// backfillLegacyDataUp fills in the columns the bunrewrite model added but the
// legacy reshape (legacymigrate's 000005) left at their defaults, and converts
// legacy audit details into the JSON the live model reads. Every statement is
// scoped to rows still holding a default/legacy value, so this is a no-op on a
// fresh install and re-runnable on a partially converted one.
func backfillLegacyDataUp(ctx context.Context, db *bun.DB) error {
	if err := backfillAccountDeployment(ctx, db); err != nil {
		return err
	}
	if err := backfillAccountDeployCache(ctx, db); err != nil {
		return err
	}
	return backfillAuditDetails(ctx, db)
}

func backfillAccountDeployment(ctx context.Context, db *bun.DB) error {
	// Legacy accounts predate deploy methods; they were all plain SSH, and
	// connector.Resolve("") fails for every one of them until this is set.
	if _, err := db.NewUpdate().Table("accounts").
		Set("deploy_method = ?", "ssh").
		Where("deploy_method = ?", "").
		Exec(ctx); err != nil {
		return err
	}

	// Legacy accounts had no port column. canonicalSSHAddress already treats 0
	// as 22, but storing it keeps the value the UI shows honest.
	if _, err := db.NewUpdate().Table("accounts").
		Set("port = ?", "22").
		Where("port = ?", "").
		Exec(ctx); err != nil {
		return err
	}

	// Prefer the active system key, falling back to the highest serial. Boolean
	// DESC puts true first on sqlite, postgres and mysql alike.
	var privateKey string
	err := db.NewSelect().Table("system_keys").
		Column("private_key").
		OrderExpr("is_active DESC, serial DESC").
		Limit(1).
		Scan(ctx, &privateKey)
	if errors.Is(err, sql.ErrNoRows) {
		// Fresh install, or a legacy one that never generated a system key.
		return nil
	}
	if err != nil {
		return err
	}
	if privateKey == "" {
		return nil
	}

	// The legacy schema kept exactly one shared SSH identity in system_keys and
	// every deployment used it. The bunrewrite model has no system_keys table:
	// the credential lives per account, in accounts.deploy_secret, so the one
	// legacy key is fanned out into every account row. Accounts created or
	// edited afterwards carry their own secret, which is the intended model;
	// the duplication here is only what the legacy data translates to.
	//
	// What the fan-out leaves for the client to deal with:
	//   - The private key is stored in plaintext once per account row.
	//   - Legacy RotateSystemKey deactivated one row and inserted a new one.
	//     The equivalent is an N-row rewrite that no client implements yet.
	//   - accounts.deploy_secret_rollback exists for a two-phase secret swap
	//     and is written by nobody, so an interrupted rewrite has no recovery
	//     path.
	//   - accounts.serial still records the system key serial last deployed,
	//     but nothing maps that serial back to a key any more.
	//
	// An unparsable or encrypted key leaves the public key empty: there is no
	// passphrase in the legacy schema to unlock one with, and guessing is not an
	// option when the result would be written into authorized_keys. Such an
	// account reports a missing public key at deploy time until it is saved again.
	publicKey := ""
	if signer, err := ssh.ParsePrivateKey([]byte(privateKey)); err == nil {
		publicKey = strings.TrimSpace(string(ssh.MarshalAuthorizedKey(signer.PublicKey())))
	}

	// Marshalling in Go rather than assembling JSON in SQL keeps the PEM's
	// newlines escaped correctly.
	encoded, err := json.Marshal(sshSecret{privateKey, publicKey})
	if err != nil {
		return err
	}
	_, err = db.NewUpdate().Table("accounts").
		Set("deploy_secret = ?", string(encoded)).
		Where("deploy_secret = ?", "").
		Exec(ctx)
	return err
}

// backfillAccountDeployCache salvages the legacy known_hosts table into each
// account's deploy_cache before 00000000000004 drops it. The authorized_keys
// hash is deliberately left unset: nothing has been deployed through the new
// stack yet, so every account should still report dirty and get a real deploy.
func backfillAccountDeployCache(ctx context.Context, db *bun.DB) error {
	var knownHosts []legacyKnownHost
	if err := db.NewSelect().Model(&knownHosts).Column("hostname", "key").Scan(ctx); err != nil {
		return err
	}
	if len(knownHosts) == 0 {
		return nil
	}

	byHostname := make(map[string]string, len(knownHosts))
	for _, knownHost := range knownHosts {
		byHostname[strings.TrimSpace(knownHost.Hostname)] = strings.TrimSpace(knownHost.Key)
	}

	var accounts []legacyAccountHost
	if err := db.NewSelect().Model(&accounts).
		Column("id", "host", "port").
		Where("deploy_cache = ?", "").
		Scan(ctx); err != nil {
		return err
	}

	for _, account := range accounts {
		host := strings.TrimSpace(account.Host)
		port := strings.TrimSpace(account.Port)
		if port == "" {
			// Legacy accounts had no port column; core/deploy already treated a
			// missing port as 22 when it canonicalized the hostname.
			port = "22"
		}

		// The normal verification path stored a canonical host:port, while the
		// pinned-key path stored the host alone. Prefer the canonical form.
		key, ok := byHostname[net.JoinHostPort(host, port)]
		if !ok {
			key, ok = byHostname[host]
		}
		if !ok || key == "" {
			continue
		}

		// The connector pins host keys by comparing the canonical authorized_keys
		// rendering, and refuses to parse a cache holding anything else. The legacy
		// writer produced that form, but a restored backup can hold whatever was in
		// it, so re-render here and skip what does not parse; a skipped row just
		// prompts for the host key on the next deploy.
		hostKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(key))
		if err != nil {
			continue
		}

		encoded, err := json.Marshal(sshCache{strings.TrimSpace(string(ssh.MarshalAuthorizedKey(hostKey)))})
		if err != nil {
			return err
		}
		if _, err := db.NewUpdate().Table("accounts").
			Set("deploy_cache = ?", string(encoded)).
			Where("id = ?", account.ID).
			Exec(ctx); err != nil {
			return err
		}
	}

	return nil
}

// backfillAuditDetails rewrites legacy free-text audit details (e.g.
// "account: root@example.com") as the JSON array the live AuditLogModel scans
// into, so ListAuditLogs does not fail on pre-cutover rows. Rows already in the
// new format are skipped, which also makes a re-run harmless.
func backfillAuditDetails(ctx context.Context, db *bun.DB) error {
	var rows []legacyAuditRow
	err := db.NewSelect().
		Model(&rows).
		Column("id", "details").
		Where("details IS NOT NULL").
		Where("details <> ?", "").
		// A JSON array is the new format; 'null' is what bun writes for a nil
		// details slice. Neither needs converting.
		Where("details NOT LIKE ?", "[%").
		Where("details <> ?", "null").
		Scan(ctx)
	if err != nil {
		return err
	}

	for _, row := range rows {
		// Marshalling in Go rather than assembling JSON in SQL keeps the
		// escaping of quotes and backslashes in the original text correct.
		encoded, err := json.Marshal([]auditDetail{{"legacy", row.Details}})
		if err != nil {
			return err
		}
		if _, err := db.NewUpdate().Table("audit_log").
			Set("details = ?", string(encoded)).
			Where("id = ?", row.ID).
			Exec(ctx); err != nil {
			return err
		}
	}

	return nil
}

// backfillLegacyDataDown is a no-op: a backfill cannot tell what it filled in
// from what was already there, so there is nothing safe to undo.
func backfillLegacyDataDown(_ context.Context, _ *bun.DB) error {
	return nil
}
