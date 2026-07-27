// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/toeirei/keymaster/client/bunrewrite/db/legacymigrate"
	"github.com/toeirei/keymaster/client/bunrewrite/db/migrations"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

// runMigrations brings conn/bunDB up to date using the Go-based migrator,
// bridging a pre-existing legacy (SQL-file) database into the new system
// exactly once.
func runMigrations(ctx context.Context, conn *sql.DB, bunDB *bun.DB, dbType string) error {
	migrator := migrate.NewMigrator(bunDB, migrations.Migrations, migrate.WithMarkAppliedOnSuccess(true))

	if err := migrator.Init(ctx); err != nil {
		return fmt.Errorf("initializing migrator: %w", err)
	}

	if err := bridgeLegacySchema(ctx, conn, migrator, dbType); err != nil {
		return fmt.Errorf("bridging legacy schema: %w", err)
	}

	if _, err := migrator.Migrate(ctx); err != nil {
		return fmt.Errorf("applying migrations: %w", err)
	}
	return nil
}

// bridgeLegacySchema detects which of three starting states conn is in and
// does the minimally-correct thing:
//  1. Already on the new system (bun_migrations has entries) - nothing to do.
//  2. Legacy schema_migrations table exists (mid-chain 000001-000004, or
//     already fully at the old 000005 shape - both handled identically,
//     since the legacy runner is idempotent) - run the frozen legacy runner,
//     then mark ONLY the baseline migration applied without running its
//     Up(), since the schema it would create already exists. Any migration
//     registered after the baseline is left unapplied and runs for real.
//  3. Fresh database (neither table exists) - nothing to do; the baseline's
//     real Up() creates the schema from scratch via the query builder.
func bridgeLegacySchema(ctx context.Context, conn *sql.DB, migrator *migrate.Migrator, dbType string) error {
	applied, err := migrator.AppliedMigrations(ctx)
	if err != nil {
		return err
	}
	if len(applied) > 0 {
		return nil
	}

	hasLegacyTable, err := tableExists(ctx, conn, dbType, "schema_migrations")
	if err != nil {
		return fmt.Errorf("checking for legacy schema_migrations table: %w", err)
	}
	if !hasLegacyTable {
		return nil
	}

	if err := legacymigrate.RunMigrations(conn, dbType); err != nil {
		return fmt.Errorf("running legacy migrations: %w", err)
	}

	return markBaselineApplied(ctx, migrator)
}

// markBaselineApplied records only the baseline migration as applied,
// without running its Up() (which would try to CREATE TABLE against a
// schema a legacy-bridged database already has).
func markBaselineApplied(ctx context.Context, migrator *migrate.Migrator) error {
	for _, m := range migrations.Migrations.Sorted() {
		if m.Name == migrations.BaselineName {
			m.GroupID = 1
			return migrator.MarkApplied(ctx, &m)
		}
	}
	return fmt.Errorf("bridge: baseline migration %q not found among registered migrations", migrations.BaselineName)
}

// tableExists reports whether table exists in the database identified by
// dbType, dispatching per-dialect the same way resolveDriver does elsewhere
// in this package.
func tableExists(ctx context.Context, conn *sql.DB, dbType, table string) (bool, error) {
	var query string
	switch dbType {
	case "sqlite":
		query = `SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = ?`
	case "postgres":
		query = `SELECT 1 FROM information_schema.tables WHERE table_schema = current_schema() AND table_name = $1`
	case "mysql":
		query = `SELECT 1 FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?`
	default:
		return false, fmt.Errorf("unknown db type: %s", dbType)
	}
	var exists int
	err := conn.QueryRowContext(ctx, query, table).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
