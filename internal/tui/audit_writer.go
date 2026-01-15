// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package tui

import (
	"github.com/toeirei/keymaster/internal/db"
)

// package-level audit writer override for tui tests
var auditWriter db.AuditWriter

// SetAuditWriter sets a package-level AuditWriter for tui components.
func SetAuditWriter(w db.AuditWriter) {
	auditWriter = w
}

// ClearAuditWriter clears any previously set package-level AuditWriter for tui.
func ClearAuditWriter() {
	auditWriter = nil
}

// logAction writes an audit entry using the package override when present,
// otherwise falls back to the global `ui.DefaultAuditWriter` helper.
func logAction(action, details string) error {
	if auditWriter != nil {
		return auditWriter.LogAction(action, details)
	}
	if w := db.DefaultAuditWriter(); w != nil {
		return w.LogAction(action, details)
	}
	return nil
}
