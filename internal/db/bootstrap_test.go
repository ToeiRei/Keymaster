// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package db

import (
	"testing"
	"time"
)

func TestBootstrapSession_LifecycleAndQueries(t *testing.T) {
	_ = newTestDB(t)

	now := time.Now()

	// Create an expired session
	expiredID := "bs-expired"
	expiresExpired := now.Add(-1 * time.Hour)
	if err := SaveBootstrapSession(expiredID, "alice", "host-exp", "", "", "tmp1", expiresExpired, "active"); err != nil {
		t.Fatalf("SaveBootstrapSession expired failed: %v", err)
	}

	// Create an orphaned session (not expired but status orphaned)
	orphanID := "bs-orphan"
	expiresOrphan := now.Add(2 * time.Hour)
	if err := SaveBootstrapSession(orphanID, "bob", "host-orph", "", "", "tmp2", expiresOrphan, "orphaned"); err != nil {
		t.Fatalf("SaveBootstrapSession orphan failed: %v", err)
	}

	// Create an active session
	activeID := "bs-active"
	expiresActive := now.Add(2 * time.Hour)
	if err := SaveBootstrapSession(activeID, "carol", "host-act", "", "", "tmp3", expiresActive, "active"); err != nil {
		t.Fatalf("SaveBootstrapSession active failed: %v", err)
	}

	// Fetch individual session
	bs, err := GetBootstrapSession(expiredID)
	if err != nil {
		t.Fatalf("GetBootstrapSession failed: %v", err)
	}
	if bs == nil || bs.ID != expiredID {
		t.Fatalf("expected to get expired session by id, got: %+v", bs)
	}

	// Expired sessions should include expiredID
	expiredList, err := GetExpiredBootstrapSessions()
	if err != nil {
		t.Fatalf("GetExpiredBootstrapSessions failed: %v", err)
	}
	found := false
	for _, s := range expiredList {
		if s.ID == expiredID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected expired session %s in expired list", expiredID)
	}

	// Orphaned sessions should include orphanID
	orphans, err := GetOrphanedBootstrapSessions()
	if err != nil {
		t.Fatalf("GetOrphanedBootstrapSessions failed: %v", err)
	}
	found = false
	for _, s := range orphans {
		if s.ID == orphanID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected orphan session %s in orphan list", orphanID)
	}

	// Update active session to orphaned and verify it appears in orphan list
	if err := UpdateBootstrapSessionStatus(activeID, "orphaned"); err != nil {
		t.Fatalf("UpdateBootstrapSessionStatus failed: %v", err)
	}
	orphans, err = GetOrphanedBootstrapSessions()
	if err != nil {
		t.Fatalf("GetOrphanedBootstrapSessions failed after update: %v", err)
	}
	found = false
	for _, s := range orphans {
		if s.ID == activeID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected updated session %s in orphan list", activeID)
	}

	// Delete orphan session and ensure it's gone
	if err := DeleteBootstrapSession(orphanID); err != nil {
		t.Fatalf("DeleteBootstrapSession failed: %v", err)
	}
	bs, err = GetBootstrapSession(orphanID)
	if err != nil {
		t.Fatalf("GetBootstrapSession after delete failed: %v", err)
	}
	if bs != nil {
		t.Fatalf("expected deleted session to be nil, got: %+v", bs)
	}
}
