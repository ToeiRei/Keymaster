package tui

import "github.com/toeirei/keymaster/internal/core"

// containsIgnoreCase kept for backward-compatible tests and callers.
func containsIgnoreCase(s, substr string) bool {
	return core.ContainsIgnoreCase(s, substr)
}
