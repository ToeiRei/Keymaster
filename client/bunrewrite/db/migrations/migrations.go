// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// Package migrations holds the Go-authored, bun/migrate-driven schema
// migrations for the bunrewrite client, starting from the "baseline" - the
// full schema as of the SQL -> Go cutover. Every migration here defines its
// own small, migration-scoped row structs instead of importing the live
// models from the parent db package: partly to avoid an import cycle (db
// imports this package to build the migrator), but more importantly because
// a migration must reproduce the schema exactly as it existed at the time it
// was authored, independent of how the application's models evolve later.
//
// New migrations: add one small <14-digit-timestamp>_<comment>.go file per
// change, each with `func init() { Migrations.MustRegister(up, down) }`,
// built against the bun query builder so it works identically across
// sqlite/postgres/mysql. Branch on db.Dialect().Name() only for genuinely
// dialect-specific DDL.
package migrations

import "github.com/uptrace/bun/migrate"

// Migrations is the shared registry; every migration file's init() registers
// into it via Migrations.MustRegister(up, down).
var Migrations = migrate.NewMigrations()

// BaselineName is the registered Name of the baseline migration. The
// bunrewrite db package's bridge looks this migration up BY NAME - never by
// position - to mark it applied without running its Up() when adopting a
// pre-existing legacy database.
const BaselineName = "00000000000001"
