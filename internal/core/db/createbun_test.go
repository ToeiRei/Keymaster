// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package db

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestCreateBunDB_VariousDialects(t *testing.T) {
	sqlDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open sqlite in-memory: %v", err)
	}
	defer func() { _ = sqlDB.Close() }()

	cases := []string{"sqlite", "postgres", "mysql", "unknown"}
	for _, c := range cases {
		b := createBunDB(sqlDB, c)
		if b == nil {
			t.Fatalf("createBunDB returned nil for dialect %s", c)
		}
	}
}
