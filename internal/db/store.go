// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package db

import (
	"time"

	"github.com/toeirei/keymaster/internal/model"
	"github.com/uptrace/bun"
)

// Store defines the interface for all database operations in Keymaster.
// This allows for multiple database backends to be implemented.
type Store interface {
	// Account methods
	GetAllAccounts() ([]model.Account, error)
	AddAccount(username, hostname, label, tags string) (int, error)
	DeleteAccount(id int) error
	UpdateAccountSerial(id, serial int) error
	ToggleAccountStatus(id int) error
	UpdateAccountLabel(id int, label string) error
	UpdateAccountHostname(id int, hostname string) error
	UpdateAccountTags(id int, tags string) error
	GetAllActiveAccounts() ([]model.Account, error)
	// UpdateAccountIsDirty sets or clears the is_dirty flag for an account.
	UpdateAccountIsDirty(id int, dirty bool) error

	// Public Key methods
	// Public Key methods have been moved to the KeyManager abstraction. Store
	// implementations continue to provide Bun helpers in `bun_adapter.go`.

	// Host Key methods
	GetKnownHostKey(hostname string) (string, error)
	AddKnownHostKey(hostname, key string) error

	// System Key methods
	CreateSystemKey(publicKey, privateKey string) (int, error)
	RotateSystemKey(publicKey, privateKey string) (int, error)
	GetActiveSystemKey() (*model.SystemKey, error)
	GetSystemKeyBySerial(serial int) (*model.SystemKey, error)
	HasSystemKeys() (bool, error)

	// Assignment methods
	// NOTE: key<->account assignment helpers have been moved behind the
	// `KeyManager` abstraction. Store implementations should continue to
	// provide low-level Bun helpers in `bun_adapter.go` (used by the
	// KeyManager) but no longer need to expose assignment methods here.
	// SearchAccounts performs a fuzzy search for accounts matching the query.
	// Implementations should provide sensible, portable search semantics.
	SearchAccounts(query string) ([]model.Account, error)

	// Audit Log methods
	GetAllAuditLogEntries() ([]model.AuditLogEntry, error)
	LogAction(action string, details string) error

	// Bootstrap Session methods
	SaveBootstrapSession(id, username, hostname, label, tags, tempPublicKey string, expiresAt time.Time, status string) error
	GetBootstrapSession(id string) (*model.BootstrapSession, error)
	DeleteBootstrapSession(id string) error
	UpdateBootstrapSessionStatus(id string, status string) error
	GetExpiredBootstrapSessions() ([]*model.BootstrapSession, error)
	GetOrphanedBootstrapSessions() ([]*model.BootstrapSession, error)

	// Backup/Restore methods
	ExportDataForBackup() (*model.BackupData, error)
	ImportDataFromBackup(*model.BackupData) error
	IntegrateDataFromBackup(*model.BackupData) error

	// BunDB exposes the underlying *bun.DB for advanced operations or diagnostics.
	BunDB() *bun.DB
}
