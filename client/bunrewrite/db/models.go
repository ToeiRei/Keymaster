// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// Package db holds the Bun ORM models and schema migrations owned by the
// bunrewrite client. These are vendored (copied and adapted) from core/db so
// the rewrite is self-contained and does not depend on the legacy package.
package db

import (
	"database/sql"
	"time"

	"github.com/toeirei/keymaster/client"
	"github.com/uptrace/bun"
)

type AccountModel struct {
	bun.BaseModel `bun:"table:accounts"`

	ID           int    `bun:"id,pk,autoincrement"`
	Username     string `bun:"username"`
	Host         string `bun:"host"`
	Port         string `bun:"port"`
	IsActive     bool   `bun:"is_active"`
	IsDirty      bool   `bun:"is_dirty"`
	DeployMethod string `bun:"deploy_method"`
	DeploySecret string `bun:"deploy_secret"`

	Links []LinkModel `bun:"rel:has-many,join:id=account_id"`
}

type PublicKeyModel struct {
	bun.BaseModel `bun:"table:public_keys"`

	ID        int          `bun:"id,pk,autoincrement"`
	Algorithm string       `bun:"algorithm"`
	Data      string       `bun:"data"`
	Comment   string       `bun:"comment"`
	ExpiresAt sql.NullTime `bun:"expires_at"`
	IsGlobal  bool         `bun:"is_global"`

	Links []LinkModel `bun:"rel:has-many,join:id=public_key_id"`
}

type LinkModel struct {
	bun.BaseModel `bun:"table:links"`

	AccountId   int          `bun:"account_id,pk"`
	PublicKeyId int          `bun:"public_key_id,pk"`
	ExpiresAt   sql.NullTime `bun:"expires_at"`

	Account   *AccountModel   `bun:"rel:belongs-to,join:account_id=id"`
	PublicKey *PublicKeyModel `bun:"rel:belongs-to,join:public_key_id=id"`
}

type AuditLogModel struct {
	bun.BaseModel `bun:"table:audit_log"`

	ID         int                    `bun:"id,pk,autoincrement"`
	Timestamp  time.Time              `bun:"timestamp"`
	Username   string                 `bun:"username"`
	Hostname   sql.NullString         `bun:"hostname"`
	ClientImpl sql.NullString         `bun:"client_impl"`
	Referrer   sql.NullString         `bun:"referrer"`
	Action     string                 `bun:"action"`
	Details    client.AuditLogDetails `bun:"details,type:text"`
}
