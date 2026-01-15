// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package main

import "github.com/toeirei/keymaster/internal/db"

// logAction writes an audit entry using a default AuditWriter when available.
// This avoids calling db.LogAction directly from command code.
func logAction(action, details string) error {
	if w := db.DefaultAuditWriter(); w != nil {
		return w.LogAction(action, details)
	}
	return nil
}
