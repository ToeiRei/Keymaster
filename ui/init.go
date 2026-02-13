// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// Package ui contains the top-level UI wiring and initialization for Keymaster.
package ui

// InitializeDefaults registers UI-level defaults. For now, defer to
// deploy.InitializeDefaults() which wires the canonical core adapters used
// by UI packages. This preserves previous behavior after package moves.
import "github.com/toeirei/keymaster/core/deploy"

// InitializeDefaults is provided for callers that expect a UI-level
// initialization entrypoint. It delegates to `core/deploy` to wire the
// canonical default adapters.
func InitializeDefaults() {
	deploy.InitializeDefaults()
}
