// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/mysqldialect"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/schema"

	// sql drivers
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

// Open resolves the SQL driver for dbType, opens a connection to dsn, applies
// the embedded schema migrations, and returns a bun database with the client's
// models registered.
func Open(dbType, dsn string) (*bun.DB, error) {
	driver, dialect, err := resolveDriver(dbType)
	if err != nil {
		return nil, err
	}

	if driver == "sqlite" && strings.Contains(dsn, ":memory:") {
		// no-op: in-memory databases don't benefit from busy_timeout/WAL and
		// are handled by the pool-pinning branch below.
	} else if driver == "sqlite" && !strings.Contains(dsn, "_pragma=busy_timeout") {
		// Without a busy_timeout, a writer that finds the database locked by
		// another connection fails immediately with "database is locked"
		// instead of waiting, exactly what happens when deploy/verify run
		// several accounts concurrently (see runAccounts). WAL mode also lets
		// readers proceed without blocking on a writer. Both are applied by
		// the modernc.org/sqlite driver per-connection via _pragma params.
		sep := "?"
		if strings.Contains(dsn, "?") {
			sep = "&"
		}
		dsn += sep + "_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	}

	conn, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// SQLite in-memory databases live per-connection; pin the pool to a single
	// connection so the schema created by migrations is visible to every query.
	if driver == "sqlite" && strings.Contains(dsn, ":memory:") {
		conn.SetMaxOpenConns(1)
		conn.SetMaxIdleConns(1)
	}

	bunDB := bun.NewDB(conn, dialect)

	// register the links junction so account<->public_key m2m relations resolve
	bunDB.RegisterModel((*LinkModel)(nil))

	if err := runMigrations(context.Background(), conn, bunDB, dbType); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return bunDB, nil
}

// resolveDriver maps a configured database type to its database/sql driver name
// and matching bun dialect.
func resolveDriver(dbType string) (driver string, dialect schema.Dialect, err error) {
	switch dbType {
	case "sqlite":
		return "sqlite", sqlitedialect.New(), nil
	case "postgres":
		return "pgx", pgdialect.New(), nil
	case "mysql":
		return "mysql", mysqldialect.New(), nil
	default:
		return "", nil, fmt.Errorf("unknown db type: %s", dbType)
	}
}
