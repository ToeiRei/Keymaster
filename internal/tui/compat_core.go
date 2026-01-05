// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package tui

import "github.com/toeirei/keymaster/internal/core"

// containsIgnoreCase kept for backward-compatible tests and callers.
func containsIgnoreCase(s, substr string) bool {
	return core.ContainsIgnoreCase(s, substr)
}

