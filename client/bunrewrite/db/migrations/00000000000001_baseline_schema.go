// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package migrations

import (
	"context"
	"database/sql"
	"time"

	"github.com/uptrace/bun"
)

// The row structs below reproduce the full schema as it exists at the
// SQL -> Go migration cutover (equivalent to legacymigrate's 000001-000005
// combined). They are intentionally separate from the live models in the
// parent db package - a migration must reproduce schema history exactly,
// independent of how application models evolve afterwards.

type baselineAccount struct {
	bun.BaseModel `bun:"table:accounts"`

	ID                   int            `bun:"id,pk,autoincrement"`
	Username             string         `bun:"username,notnull,unique:accounts_username_host"`
	Host                 string         `bun:"host,notnull,default:'',unique:accounts_username_host"`
	Port                 string         `bun:"port,notnull,default:''"`
	Serial               int            `bun:"serial,notnull,default:0"`
	IsActive             bool           `bun:"is_active,notnull,default:true"`
	IsDirty              bool           `bun:"is_dirty,notnull,default:true"`
	DeployMethod         string         `bun:"deploy_method,notnull,default:''"`
	DeploySecret         string         `bun:"deploy_secret,type:text,notnull,default:''"`
	DeploySecretRollback sql.NullString `bun:"deploy_secret_rollback,type:text"`
	DeployCache          string         `bun:"deploy_cache,type:text,notnull,default:''"`
}

type baselinePublicKey struct {
	bun.BaseModel `bun:"table:public_keys"`

	ID        int          `bun:"id,pk,autoincrement"`
	Algorithm string       `bun:"algorithm,notnull"`
	Data      string       `bun:"data,type:text,notnull"`
	Comment   string       `bun:"comment,notnull,unique"`
	IsGlobal  bool         `bun:"is_global,notnull,default:false"`
	ExpiresAt sql.NullTime `bun:"expires_at"`
}

type baselineLink struct {
	bun.BaseModel `bun:"table:links"`

	AccountID   int          `bun:"account_id,pk"`
	PublicKeyID int          `bun:"public_key_id,pk"`
	ExpiresAt   sql.NullTime `bun:"expires_at"`
}

type baselineAuditLog struct {
	bun.BaseModel `bun:"table:audit_log"`

	ID         int            `bun:"id,pk,autoincrement"`
	Timestamp  time.Time      `bun:"timestamp,notnull,nullzero,default:current_timestamp"`
	Username   string         `bun:"username,notnull"`
	Hostname   sql.NullString `bun:"hostname"`
	ClientImpl sql.NullString `bun:"client_impl"`
	Referrer   sql.NullString `bun:"referrer"`
	Action     string         `bun:"action,notnull"`
	Details    sql.NullString `bun:"details,type:text"`
}

type baselineSystemKey struct {
	bun.BaseModel `bun:"table:system_keys"`

	ID         int    `bun:"id,pk,autoincrement"`
	Serial     int    `bun:"serial,notnull,unique"`
	PublicKey  string `bun:"public_key,type:text,notnull"`
	PrivateKey string `bun:"private_key,type:text,notnull"`
	IsActive   bool   `bun:"is_active,notnull,default:false"`
}

type baselineKnownHost struct {
	bun.BaseModel `bun:"table:known_hosts"`

	Hostname string `bun:"hostname,pk"`
	Key      string `bun:"key,type:text,notnull"`
}

type baselineBootstrapSession struct {
	bun.BaseModel `bun:"table:bootstrap_sessions"`

	ID            string         `bun:"id,pk"`
	Username      string         `bun:"username,notnull"`
	Hostname      string         `bun:"hostname,notnull"`
	Label         sql.NullString `bun:"label,type:text"`
	Tags          sql.NullString `bun:"tags,type:text"`
	TempPublicKey string         `bun:"temp_public_key,type:text,notnull"`
	CreatedAt     time.Time      `bun:"created_at,notnull,nullzero,default:current_timestamp"`
	ExpiresAt     time.Time      `bun:"expires_at,notnull,nullzero"`
	Status        string         `bun:"status,notnull,default:'active'"`
}

func init() {
	Migrations.MustRegister(baselineUp, baselineDown)
}

func baselineUp(ctx context.Context, db *bun.DB) error {
	for _, model := range []any{
		(*baselineAccount)(nil),
		(*baselinePublicKey)(nil),
		(*baselineAuditLog)(nil),
		(*baselineSystemKey)(nil),
		(*baselineKnownHost)(nil),
		(*baselineBootstrapSession)(nil),
	} {
		if _, err := db.NewCreateTable().Model(model).IfNotExists().Exec(ctx); err != nil {
			return err
		}
	}

	if _, err := db.NewCreateTable().Model((*baselineLink)(nil)).
		ForeignKey(`(account_id) REFERENCES accounts (id) ON DELETE CASCADE`).
		ForeignKey(`(public_key_id) REFERENCES public_keys (id) ON DELETE CASCADE`).
		IfNotExists().Exec(ctx); err != nil {
		return err
	}

	if _, err := db.NewCreateIndex().Model((*baselineBootstrapSession)(nil)).
		Index("idx_bootstrap_sessions_expires_at").Column("expires_at").
		IfNotExists().Exec(ctx); err != nil {
		return err
	}
	if _, err := db.NewCreateIndex().Model((*baselineBootstrapSession)(nil)).
		Index("idx_bootstrap_sessions_status").Column("status").
		IfNotExists().Exec(ctx); err != nil {
		return err
	}

	return nil
}

func baselineDown(ctx context.Context, db *bun.DB) error {
	// Reverses in dependency order. Rolling back the baseline on a legacy-
	// bridged install destroys real data - true of rolling back to "before
	// any schema existed" in any migration tool, not a defect specific to
	// baselining (see bridge.go).
	for _, model := range []any{
		(*baselineBootstrapSession)(nil),
		(*baselineKnownHost)(nil),
		(*baselineSystemKey)(nil),
		(*baselineAuditLog)(nil),
		(*baselineLink)(nil),
		(*baselinePublicKey)(nil),
		(*baselineAccount)(nil),
	} {
		if _, err := db.NewDropTable().Model(model).IfExists().Exec(ctx); err != nil {
			return err
		}
	}
	return nil
}
