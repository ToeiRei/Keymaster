// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
// Package ui contains the top-level UI wiring and initialization for Keymaster.
package ui

import "github.com/toeirei/keymaster/core/deploy"

// InitializeDefaults registers UI-level defaults. For now, defer to
// deploy.InitializeDefaults() which wires the canonical core adapters used
// by UI packages. This preserves previous behavior after package moves.
func InitializeDefaults() {
	deploy.InitializeDefaults()
}
