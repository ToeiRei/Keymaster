// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package bootstrap

import (
	"testing"
	"time"

	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/model"
)

func TestRemoveLine_Basic(t *testing.T) {
	content := "one\ntwo\nthree\n"
	out := removeLine(content, "two")
	if out == content {
		t.Fatalf("expected line removed, got same content: %q", out)
	}
	if out == "one\nthree\n" {
		// ok
	} else {
		t.Fatalf("unexpected result: %q", out)
	}

	// removing non-existent line should be a no-op
	unchanged := removeLine(content, "not-found")
	if unchanged != content {
		t.Fatalf("expected unchanged when line not present, got: %q", unchanged)
	}
}

func TestCleanupAllActiveSessions_ClearsRegistry(t *testing.T) {
	if err := db.InitDB("sqlite", ":memory:"); err != nil {
		t.Fatalf("db.InitDB failed: %v", err)
	}

	s1 := &BootstrapSession{ID: "a", TempKeyPair: &TemporaryKeyPair{privateKey: []byte("a"), publicKey: "p1"}}
	s2 := &BootstrapSession{ID: "b", TempKeyPair: &TemporaryKeyPair{privateKey: []byte("b"), publicKey: "p2"}}
	RegisterSession(s1)
	RegisterSession(s2)

	if err := CleanupAllActiveSessions(); err != nil {
		t.Fatalf("CleanupAllActiveSessions returned error: %v", err)
	}

	sessionsMutex.RLock()
	if len(activeSessions) != 0 {
		sessionsMutex.RUnlock()
		t.Fatalf("expected activeSessions cleared, got %d entries", len(activeSessions))
	}
	sessionsMutex.RUnlock()

	if s1.TempKeyPair != nil && len(s1.TempKeyPair.privateKey) != 0 {
		t.Fatalf("expected s1 private key wiped")
	}
	if s2.TempKeyPair != nil && len(s2.TempKeyPair.privateKey) != 0 {
		t.Fatalf("expected s2 private key wiped")
	}
}

func TestCleanupOrphanedAndExpiredModel(t *testing.T) {
	if err := db.InitDB("sqlite", ":memory:"); err != nil {
		t.Fatalf("db.InitDB failed: %v", err)
	}

	id := "orphan-1"
	// create a DB row to be deleted
	expires := time.Now().Add(1 * time.Minute)
	if err := db.SaveBootstrapSession(id, "u", "h", "lbl", "", "pk", expires, "orphaned"); err != nil {
		t.Fatalf("SaveBootstrapSession failed: %v", err)
	}

	m := &model.BootstrapSession{ID: id, Username: "u", Hostname: "h"}
	if err := cleanupOrphanedSessionModel(m); err != nil {
		t.Fatalf("cleanupOrphanedSessionModel failed: %v", err)
	}

	// ensure delete removed it (Get should return nil)
	got, err := db.GetBootstrapSession(id)
	if err != nil {
		t.Fatalf("GetBootstrapSession failed: %v", err)
	}
	if got != nil {
		t.Fatalf("expected session deleted from DB, still present: %+v", got)
	}

	// expired cleanup - create and delete
	id2 := "exp-1"
	expires2 := time.Now().Add(-1 * time.Hour)
	if err := db.SaveBootstrapSession(id2, "u2", "h2", "lbl2", "", "pk2", expires2, "failed"); err != nil {
		t.Fatalf("SaveBootstrapSession failed: %v", err)
	}
	m2 := &model.BootstrapSession{ID: id2, Username: "u2", Hostname: "h2"}
	if err := cleanupExpiredSessionModel(m2); err != nil {
		t.Fatalf("cleanupExpiredSessionModel failed: %v", err)
	}
	got2, err := db.GetBootstrapSession(id2)
	if err != nil {
		t.Fatalf("GetBootstrapSession failed: %v", err)
	}
	if got2 != nil {
		t.Fatalf("expected expired session deleted from DB, still present: %+v", got2)
	}
}

func TestRecoverFromCrash_RemovesOrphanedSessions(t *testing.T) {
	if err := db.InitDB("sqlite", ":memory:"); err != nil {
		t.Fatalf("db.InitDB failed: %v", err)
	}

	id := "recover-1"
	expires := time.Now().Add(1 * time.Hour)
	if err := db.SaveBootstrapSession(id, "u", "h", "lbl", "", "pk", expires, "orphaned"); err != nil {
		t.Fatalf("SaveBootstrapSession failed: %v", err)
	}

	if err := RecoverFromCrash(); err != nil {
		t.Fatalf("RecoverFromCrash returned error: %v", err)
	}

	got, err := db.GetBootstrapSession(id)
	if err != nil {
		t.Fatalf("GetBootstrapSession failed: %v", err)
	}
	if got != nil {
		t.Fatalf("expected orphaned session removed by RecoverFromCrash, still present: %+v", got)
	}
}

func TestCleanupExpiredSessions_RemovesExpired(t *testing.T) {
	if err := db.InitDB("sqlite", ":memory:"); err != nil {
		t.Fatalf("db.InitDB failed: %v", err)
	}

	id := "cleanup-exp-1"
	expires := time.Now().Add(-2 * time.Hour)
	if err := db.SaveBootstrapSession(id, "u", "h", "lbl", "", "pk", expires, "failed"); err != nil {
		t.Fatalf("SaveBootstrapSession failed: %v", err)
	}

	if err := CleanupExpiredSessions(); err != nil {
		t.Fatalf("CleanupExpiredSessions returned error: %v", err)
	}

	got, err := db.GetBootstrapSession(id)
	if err != nil {
		t.Fatalf("GetBootstrapSession failed: %v", err)
	}
	if got != nil {
		t.Fatalf("expected expired session removed by CleanupExpiredSessions, still present: %+v", got)
	}
}

func TestCleanupSession_UpdatesStatus(t *testing.T) {
	if err := db.InitDB("sqlite", ":memory:"); err != nil {
		t.Fatalf("db.InitDB failed: %v", err)
	}

	s, err := NewBootstrapSession("tmpuser", "example.invalid", "lbl", "")
	if err != nil {
		t.Fatalf("NewBootstrapSession failed: %v", err)
	}

	// Save session to DB so UpdateStatus can succeed
	if err := s.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Run cleanupSession (remote removal will likely fail but is ignored)
	if err := cleanupSession(s); err != nil {
		t.Fatalf("cleanupSession returned error: %v", err)
	}

	got, err := db.GetBootstrapSession(s.ID)
	if err != nil {
		t.Fatalf("GetBootstrapSession failed: %v", err)
	}
	if got == nil {
		t.Fatalf("expected session present in DB after cleanup")
	}
	if got.Status != string(StatusFailed) {
		t.Fatalf("expected status %s, got %s", StatusFailed, got.Status)
	}
}

func TestRemoveTempKeyFromRemoteHost_NoTempKey(t *testing.T) {
	s := &BootstrapSession{ID: "no-key", PendingAccount: model.Account{Username: "u", Hostname: "h"}, TempKeyPair: nil}
	if err := removeTempKeyFromRemoteHost(s); err == nil {
		t.Fatalf("expected error when TempKeyPair is nil")
	}
}

func TestStartSessionReaper_CleansExpired(t *testing.T) {
	if err := db.InitDB("sqlite", ":memory:"); err != nil {
		t.Fatalf("db.InitDB failed: %v", err)
	}

	// create an expired session
	id := "reaper-exp-1"
	expires := time.Now().Add(-1 * time.Hour)
	if err := db.SaveBootstrapSession(id, "u", "h", "lbl", "", "pk", expires, "failed"); err != nil {
		t.Fatalf("SaveBootstrapSession failed: %v", err)
	}

	// run reaper quickly
	oldInterval := sessionReaperInterval
	sessionReaperInterval = 20 * time.Millisecond
	StartSessionReaper()

	// wait for a couple of ticks
	time.Sleep(200 * time.Millisecond)

	// stop the ticker to avoid goroutine leak in tests
	if currentReaperTicker != nil {
		currentReaperTicker.Stop()
		currentReaperTicker = nil
	}
	sessionReaperInterval = oldInterval

	got, err := db.GetBootstrapSession(id)
	if err != nil {
		t.Fatalf("GetBootstrapSession failed: %v", err)
	}
	if got != nil {
		t.Fatalf("expected expired session removed by reaper, still present: %+v", got)
	}
}

