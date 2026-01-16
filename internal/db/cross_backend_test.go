package db

import (
	"os"
	"testing"
)

// Cross-backend integration checks. These tests run only when the corresponding
// DSN environment variable is set. They are skipped by default to keep local
// developer test runs fast.
func TestCrossBackend_Postgres(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN not set; skipping Postgres integration test")
	}
	if _, err := New("postgres", dsn); err != nil {
		t.Fatalf("postgres New failed: %v", err)
	}
}

func TestCrossBackend_MySQL(t *testing.T) {
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		t.Skip("MYSQL_DSN not set; skipping MySQL integration test")
	}
	if _, err := New("mysql", dsn); err != nil {
		t.Fatalf("mysql New failed: %v", err)
	}
}
