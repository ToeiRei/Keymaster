// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package migrations

import (
	"context"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(dropAccountKeysUp, dropAccountKeysDown)
}

// dropAccountKeysUp removes the account_keys join table. It was superseded
// by links in the legacy runner's 000005 migration but never actually
// dropped there. This is a no-op on fresh installs (the baseline never
// creates account_keys) and the real cleanup step for bridged legacy
// installs.
func dropAccountKeysUp(ctx context.Context, db *bun.DB) error {
	_, err := db.NewDropTable().Table("account_keys").IfExists().Exec(ctx)
	return err
}

// dropAccountKeysDown deliberately does not recreate account_keys: this is
// dead weight being intentionally removed, not a schema change meant to be
// reversible.
func dropAccountKeysDown(_ context.Context, _ *bun.DB) error {
	return nil
}
