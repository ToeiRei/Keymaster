// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// package db provides the data access layer for Keymaster.
// This file contains a minimal PostgreSQL compatibility wrapper.
//
// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package db // import "github.com/toeirei/keymaster/internal/db"

import (
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib" // PostgreSQL driver
)

// PostgresStore is a compatibility alias to the consolidated BunStore.
// This file preserves the pgx driver blank import so runtime builds that
// expect the driver continue to work while we migrate to a single
// bun-backed store implementation.
type PostgresStore = BunStore

// NewPostgresStore delegates to the canonical db.New initializer and returns
// a PostgresStore typed pointer for compatibility with existing call sites.
func NewPostgresStore(dataSourceName string) (*PostgresStore, error) {
	s, err := New("postgres", dataSourceName)
	if err != nil {
		return nil, err
	}
	bs, ok := s.(*BunStore)
	if !ok {
		return nil, fmt.Errorf("internal error: expected *BunStore, got %T", s)
	}
	return (*PostgresStore)(bs), nil
}
