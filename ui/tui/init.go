// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// Package tui contains the terminal UI (TUI) implementation for Keymaster.
package tui

// InitializeDefaults registers TUI-specific defaults. Defer to
// deploy.InitializeDefaults() so core defaults are consistent across UIs.
import "github.com/toeirei/keymaster/core/deploy"

// InitializeDefaults is provided for callers that expect a TUI-level
// initialization entrypoint. It delegates to `core/deploy` to wire the
// canonical default adapters.
func InitializeDefaults() {
	deploy.InitializeDefaults()
}
