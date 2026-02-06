// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
//
// Package uiadapters contains thin adapter implementations that bridge
// `core` interfaces to concrete package-level helpers (e.g., `core/db`).
// Adapters are intentionally small and deterministic so UIs can depend on
// stable interfaces while implementation resides in dedicated adapter code.
package uiadapters
