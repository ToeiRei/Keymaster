// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/toeirei/keymaster/internal/model"
)

// setupTestDB creates a temporary SQLite database for testing
func setupTestDB(t *testing.T) (*SqliteStore, func()) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := sql.Open("sqlite", dbPath+"?_busy_timeout=5000")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	store := &SqliteStore{db: db}

	// Run migrations manually for test
	if err := RunMigrations(db, "sqlite"); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.RemoveAll(tempDir)
	}

	return store, cleanup
}

// TestRecordDriftEvent tests recording a drift event
func TestRecordDriftEvent(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	// Create a test account first
	accountID, err := store.AddAccount("testuser", "testhost", "test-label", "test-tag")
	if err != nil {
		t.Fatalf("Failed to create test account: %v", err)
	}

	// Record drift event
	err = store.RecordDriftEvent(accountID, string(model.DriftCritical), "test drift details")
	if err != nil {
		t.Fatalf("Failed to record drift event: %v", err)
	}

	// Verify the drift event was recorded
	events, err := store.GetDriftEventsForAccount(accountID, 10)
	if err != nil {
		t.Fatalf("Failed to get drift events: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("Expected 1 drift event, got %d", len(events))
	}

	event := events[0]
	if event.AccountID != accountID {
		t.Errorf("Expected account ID %d, got %d", accountID, event.AccountID)
	}
	if event.DriftType != model.DriftCritical {
		t.Errorf("Expected drift type %s, got %s", model.DriftCritical, event.DriftType)
	}
	if event.Details != "test drift details" {
		t.Errorf("Expected details 'test drift details', got '%s'", event.Details)
	}
	if event.WasRemediated {
		t.Errorf("Expected WasRemediated to be false, got true")
	}
}

// TestMarkDriftRemediated tests marking a drift event as remediated
func TestMarkDriftRemediated(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	// Create account and drift event
	accountID, _ := store.AddAccount("testuser", "testhost", "test-label", "test-tag")
	err := store.RecordDriftEvent(accountID, string(model.DriftWarning), "test drift")
	if err != nil {
		t.Fatalf("Failed to record drift event: %v", err)
	}

	// Get the event ID
	events, _ := store.GetDriftEventsForAccount(accountID, 10)
	eventID := events[0].ID

	// Mark as remediated
	err = store.MarkDriftRemediated(eventID)
	if err != nil {
		t.Fatalf("Failed to mark drift as remediated: %v", err)
	}

	// Verify it was marked
	events, _ = store.GetDriftEventsForAccount(accountID, 10)
	if !events[0].WasRemediated {
		t.Errorf("Expected WasRemediated to be true")
	}
	if events[0].RemediatedAt == nil {
		t.Errorf("Expected RemediatedAt to be set")
	}
}

// TestGetDriftEventsForAccount tests retrieving drift events for an account
func TestGetDriftEventsForAccount(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	// Create test account
	accountID, _ := store.AddAccount("testuser", "testhost", "test-label", "test-tag")

	// Record multiple drift events
	driftTypes := []model.DriftClassification{
		model.DriftCritical,
		model.DriftWarning,
		model.DriftInfo,
		model.DriftCritical,
		model.DriftWarning,
	}

	for i, driftType := range driftTypes {
		err := store.RecordDriftEvent(accountID, string(driftType), "drift "+string(rune(i)))
		if err != nil {
			t.Fatalf("Failed to record drift event %d: %v", i, err)
		}
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	// Test retrieving all events
	events, err := store.GetDriftEventsForAccount(accountID, 100)
	if err != nil {
		t.Fatalf("Failed to get drift events: %v", err)
	}

	if len(events) != len(driftTypes) {
		t.Errorf("Expected %d events, got %d", len(driftTypes), len(events))
	}

	// Test limit
	events, err = store.GetDriftEventsForAccount(accountID, 3)
	if err != nil {
		t.Fatalf("Failed to get drift events with limit: %v", err)
	}

	if len(events) != 3 {
		t.Errorf("Expected 3 events with limit, got %d", len(events))
	}

	// Events should be ordered by detected_at DESC (newest first)
	if len(events) >= 2 {
		if events[0].DetectedAt.Before(events[1].DetectedAt) {
			t.Errorf("Events not ordered by detected_at DESC")
		}
	}
}

// TestGetDriftStatistics tests retrieving drift statistics
func TestGetDriftStatistics(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	// Create accounts and drift events
	account1, _ := store.AddAccount("user1", "host1", "label1", "tag1")
	account2, _ := store.AddAccount("user2", "host2", "label2", "tag2")

	// Record drift events
	store.RecordDriftEvent(account1, string(model.DriftCritical), "drift 1")
	store.RecordDriftEvent(account1, string(model.DriftWarning), "drift 2")
	store.RecordDriftEvent(account2, string(model.DriftInfo), "drift 3")

	// Get statistics
	totalDrifts, remediatedDrifts, err := store.GetDriftStatistics()
	if err != nil {
		t.Fatalf("Failed to get drift statistics: %v", err)
	}

	if totalDrifts != 3 {
		t.Errorf("Expected 3 total drifts, got %d", totalDrifts)
	}

	if remediatedDrifts != 0 {
		t.Errorf("Expected 0 remediated drifts, got %d", remediatedDrifts)
	}

	// Mark one as remediated
	events, _ := store.GetDriftEventsForAccount(account1, 10)
	store.MarkDriftRemediated(events[0].ID)

	// Check statistics again
	totalDrifts, remediatedDrifts, err = store.GetDriftStatistics()
	if err != nil {
		t.Fatalf("Failed to get drift statistics after remediation: %v", err)
	}

	if totalDrifts != 3 {
		t.Errorf("Expected 3 total drifts, got %d", totalDrifts)
	}

	if remediatedDrifts != 1 {
		t.Errorf("Expected 1 remediated drift, got %d", remediatedDrifts)
	}
}

// TestGetHostsWithFrequentDrift tests retrieving hosts with frequent drift
func TestGetHostsWithFrequentDrift(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	// Create accounts
	account1, _ := store.AddAccount("user1", "host1", "label1", "tag1")
	account2, _ := store.AddAccount("user2", "host2", "label2", "tag2")
	account3, _ := store.AddAccount("user3", "host3", "label3", "tag3")

	// Record drift events with different frequencies
	// account1: 5 drifts
	for i := 0; i < 5; i++ {
		store.RecordDriftEvent(account1, string(model.DriftCritical), "drift")
		time.Sleep(5 * time.Millisecond)
	}

	// account2: 2 drifts
	for i := 0; i < 2; i++ {
		store.RecordDriftEvent(account2, string(model.DriftWarning), "drift")
		time.Sleep(5 * time.Millisecond)
	}

	// account3: 1 drift
	store.RecordDriftEvent(account3, string(model.DriftInfo), "drift")

	// Get hosts with frequent drift
	stats, err := store.GetHostsWithFrequentDrift(10)
	if err != nil {
		t.Fatalf("Failed to get hosts with frequent drift: %v", err)
	}

	if len(stats) != 3 {
		t.Errorf("Expected 3 hosts with drift, got %d", len(stats))
	}

	// Should be ordered by drift count DESC
	if stats[0].DriftCount != 5 {
		t.Errorf("Expected first host to have 5 drifts, got %d", stats[0].DriftCount)
	}
	if stats[1].DriftCount != 2 {
		t.Errorf("Expected second host to have 2 drifts, got %d", stats[1].DriftCount)
	}
	if stats[2].DriftCount != 1 {
		t.Errorf("Expected third host to have 1 drift, got %d", stats[2].DriftCount)
	}

	// Test limit
	statsLimited, err := store.GetHostsWithFrequentDrift(2)
	if err != nil {
		t.Fatalf("Failed to get hosts with frequent drift (limited): %v", err)
	}

	if len(statsLimited) != 2 {
		t.Errorf("Expected 2 hosts with limit, got %d", len(statsLimited))
	}
}

// TestGetRecentDriftEvents tests retrieving recent drift events
func TestGetRecentDriftEvents(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	// Create accounts
	account1, _ := store.AddAccount("user1", "host1", "label1", "tag1")
	account2, _ := store.AddAccount("user2", "host2", "label2", "tag2")

	// Record drift events
	for i := 0; i < 5; i++ {
		accountID := account1
		if i%2 == 0 {
			accountID = account2
		}
		store.RecordDriftEvent(accountID, string(model.DriftCritical), "drift")
		time.Sleep(10 * time.Millisecond)
	}

	// Get recent drift events
	events, err := store.GetRecentDriftEvents(10)
	if err != nil {
		t.Fatalf("Failed to get recent drift events: %v", err)
	}

	if len(events) != 5 {
		t.Errorf("Expected 5 recent drift events, got %d", len(events))
	}

	// Should be ordered by detected_at DESC
	for i := 0; i < len(events)-1; i++ {
		if events[i].DetectedAt.Before(events[i+1].DetectedAt) {
			t.Errorf("Events not ordered by detected_at DESC")
			break
		}
	}

	// Test limit
	eventsLimited, err := store.GetRecentDriftEvents(3)
	if err != nil {
		t.Fatalf("Failed to get recent drift events (limited): %v", err)
	}

	if len(eventsLimited) != 3 {
		t.Errorf("Expected 3 events with limit, got %d", len(eventsLimited))
	}
}

// TestDriftEventCascadeDelete tests that drift events are deleted when account is deleted
func TestDriftEventCascadeDelete(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	// Create account and drift events
	accountID, _ := store.AddAccount("testuser", "testhost", "test-label", "test-tag")

	for i := 0; i < 3; i++ {
		store.RecordDriftEvent(accountID, string(model.DriftCritical), "drift")
	}

	// Verify events exist
	events, _ := store.GetDriftEventsForAccount(accountID, 10)
	if len(events) != 3 {
		t.Fatalf("Expected 3 drift events before deletion, got %d", len(events))
	}

	// Delete the account
	err := store.DeleteAccount(accountID)
	if err != nil {
		t.Fatalf("Failed to delete account: %v", err)
	}

	// Verify drift events are also deleted (CASCADE)
	events, _ = store.GetDriftEventsForAccount(accountID, 10)
	if len(events) != 0 {
		t.Errorf("Expected 0 drift events after account deletion, got %d", len(events))
	}
}

// Benchmark tests
func BenchmarkRecordDriftEvent(b *testing.B) {
	tempDir := b.TempDir()
	dbPath := filepath.Join(tempDir, "bench.db")
	db, _ := sql.Open("sqlite", dbPath+"?_busy_timeout=5000")
	defer db.Close()

	db.Exec("PRAGMA foreign_keys = ON;")
	store := &SqliteStore{db: db}
	RunMigrations(db, "sqlite")

	accountID, _ := store.AddAccount("benchuser", "benchhost", "bench", "bench")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.RecordDriftEvent(accountID, string(model.DriftCritical), "benchmark drift")
	}
}

func BenchmarkGetDriftStatistics(b *testing.B) {
	tempDir := b.TempDir()
	dbPath := filepath.Join(tempDir, "bench.db")
	db, _ := sql.Open("sqlite", dbPath+"?_busy_timeout=5000")
	defer db.Close()

	db.Exec("PRAGMA foreign_keys = ON;")
	store := &SqliteStore{db: db}
	RunMigrations(db, "sqlite")

	// Create test data
	accountID, _ := store.AddAccount("benchuser", "benchhost", "bench", "bench")
	for i := 0; i < 100; i++ {
		store.RecordDriftEvent(accountID, string(model.DriftCritical), "drift")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.GetDriftStatistics()
	}
}
