// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package tagsbun_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

func WithPostgres(t *testing.T) *bun.DB {
	dbName := "test"
	dbUser := "test"
	dbPassword := "test"

	postgresC, err := postgres.Run(
		t.Context(),
		"postgres:18-alpine",
		postgres.WithDatabase(dbName),
		postgres.WithUsername(dbUser),
		postgres.WithPassword(dbPassword),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(10*time.Second),
		),
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		err := postgresC.Terminate(context.Background())
		require.NoError(t, err)
	})

	dbUri, err := postgresC.ConnectionString(t.Context(), "sslmode=disable")
	require.NoError(t, err)

	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dbUri)))
	require.NoError(t, err)

	db := bun.NewDB(sqldb, pgdialect.New())

	t.Cleanup(func() {
		err := db.Close()
		require.NoError(t, err)
	})

	return db
}

func TestTagsExprToWherePostgres(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping tests that require testcontainers.")
	}

	runTests(t, WithPostgres(t))
}
