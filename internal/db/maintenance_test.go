package db

import (
	"testing"
	"time"
)

func TestRunDBMaintenance_Sqlite(t *testing.T) {
	// Set up an in-memory DB and create a simple table.
	dsn := "file:TestRunDBMaintenance?mode=memory&cache=shared"
	if err := InitDB("sqlite", dsn); err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	// Run maintenance — should complete without error.
	if err := RunDBMaintenance("sqlite", dsn); err != nil {
		t.Fatalf("RunDBMaintenance(sqlite) failed: %v", err)
	}
	// Make sure we can still use the DB after maintenance.
	if _, err := GetAllActiveAccounts(); err != nil {
		t.Fatalf("GetAllActiveAccounts after maintenance failed: %v", err)
	}
	// Quick sleep to ensure any background tidying completes in CI-low resource VMs.
	time.Sleep(10 * time.Millisecond)
}
