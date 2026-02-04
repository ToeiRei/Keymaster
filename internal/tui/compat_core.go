//go:build ignore

// This file is intentionally excluded from normal builds. It used to provide a
// backward-compatible wrapper `containsIgnoreCase` for tests; tests now call
// `core.ContainsIgnoreCase` directly and this file is kept only as a
// non-building artifact to avoid deleting history during phased migration.

package tui

// intentionally empty
