// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package migrations

import (
	"context"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(dropLegacyTablesUp, dropLegacyTablesDown)
}

// dropLegacyTablesUp removes the tables the bunrewrite models do not use. The
// baseline still creates them because it reproduces the schema as it stood at
// the cutover - a fresh install creates and immediately drops them, a bridged
// legacy install gets the real cleanup here.
//
// system_keys is dropped only after 00000000000003 has copied its private key
// into accounts.deploy_secret; keep that ordering.
//
// schema_migrations is the frozen legacy runner's own version log. bridge.go
// reads it once, to recognise a legacy database and decide whether to run that
// runner; from the next Open() onwards it short-circuits on bun_migrations and
// never looks at it again. By the time this migration runs the bridge is done
// with it for good.
//
// known_hosts is still read and written by core/deploy/ssh.go through the
// global core/db store, which the CLI initializes against the same DSN.
// Dropping it costs that path its SSH host-key verification until host-key
// handling is ported into the bunrewrite stack.
func dropLegacyTablesUp(ctx context.Context, db *bun.DB) error {
	for _, table := range []string{
		"system_keys", "known_hosts", "bootstrap_sessions", "schema_migrations",
	} {
		if _, err := db.NewDropTable().Table(table).IfExists().Exec(ctx); err != nil {
			return err
		}
	}
	return nil
}

// dropLegacyTablesDown deliberately does not recreate the tables: this is dead
// weight being intentionally removed, not a schema change meant to be
// reversible.
func dropLegacyTablesDown(_ context.Context, _ *bun.DB) error {
	return nil
}
