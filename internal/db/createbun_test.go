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
