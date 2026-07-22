// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package db

import (
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

	if err := RunMigrations(conn, dbType); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	bunDB := bun.NewDB(conn, dialect)

	// register the links junction so account<->public_key m2m relations resolve
	bunDB.RegisterModel((*LinkModel)(nil))

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
