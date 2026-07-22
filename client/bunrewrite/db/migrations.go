// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package db

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
	"time"
)

//go:embed migrations
var embeddedMigrations embed.FS

// RunMigrations applies the embedded schema migrations for the given database
// connection. Applied versions are tracked in a schema_migrations bookkeeping
// table so migrations run at most once. Each migration file is applied inside a
// transaction. This is a small, self-contained runner adapted from core/db.
func RunMigrations(db *sql.DB, dbType string) error {
	migrationsPath := fmt.Sprintf("migrations/%s", dbType)

	entries, err := fs.ReadDir(embeddedMigrations, migrationsPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// No migrations embedded for this DB type.
			return nil
		}
		return fmt.Errorf("failed to read embedded migrations (%s): %w", migrationsPath, err)
	}

	// Collect .up.sql files and sort them for deterministic ordering.
	var ups []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(e.Name(), ".up.sql") {
			ups = append(ups, e.Name())
		}
	}
	sort.Strings(ups)

	if err := ensureSchemaMigrationsTable(db, dbType); err != nil {
		return fmt.Errorf("failed to ensure schema_migrations table: %w", err)
	}

	for _, fname := range ups {
		version := strings.TrimSuffix(fname, ".up.sql")

		// Skip already-applied migrations.
		query := "SELECT 1 FROM schema_migrations WHERE version = ?"
		if dbType == "postgres" {
			query = "SELECT 1 FROM schema_migrations WHERE version = $1"
		}
		var exists int
		err := db.QueryRow(query, version).Scan(&exists)
		if err == nil {
			continue
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("failed to check migration version %s: %w", version, err)
		}

		data, err := embeddedMigrations.ReadFile(path.Join(migrationsPath, fname))
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", fname, err)
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction for migration %s: %w", version, err)
		}
		if _, err := tx.Exec(string(data)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("failed to execute migration %s: %w", version, err)
		}

		insertQuery := "INSERT INTO schema_migrations(version, applied_at) VALUES(?, ?)"
		if dbType == "postgres" {
			insertQuery = "INSERT INTO schema_migrations(version, applied_at) VALUES($1, $2)"
		}
		if _, err := tx.Exec(insertQuery, version, time.Now()); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("failed to record migration %s: %w", version, err)
		}
		if err := tx.Commit(); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("failed to commit migration %s: %w", version, err)
		}
	}

	return nil
}

// ensureSchemaMigrationsTable creates the schema_migrations bookkeeping table if
// it does not already exist.
func ensureSchemaMigrationsTable(db *sql.DB, dbType string) error {
	if dbType == "mysql" {
		// MySQL cannot index a TEXT/BLOB primary key without a length.
		_, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (version VARCHAR(191) PRIMARY KEY, applied_at TIMESTAMP NULL)`)
		return err
	}
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY, applied_at TIMESTAMP)`)
	return err
}
