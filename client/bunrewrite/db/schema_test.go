// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package db

import (
	"context"
	"database/sql"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/uptrace/bun/schema"
)

// modelTables lists every table the client's models read or write. Anything
// else the migrations leave behind is bookkeeping, checked separately below.
var modelTables = []any{
	(*AccountModel)(nil),
	(*PublicKeyModel)(nil),
	(*LinkModel)(nil),
	(*AuditLogModel)(nil),
}

// bookkeepingPrefixes match the tables that legitimately exist alongside the
// model tables: the bun migrator's own (bun_migrations, bun_migration_locks),
// sqlite internals (sqlite_sequence, sqlite_autoindex_*), and - on a bridged
// install only - the legacy runner's version log, which bridge.go still uses to
// recognise a legacy-origin database.
var bookkeepingPrefixes = []string{"bun_", "sqlite_", "schema_migrations"}

// TestSchema_MatchesModels asserts the schema the migrations actually produce
// is exactly what the models expect - same tables, same columns, nothing left
// over - for both a fresh install and a bridged legacy one. The bridged path is
// the interesting half: it reaches the same shape through the legacy runner's
// in-place ALTERs rather than the baseline's CREATE TABLEs.
func TestSchema_MatchesModels(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		name := "fresh"
		if legacy {
			name = "bridged legacy"
		}
		t.Run(name, func(t *testing.T) {
			conn := openMemSQLite(t)
			if legacy {
				applyLegacyMigrations(t, conn, "000005") // 000001-000004, pre-reshape
			}
			bunDB := newBunDB(conn)
			if err := runMigrations(context.Background(), conn, bunDB, "sqlite"); err != nil {
				t.Fatalf("runMigrations: %v", err)
			}

			var expected []string
			for _, model := range modelTables {
				table := bunDB.Dialect().Tables().Get(reflect.TypeOf(model).Elem())
				expected = append(expected, table.Name)
				assertColumnsMatch(t, conn, table)
			}

			for _, table := range tableNames(t, conn) {
				if slices.Contains(expected, table) || isBookkeeping(table) {
					continue
				}
				t.Errorf("table %q survives the migrations but no model uses it", table)
			}
		})
	}
}

func isBookkeeping(table string) bool {
	return slices.ContainsFunc(bookkeepingPrefixes, func(prefix string) bool {
		return strings.HasPrefix(table, prefix)
	})
}

// assertColumnsMatch compares a model's bun fields against the columns the
// database actually has, in both directions.
func assertColumnsMatch(t *testing.T, conn *sql.DB, table *schema.Table) {
	t.Helper()

	var want []string
	for _, field := range table.Fields {
		want = append(want, field.Name)
	}
	slices.Sort(want)

	got := columnNames(t, conn, table.Name)
	slices.Sort(got)

	if !slices.Equal(want, got) {
		t.Errorf("table %s columns = %v, model declares %v", table.Name, got, want)
	}
}

func tableNames(t *testing.T, conn *sql.DB) []string {
	t.Helper()
	rows, err := conn.Query(`SELECT name FROM sqlite_master WHERE type = 'table' ORDER BY name`)
	if err != nil {
		t.Fatalf("list tables: %v", err)
	}
	defer rows.Close()
	return scanStrings(t, rows)
}

func columnNames(t *testing.T, conn *sql.DB, table string) []string {
	t.Helper()
	rows, err := conn.Query(`SELECT name FROM pragma_table_info(?)`, table)
	if err != nil {
		t.Fatalf("table_info(%s): %v", table, err)
	}
	defer rows.Close()
	return scanStrings(t, rows)
}

func scanStrings(t *testing.T, rows *sql.Rows) []string {
	t.Helper()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatalf("scan: %v", err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	return out
}
