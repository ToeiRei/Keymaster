package db

import (
	"github.com/uptrace/bun"
)

// RunMigrationsBun runs the existing SQL migrations using the underlying
// *sql.DB from the provided bun.DB. This delegates to the legacy RunMigrations
// implementation that operates on *sql.DB to avoid duplicating logic.
func RunMigrationsBun(b *bun.DB, dbType string) error {
	if b == nil || b.DB == nil {
		return nil
	}
	return RunMigrations(b.DB, dbType)
}
