// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package ui

import (
	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/model"
)

// Convenience wrappers so TUI code can avoid importing `internal/db` directly.
// These are thin adapters that forward to the db package when available.

// GetAllAccounts returns all accounts via the DB package, or nil/error if unavailable.
func GetAllAccounts() ([]model.Account, error) {
	return db.GetAllAccounts()
}

// GetAllActiveAccounts returns active accounts via the DB package.
func GetAllActiveAccounts() ([]model.Account, error) {
	return db.GetAllActiveAccounts()
}

// GetActiveSystemKey returns the active system key from the DB.
func GetActiveSystemKey() (*model.SystemKey, error) {
	return db.GetActiveSystemKey()
}

// CreateSystemKey creates a new system key in the database and returns the assigned serial.
func CreateSystemKey(publicKey, privateKey string) (int, error) {
	return db.CreateSystemKey(publicKey, privateKey)
}

// RotateSystemKey rotates the system key in the database and returns the new serial.
func RotateSystemKey(publicKey, privateKey string) (int, error) {
	return db.RotateSystemKey(publicKey, privateKey)
}

// HasSystemKeys returns whether any system keys exist.
func HasSystemKeys() (bool, error) {
	return db.HasSystemKeys()
}

// ToggleAccountStatus proxies to db.ToggleAccountStatus.
func ToggleAccountStatus(id int) error {
	return db.ToggleAccountStatus(id)
}

// AddKnownHostKey proxies to db.AddKnownHostKey.
func AddKnownHostKey(hostname, key string) error {
	return db.AddKnownHostKey(hostname, key)
}

// UpdateAccountLabel updates the saved label for an account.
func UpdateAccountLabel(id int, label string) error {
	return db.UpdateAccountLabel(id, label)
}

// UpdateAccountTags updates the saved tags for an account.
func UpdateAccountTags(id int, tags string) error {
	return db.UpdateAccountTags(id, tags)
}

// GetAllAuditLogEntries proxies to db.GetAllAuditLogEntries.
func GetAllAuditLogEntries() ([]model.AuditLogEntry, error) {
	return db.GetAllAuditLogEntries()
}

// AuditWriter is a UI-facing abstraction for writing audit entries.
// It mirrors the db.AuditWriter interface to decouple TUI from internal/db.
type AuditWriter interface {
	LogAction(action, details string) error
}

// DefaultAuditWriter returns the package default AuditWriter from the DB layer,
// wrapped as the UI AuditWriter interface. It may return nil.
func DefaultAuditWriter() AuditWriter {
	if w := db.DefaultAuditWriter(); w != nil {
		return w
	}
	return nil
}
