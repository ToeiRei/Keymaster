// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package db

import (
	"testing"
)

func TestSystemKey_CreateRotateAndActive(t *testing.T) {
	_ = newTestDB(t)

	// Initially no system keys
	has, err := HasSystemKeys()
	if err != nil {
		t.Fatalf("HasSystemKeys error: %v", err)
	}
	if has {
		t.Fatalf("expected no system keys initially")
	}

	// Create first system key
	s1, err := CreateSystemKey("pub1", "priv1")
	if err != nil {
		t.Fatalf("CreateSystemKey failed: %v", err)
	}
	if s1 <= 0 {
		t.Fatalf("expected positive serial for first system key, got %d", s1)
	}

	// Now HasSystemKeys should be true
	has, err = HasSystemKeys()
	if err != nil {
		t.Fatalf("HasSystemKeys error after create: %v", err)
	}
	if !has {
		t.Fatalf("expected HasSystemKeys true after creating a key")
	}

	// Active key should be the one we just created
	active, err := GetActiveSystemKey()
	if err != nil {
		t.Fatalf("GetActiveSystemKey error: %v", err)
	}
	if active == nil {
		t.Fatalf("expected active system key, got nil")
	}
	if active.Serial != s1 {
		t.Fatalf("expected active serial %d, got %d", s1, active.Serial)
	}

	// Rotate system key
	s2, err := RotateSystemKey("pub2", "priv2")
	if err != nil {
		t.Fatalf("RotateSystemKey failed: %v", err)
	}
	if s2 <= s1 {
		t.Fatalf("expected rotated serial > previous serial: got %d <= %d", s2, s1)
	}

	// After rotation, active key should have serial s2
	active, err = GetActiveSystemKey()
	if err != nil {
		t.Fatalf("GetActiveSystemKey after rotate error: %v", err)
	}
	if active == nil {
		t.Fatalf("expected active system key after rotate, got nil")
	}
	if active.Serial != s2 {
		t.Fatalf("expected active serial %d after rotate, got %d", s2, active.Serial)
	}

	// GetSystemKeyBySerial should find both keys
	k1, err := GetSystemKeyBySerial(s1)
	if err != nil {
		t.Fatalf("GetSystemKeyBySerial s1 error: %v", err)
	}
	if k1 == nil {
		t.Fatalf("expected to find system key for serial %d", s1)
	}
	k2, err := GetSystemKeyBySerial(s2)
	if err != nil {
		t.Fatalf("GetSystemKeyBySerial s2 error: %v", err)
	}
	if k2 == nil {
		t.Fatalf("expected to find system key for serial %d", s2)
	}
}

