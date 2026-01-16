// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package db // import "github.com/toeirei/keymaster/internal/db"

import (
	"fmt"

	_ "github.com/go-sql-driver/mysql" // MySQL driver (kept for Bun/runtime)
)

// MySQLStore is a compatibility alias to the consolidated BunStore.
type MySQLStore = BunStore

// NewMySQLStore initializes a Bun-backed store for MySQL DSNs and returns
// it typed as *MySQLStore for compatibility with existing call sites.
func NewMySQLStore(dataSourceName string) (*MySQLStore, error) {
	s, err := New("mysql", dataSourceName)
	if err != nil {
		return nil, err
	}
	bs, ok := s.(*BunStore)
	if !ok {
		return nil, fmt.Errorf("internal error: expected *BunStore, got %T", s)
	}
	return (*MySQLStore)(bs), nil
}
