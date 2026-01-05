// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package db

import (
	"testing"
)

// WithTestStore initializes an in-memory sqlite Store for the duration of the
// provided function and restores package-level globals afterwards.
func WithTestStore(t *testing.T, fn func(s *SqliteStore)) {
	t.Helper()

	// Save previous globals
	prevStore := store
	prevDefaultSearcher := defaultSearcher
	prevDefaultAuditSearcher := defaultAuditSearcher
	prevDefaultKeySearcher := defaultKeySearcher
	prevDefaultAccountManager := defaultAccountManager
	prevDefaultKeyManager := defaultKeyManager
	prevDefaultAuditWriter := defaultAuditWriter

	// Initialize in-memory sqlite DB for this test
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	if err := InitDB("sqlite", dsn); err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	s, ok := store.(*SqliteStore)
	if !ok {
		t.Fatalf("store is not *SqliteStore")
	}

	// Ensure restoration of globals after fn completes
	defer func() {
		store = prevStore
		defaultSearcher = prevDefaultSearcher
		defaultAuditSearcher = prevDefaultAuditSearcher
		defaultKeySearcher = prevDefaultKeySearcher
		defaultAccountManager = prevDefaultAccountManager
		defaultKeyManager = prevDefaultKeyManager
		defaultAuditWriter = prevDefaultAuditWriter
	}()

	fn(s)
}

// WithAuditWriter temporarily sets the package-level AuditWriter for the
// duration of fn and restores the previous writer afterwards.
func WithAuditWriter(t *testing.T, w AuditWriter, fn func()) {
	t.Helper()
	prev := defaultAuditWriter
	defaultAuditWriter = w
	defer func() { defaultAuditWriter = prev }()
	fn()
}

