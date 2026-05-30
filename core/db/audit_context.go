// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package db

import "sync"

// AuditContext holds optional metadata captured for audit log writes.
type AuditContext struct {
	ClientImplementation string
	Referrer             string
}

var (
	auditContextMu sync.RWMutex
	auditContext   AuditContext
)

// SetAuditContext sets process-level metadata included in future audit writes.
func SetAuditContext(clientImplementation, referrer string) {
	auditContextMu.Lock()
	defer auditContextMu.Unlock()
	auditContext.ClientImplementation = clientImplementation
	auditContext.Referrer = referrer
}

// ClearAuditContext clears process-level metadata used in audit writes.
func ClearAuditContext() {
	SetAuditContext("", "")
}

func getAuditContext() AuditContext {
	auditContextMu.RLock()
	defer auditContextMu.RUnlock()
	return auditContext
}
