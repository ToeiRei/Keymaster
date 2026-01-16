package db

import "testing"

func TestInitCompatibility(t *testing.T) {
	dsn1 := "file:TestInitCompat1?mode=memory&cache=shared"
	if _, err := New("sqlite", dsn1); err != nil {
		t.Fatalf("db.New failed: %v", err)
	}
	if !IsInitialized() {
		t.Fatalf("db not initialized after db.New")
	}

	// Basic operation should work after New
	if _, err := CreateSystemKey("pub1", "priv1"); err != nil {
		t.Fatalf("CreateSystemKey failed after db.New: %v", err)
	}

	// Reinitialize via the legacy InitDB path (should behave equivalently)
	dsn2 := "file:TestInitCompat2?mode=memory&cache=shared"
	if err := InitDB("sqlite", dsn2); err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	if !IsInitialized() {
		t.Fatalf("db not initialized after InitDB")
	}

	if _, err := CreateSystemKey("pub2", "priv2"); err != nil {
		t.Fatalf("CreateSystemKey failed after InitDB: %v", err)
	}
}
