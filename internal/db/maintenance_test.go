package db

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func TestRunDBMaintenance_SqliteSuccess(t *testing.T) {
	// in-memory sqlite should succeed
	if err := RunDBMaintenance("sqlite", ":memory:"); err != nil {
		t.Fatalf("expected sqlite maintenance to succeed, got: %v", err)
	}
}

func TestRunDBMaintenance_UnknownDriver(t *testing.T) {
	// an unknown driver name should cause an error (sql.Open fails)
	if err := RunDBMaintenance("no-such-driver", "dsn"); err == nil {
		t.Fatalf("expected error for unknown driver")
	}
}

func TestCreateBunDB_Dialects(t *testing.T) {
	sqlDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer func() { _ = sqlDB.Close() }()

	cases := map[string]string{
		"sqlite":   "sqlitedialect",
		"postgres": "pgdialect",
		"mysql":    "mysqldialect",
		"unknown":  "sqlitedialect",
	}

	for typ, want := range cases {
		b := createBunDB(sqlDB, typ)
		if b == nil {
			t.Fatalf("createBunDB returned nil for %s", typ)
		}
		got := fmt.Sprintf("%T", b.Dialect())
		if !strings.Contains(got, want) {
			t.Fatalf("for type %s expected dialect containing %s, got %s", typ, want, got)
		}
	}

}
