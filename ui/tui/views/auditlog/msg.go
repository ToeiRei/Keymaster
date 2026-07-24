// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package auditlog

import "github.com/toeirei/keymaster/client"

// msgPageLoaded carries the result of a single windowed ListAuditLogs fetch.
// gen is compared against the model's current loadGen so stale results (from a
// window the user has since navigated away from) can be dropped.
type msgPageLoaded struct {
	gen     int
	offset  int
	limit   int
	entries []client.AuditLog
	err     error
}
