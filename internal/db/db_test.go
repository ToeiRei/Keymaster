package db

import (
	"database/sql"
	"errors"
	"testing"

	_ "modernc.org/sqlite"
)

func newTestDB(t *testing.T) string {
	t.Helper()
	dsn := "file:test_" + t.Name() + "?mode=memory&cache=shared"
	if err := InitDB("sqlite", dsn); err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	return dsn
}

func TestInitDB_Migrations_Applied(t *testing.T) {
	dsn := newTestDB(t)

	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("failed to open sql.DB for inspection: %v", err)
	}
	defer func() { _ = sqlDB.Close() }()

	rows, err := sqlDB.Query("PRAGMA table_info(schema_migrations)")
	if err != nil {
		t.Fatalf("failed to query schema_migrations table info: %v", err)
	}
	defer func() { _ = rows.Close() }()

	foundAppliedAt := false
	for rows.Next() {
		var cid int
		var name string
		var typ string
		var notnull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
			t.Fatalf("failed scanning pragma row: %v", err)
		}
		if name == "applied_at" {
			foundAppliedAt = true
			break
		}
	}
	if !foundAppliedAt {
		t.Fatalf("expected schema_migrations.applied_at column to exist after migrations")
	}
}

func TestPublicKey_AddDuplicateBehavior(t *testing.T) {
	_ = newTestDB(t)

	keyData := "AAAAB3NzaC1lZDI1NTE5AAAAItestkeydata"

	// Add via AddPublicKeyAndGetModel
	km := DefaultKeyManager()
	if km == nil {
		t.Fatalf("no key manager available")
	}
	pk, err := km.AddPublicKeyAndGetModel("ed25519", keyData, "dup-comment", false)
	if err != nil {
		t.Fatalf("unexpected error adding public key: %v", err)
	}
	if pk == nil {
		t.Fatalf("expected first AddPublicKeyAndGetModel to return model, got nil")
	}

	// Second call should return (nil, nil) to indicate duplicate
	pk2, err := km.AddPublicKeyAndGetModel("ed25519", keyData, "dup-comment", false)
	if err != nil {
		t.Fatalf("unexpected error on duplicate AddPublicKeyAndGetModel: %v", err)
	}
	if pk2 != nil {
		t.Fatalf("expected duplicate AddPublicKeyAndGetModel to return (nil, nil), got %v", pk2)
	}

	// AddPublicKey should return ErrDuplicate on the second insert
	if err := km.AddPublicKey("ed25519", keyData, "another-comment", false); err != nil {
		t.Fatalf("unexpected error on first AddPublicKey: %v", err)
	}
	if err := km.AddPublicKey("ed25519", keyData, "another-comment", false); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("expected ErrDuplicate on duplicate AddPublicKey, got: %v", err)
	}
}

func TestAccount_AddDuplicateBehavior(t *testing.T) {
	_ = newTestDB(t)

	mgr := DefaultAccountManager()
	if mgr == nil {
		t.Fatalf("no account manager available")
	}
	_, err := mgr.AddAccount("alice", "host1.example", "", "")
	if err != nil {
		t.Fatalf("unexpected error adding account: %v", err)
	}
	_, err = mgr.AddAccount("alice", "host1.example", "", "")
	if !errors.Is(err, ErrDuplicate) {
		t.Fatalf("expected ErrDuplicate on duplicate AddAccount, got: %v", err)
	}
}
