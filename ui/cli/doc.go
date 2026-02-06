// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
//
// Package cli implements the command-line interface for Keymaster using Cobra.
// It wires configuration, default services, and provides commands that delegate
// to deterministic `core` facades. CLI code should remain thin and delegate
// business logic to `core` and adapter packages.
package cli
