// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package db // import "github.com/toeirei/keymaster/internal/db"

import (
	"fmt"
)

// SqliteStore is kept as a compatibility alias to the consolidated BunStore.
type SqliteStore = BunStore

// NewSqliteStore returns a Bun-backed store initialized for sqlite DSNs.
func NewSqliteStore(dataSourceName string) (*SqliteStore, error) {
	s, err := New("sqlite", dataSourceName)
	if err != nil {
		return nil, err
	}
	bs, ok := s.(*BunStore)
	if !ok {
		return nil, fmt.Errorf("internal error: expected *BunStore, got %T", s)
	}
	return bs, nil
}
