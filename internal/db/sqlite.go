// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// package db provides the data access layer for Keymaster.
// This file contains the SQLite implementation of the database store.
package db // import "github.com/toeirei/keymaster/internal/db"

import (
	"fmt"
	"time"

	"github.com/toeirei/keymaster/internal/model"
	"github.com/uptrace/bun"
)

// SqliteStore is the SQLite implementation of the Store interface.
type SqliteStore struct {
	bun *bun.DB
}

// BunDB returns the underlying *bun.DB for the sqlite store.
func (s *SqliteStore) BunDB() *bun.DB {
	return s.bun
}

// NewSqliteStore initializes the database connection and creates tables if they don't exist.
func NewSqliteStore(dataSourceName string) (*SqliteStore, error) {
	// Construct a new SqliteStore using the central NewStoreFromDSN helper.
	s, err := NewStoreFromDSN("sqlite", dataSourceName)
	if err != nil {
		return nil, err
	}
	ss, ok := s.(*SqliteStore)
	if !ok {
		return nil, fmt.Errorf("internal error: expected *SqliteStore, got %T", s)
	}
	return ss, nil
}

// GetAllAccounts retrieves all accounts from the database.
func (s *SqliteStore) GetAllAccounts() ([]model.Account, error) {
	return GetAllAccountsBun(s.bun)
}

// AddAccount adds a new account to the database.
func (s *SqliteStore) AddAccount(username, hostname, label, tags string) (int, error) {
	id, err := AddAccountBun(s.bun, username, hostname, label, tags)
	if err == nil {
		_ = s.LogAction("ADD_ACCOUNT", fmt.Sprintf("account: %s@%s", username, hostname))
	}
	return id, err
}

// DeleteAccount removes an account from the database by its ID.
func (s *SqliteStore) DeleteAccount(id int) error {
	// Get account details before deleting for logging (best-effort via Bun).
	details := fmt.Sprintf("id: %d", id)
	if acc, err2 := GetAccountByIDBun(s.bun, id); err2 == nil && acc != nil {
		details = fmt.Sprintf("account: %s@%s", acc.Username, acc.Hostname)
	}
	err := DeleteAccountBun(s.bun, id)
	if err == nil {
		_ = s.LogAction("DELETE_ACCOUNT", details)
	}
	return err
}

// UpdateAccountSerial sets the serial for a given account ID to a specific value.
func (s *SqliteStore) UpdateAccountSerial(id, serial int) error {
	return UpdateAccountSerialBun(s.bun, id, serial)
}

// ToggleAccountStatus flips the active status of an account.
func (s *SqliteStore) ToggleAccountStatus(id int) error {
	// Get account details before toggling for logging.
	acc, err := GetAccountByIDBun(s.bun, id)
	if err != nil {
		return err
	}
	if acc == nil {
		return fmt.Errorf("account not found: %d", id)
	}
	newStatus, err := ToggleAccountStatusBun(s.bun, id)
	if err == nil {
		details := fmt.Sprintf("account: %s@%s, new_status: %t", acc.Username, acc.Hostname, newStatus)
		_ = s.LogAction("TOGGLE_ACCOUNT_STATUS", details)
	}
	return err
}

// UpdateAccountLabel updates the label for a given account.
func (s *SqliteStore) UpdateAccountLabel(id int, label string) error {
	err := UpdateAccountLabelBun(s.bun, id, label)
	if err == nil {
		_ = s.LogAction("UPDATE_ACCOUNT_LABEL", fmt.Sprintf("account_id: %d, new_label: '%s'", id, label))
	}
	return err
}

// UpdateAccountHostname updates the hostname for a given account.
// This is primarily used for testing to point an account to a mock server.
func (s *SqliteStore) UpdateAccountHostname(id int, hostname string) error {
	return UpdateAccountHostnameBun(s.bun, id, hostname)
}

// UpdateAccountTags updates the tags for a given account.
func (s *SqliteStore) UpdateAccountTags(id int, tags string) error {
	err := UpdateAccountTagsBun(s.bun, id, tags)
	if err == nil {
		_ = s.LogAction("UPDATE_ACCOUNT_TAGS", fmt.Sprintf("account_id: %d, new_tags: '%s'", id, tags))
	}
	return err
}

// GetAllActiveAccounts retrieves all active accounts from the database.
func (s *SqliteStore) GetAllActiveAccounts() ([]model.Account, error) {
	return GetAllActiveAccountsBun(s.bun)
}

// Public-key CRUD is provided by KeyManager; store keeps Bun helpers.

// GetKnownHostKey retrieves the trusted public key for a given hostname.
func (s *SqliteStore) GetKnownHostKey(hostname string) (string, error) {
	return GetKnownHostKeyBun(s.bun, hostname)
}

// AddKnownHostKey adds a new trusted host key to the database.
func (s *SqliteStore) AddKnownHostKey(hostname, key string) error {
	err := AddKnownHostKeyBun(s.bun, hostname, key)
	if err == nil {
		_ = s.LogAction("TRUST_HOST", fmt.Sprintf("hostname: %s", hostname))
	}
	return err
}

// CreateSystemKey adds a new system key to the database. It determines the correct serial automatically.
func (s *SqliteStore) CreateSystemKey(publicKey, privateKey string) (int, error) {
	newSerial, err := CreateSystemKeyBun(s.bun, publicKey, privateKey)
	if err == nil {
		_ = s.LogAction("CREATE_SYSTEM_KEY", fmt.Sprintf("serial: %d", newSerial))
	}
	return newSerial, err
}

// RotateSystemKey deactivates all current system keys and adds a new one as active.
// This should be performed within a transaction to ensure atomicity.
func (s *SqliteStore) RotateSystemKey(publicKey, privateKey string) (int, error) {
	// Delegate to the Bun-based implementation for SQLite. This is an
	// incremental step toward a Bun-backed sqlite store and keeps the
	// transactional semantics while using Bun's helper for portability.
	newSerial, err := RotateSystemKeyBun(s.bun, publicKey, privateKey)
	if err == nil {
		_ = s.LogAction("ROTATE_SYSTEM_KEY", fmt.Sprintf("new_serial: %d", newSerial))
	}
	return newSerial, err
}

// GetActiveSystemKey retrieves the currently active system key for deployments.
func (s *SqliteStore) GetActiveSystemKey() (*model.SystemKey, error) {
	return GetActiveSystemKeyBun(s.bun)
}

// GetSystemKeyBySerial retrieves a system key by its serial number.
func (s *SqliteStore) GetSystemKeyBySerial(serial int) (*model.SystemKey, error) {
	return GetSystemKeyBySerialBun(s.bun, serial)
}

// HasSystemKeys checks if any system keys exist in the database.
func (s *SqliteStore) HasSystemKeys() (bool, error) {
	return HasSystemKeysBun(s.bun)
}

// DeletePublicKey removes a public key and all its associations.
// The ON DELETE CASCADE constraint handles the associations in account_keys.
func (s *SqliteStore) DeletePublicKey(id int) error {
	// Get key comment before deleting for logging (best-effort via Bun).
	details := fmt.Sprintf("id: %d", id)
	if pk, err2 := GetPublicKeyByIDBun(s.bun, id); err2 == nil && pk != nil {
		details = fmt.Sprintf("comment: %s", pk.Comment)
	}
	err := DeletePublicKeyBun(s.bun, id)
	if err == nil {
		_ = s.LogAction("DELETE_PUBLIC_KEY", details)
	}
	return err
}

// Key<->Account assignment methods are provided by the `KeyManager` and
// implemented via Bun helpers (see `searcher.go` for the adapter).

// SearchAccounts performs a fuzzy search for accounts using the centralized Bun helper.
func (s *SqliteStore) SearchAccounts(query string) ([]model.Account, error) {
	// Use Bun-backed AccountSearcher adapter for search logic.
	return NewBunAccountSearcher(s.bun).SearchAccounts(query)
}

// GetAllAuditLogEntries retrieves all entries from the audit log, most recent first.
func (s *SqliteStore) GetAllAuditLogEntries() ([]model.AuditLogEntry, error) {
	return GetAllAuditLogEntriesBun(s.bun)
}

// LogAction records an audit trail event.
func (s *SqliteStore) LogAction(action string, details string) error {
	return LogActionBun(s.bun, action, details)
}

// SaveBootstrapSession saves a bootstrap session to the database.
func (s *SqliteStore) SaveBootstrapSession(id, username, hostname, label, tags, tempPublicKey string, expiresAt time.Time, status string) error {
	return SaveBootstrapSessionBun(s.bun, id, username, hostname, label, tags, tempPublicKey, expiresAt, status)
}

// GetBootstrapSession retrieves a bootstrap session by ID.
func (s *SqliteStore) GetBootstrapSession(id string) (*model.BootstrapSession, error) {
	return GetBootstrapSessionBun(s.bun, id)
}

// DeleteBootstrapSession removes a bootstrap session from the database.
func (s *SqliteStore) DeleteBootstrapSession(id string) error {
	return DeleteBootstrapSessionBun(s.bun, id)
}

// UpdateBootstrapSessionStatus updates the status of a bootstrap session.
func (s *SqliteStore) UpdateBootstrapSessionStatus(id string, status string) error {
	return UpdateBootstrapSessionStatusBun(s.bun, id, status)
}

// GetExpiredBootstrapSessions returns all expired bootstrap sessions.
func (s *SqliteStore) GetExpiredBootstrapSessions() ([]*model.BootstrapSession, error) {
	return GetExpiredBootstrapSessionsBun(s.bun)
}

// GetOrphanedBootstrapSessions returns all orphaned bootstrap sessions.
func (s *SqliteStore) GetOrphanedBootstrapSessions() ([]*model.BootstrapSession, error) {
	return GetOrphanedBootstrapSessionsBun(s.bun)
}

// ExportDataForBackup retrieves all data from the database for a backup.
// It uses a transaction to ensure a consistent snapshot of the data.
func (s *SqliteStore) ExportDataForBackup() (*model.BackupData, error) {
	return ExportDataForBackupBun(s.bun)
}

// ImportDataFromBackup restores the database from a backup data structure.
// It performs a full wipe-and-replace within a single transaction to ensure atomicity.
func (s *SqliteStore) ImportDataFromBackup(backup *model.BackupData) error {
	return ImportDataFromBackupBun(s.bun, backup)
}

// IntegrateDataFromBackup restores data from a backup in a non-destructive way,
// skipping entries that already exist.
func (s *SqliteStore) IntegrateDataFromBackup(backup *model.BackupData) error {
	return IntegrateDataFromBackupBun(s.bun, backup)
}
