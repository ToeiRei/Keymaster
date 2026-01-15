// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package ui

import "github.com/toeirei/keymaster/internal/db"

// NOTE: Most thin passthrough adapters were removed — callers should import
// `internal/db` directly. This file preserves only the small `AuditWriter`
// abstraction used by UIs and returns the DB-backed default writer when
// available.

// AuditWriter is a UI-facing abstraction for writing audit entries.
type AuditWriter interface {
	LogAction(action, details string) error
}

// DefaultAuditWriter returns the package default AuditWriter from the DB layer.
func DefaultAuditWriter() AuditWriter {
	if w := db.DefaultAuditWriter(); w != nil {
		return w
	}
	return nil
}
