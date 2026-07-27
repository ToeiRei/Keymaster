// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package migrations

import (
	"context"

	"github.com/uptrace/bun"
)

// connectorColumns maps each accounts column from its deploy_* name to the
// connector terminology the client now speaks, matching the connector package
// that owns their shape.
var connectorColumns = [][2]string{
	{"deploy_method", "connector"},
	{"deploy_secret", "connector_secret"},
	{"deploy_secret_rollback", "connector_secret_rollback"},
	{"deploy_cache", "connector_cache"},
}

func init() {
	Migrations.MustRegister(renameConnectorColumnsUp, renameConnectorColumnsDown)
}

func renameConnectorColumnsUp(ctx context.Context, db *bun.DB) error {
	return renameAccountColumns(ctx, db, false)
}

// renameConnectorColumnsDown restores the deploy_* names. Unlike the drops in
// 00000000000002 and 00000000000004, a pure rename loses nothing, so it is
// worth reversing properly.
func renameConnectorColumnsDown(ctx context.Context, db *bun.DB) error {
	return renameAccountColumns(ctx, db, true)
}

// renameAccountColumns renames every pair in connectorColumns, swapped when
// reverse is set. bun has no rename-column builder, but bare ALTER TABLE ...
// RENAME COLUMN is what the frozen legacy runner already uses on all three
// dialects, and bun.Ident lets each one quote the names itself.
func renameAccountColumns(ctx context.Context, db *bun.DB, reverse bool) error {
	for _, columns := range connectorColumns {
		from, to := columns[0], columns[1]
		if reverse {
			from, to = to, from
		}
		if _, err := db.ExecContext(ctx, "ALTER TABLE accounts RENAME COLUMN ? TO ?",
			bun.Ident(from), bun.Ident(to)); err != nil {
			return err
		}
	}
	return nil
}
