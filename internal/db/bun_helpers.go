package db

import (
	"context"
	"database/sql"

	"github.com/uptrace/bun"
)

// execRawProvider is a small interface used to accept either *bun.DB or *bun.Tx
// since both expose NewRaw(...).* methods returning *bun.RawQuery.
type execRawProvider interface {
	NewRaw(query string, args ...interface{}) *bun.RawQuery
}

// ExecRaw executes a raw SQL statement using the provided Bun DB or transaction.
// It returns the standard sql.Result to match existing call sites.
func ExecRaw(ctx context.Context, exec execRawProvider, query string, args ...interface{}) (sql.Result, error) {
	return exec.NewRaw(query, args...).Exec(ctx)
}

// QueryRawInto runs a raw query and scans the result into dest using Bun's RawQuery.Scan.
func QueryRawInto(ctx context.Context, exec execRawProvider, dest interface{}, query string, args ...interface{}) error {
	return exec.NewRaw(query, args...).Scan(ctx, dest)
}
