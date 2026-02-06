// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
// Package core contains deterministic, UI-agnostic business logic and small
// interface definitions used across the application. The core package is
// designed to be free of direct UI or DB dependencies — adapters and
// implementations are injected via small interfaces defined in
// `core/interfaces.go` and via SetDefault* helpers for test and runtime
// wiring.
package core
