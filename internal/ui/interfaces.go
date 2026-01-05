// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package ui

import "github.com/toeirei/keymaster/internal/model"

// AccountSearcher provides a small abstraction for searching accounts.
type AccountSearcher interface {
	SearchAccounts(q string) ([]model.Account, error)
}

// KeySearcher provides a small abstraction for searching public keys.
type KeySearcher interface {
	SearchPublicKeys(q string) ([]model.PublicKey, error)
}

// AuditSearcher provides a small abstraction for retrieving audit log entries.
type AuditSearcher interface {
	GetAllAuditLogEntries() ([]model.AuditLogEntry, error)
}
