// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package tui

import (
	"testing"

	"github.com/toeirei/keymaster/internal/db"
)

// initTestDBT initializes an in-memory sqlite DB for tests and registers cleanup.
func initTestDBT(t *testing.T) {
	t.Helper()
	if err := db.InitDB("sqlite", ":memory:"); err != nil {
		t.Fatalf("initTestDBT: db.InitDB failed: %v", err)
	}
	t.Cleanup(func() {
		// Reset the package-level DB state if needed. The db package does not
		// currently expose a Close function; if it does, call it here. For now,
		// rely on process exit and the in-memory database scoping.
	})
}

