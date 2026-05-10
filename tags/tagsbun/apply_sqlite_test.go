// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package tagsbun_test

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
)

func WithSqlite(t *testing.T) *bun.DB {
	sqldb, err := sql.Open(sqliteshim.ShimName, "file::memory:?cache=shared")
	if err != nil {
		panic(err)
	}
	require.NoError(t, err)

	db := bun.NewDB(sqldb, sqlitedialect.New())

	t.Cleanup(func() {
		err := db.Close()
		require.NoError(t, err)
	})

	return db
}

func TestTagsExprToWhereSqlite(t *testing.T) {
	runTests(t, WithSqlite(t))
}
