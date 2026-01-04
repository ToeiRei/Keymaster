// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// package db provides the data access layer for Keymaster.
// This file contains the MySQL implementation of the database store.
// Note: This implementation is considered experimental.
package db // import "github.com/toeirei/keymaster/internal/db"

import (
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql" // MySQL driver
	"github.com/toeirei/keymaster/internal/model"
	"github.com/uptrace/bun"
)

// MySQLStore is the MySQL implementation of the Store interface.
type MySQLStore struct {
	bun *bun.DB
}

// BunDB returns the underlying *bun.DB for the MySQL store.
func (s *MySQLStore) BunDB() *bun.DB {
	return s.bun
}

// NewMySQLStore initializes the database connection and creates tables if they don't exist.
func NewMySQLStore(dataSourceName string) (*MySQLStore, error) {
	// The MySQL driver requires a DSN format like: "user:password@tcp(host:port)/dbname"
	// It's good practice to add `?parseTime=true` to handle DATETIME columns correctly.
	// This function is a lightweight placeholder. Actual initialization happens in `InitDB`.
	s, ok := store.(*MySQLStore)
	if !ok {
		return nil, fmt.Errorf("internal error: store is not a *MySQLStore")
	}
	return s, nil
}
func (s *MySQLStore) GetAllPublicKeys() ([]model.PublicKey, error) {
	return GetAllPublicKeysBun(s.bun)
}
func (s *MySQLStore) GetAllAccounts() ([]model.Account, error) {
	return GetAllAccountsBun(s.bun)
}

func (s *MySQLStore) AddAccount(username, hostname, label, tags string) (int, error) {
	id, err := AddAccountBun(s.bun, username, hostname, label, tags)
	if err == nil {
		_ = s.LogAction("ADD_ACCOUNT", fmt.Sprintf("account: %s@%s", username, hostname))
	}
	return id, err
}

func (s *MySQLStore) DeleteAccount(id int) error {
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

func (s *MySQLStore) UpdateAccountSerial(id, serial int) error {
	return UpdateAccountSerialBun(s.bun, id, serial)
}

func (s *MySQLStore) ToggleAccountStatus(id int) error {
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

func (s *MySQLStore) UpdateAccountLabel(id int, label string) error {
	err := UpdateAccountLabelBun(s.bun, id, label)
	if err == nil {
		_ = s.LogAction("UPDATE_ACCOUNT_LABEL", fmt.Sprintf("account_id: %d, new_label: '%s'", id, label))
	}
	return err
}

func (s *MySQLStore) UpdateAccountHostname(id int, hostname string) error {
	return UpdateAccountHostnameBun(s.bun, id, hostname)
}

func (s *MySQLStore) UpdateAccountTags(id int, tags string) error {
	err := UpdateAccountTagsBun(s.bun, id, tags)
	if err == nil {
		_ = s.LogAction("UPDATE_ACCOUNT_TAGS", fmt.Sprintf("account_id: %d, new_tags: '%s'", id, tags))
	}
	return err
}

func (s *MySQLStore) GetAllActiveAccounts() ([]model.Account, error) {
	return GetAllActiveAccountsBun(s.bun)
}

func (s *MySQLStore) AddPublicKey(algorithm, keyData, comment string, isGlobal bool) error {
	err := AddPublicKeyBun(s.bun, algorithm, keyData, comment, isGlobal)
	if err == nil {
		_ = s.LogAction("ADD_PUBLIC_KEY", fmt.Sprintf("comment: %s", comment))
	}
	return err
}
func (s *MySQLStore) GetPublicKeyByComment(comment string) (*model.PublicKey, error) {
	return GetPublicKeyByCommentBun(s.bun, comment)
}
func (s *MySQLStore) AddPublicKeyAndGetModel(algorithm, keyData, comment string, isGlobal bool) (*model.PublicKey, error) {
	pk, err := AddPublicKeyAndGetModelBun(s.bun, algorithm, keyData, comment, isGlobal)
	if err == nil && pk != nil {
		_ = s.LogAction("ADD_PUBLIC_KEY", fmt.Sprintf("comment: %s", comment))
	}
	return pk, err
}
func (s *MySQLStore) TogglePublicKeyGlobal(id int) error {
	err := TogglePublicKeyGlobalBun(s.bun, id)
	if err == nil {
		_ = s.LogAction("TOGGLE_KEY_GLOBAL", fmt.Sprintf("key_id: %d", id))
	}
	return err
}
func (s *MySQLStore) GetGlobalPublicKeys() ([]model.PublicKey, error) {
	return GetGlobalPublicKeysBun(s.bun)
}
func (s *MySQLStore) DeletePublicKey(id int) error {
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
func (s *MySQLStore) GetKnownHostKey(hostname string) (string, error) {
	return GetKnownHostKeyBun(s.bun, hostname)
}

func (s *MySQLStore) AddKnownHostKey(hostname, key string) error {
	err := AddKnownHostKeyBun(s.bun, hostname, key)
	if err == nil {
		_ = s.LogAction("TRUST_HOST", fmt.Sprintf("hostname: %s", hostname))
	}
	return err
}
func (s *MySQLStore) CreateSystemKey(publicKey, privateKey string) (int, error) {
	newSerial, err := CreateSystemKeyBun(s.bun, publicKey, privateKey)
	if err == nil {
		_ = s.LogAction("CREATE_SYSTEM_KEY", fmt.Sprintf("serial: %d", newSerial))
	}
	return newSerial, err
}
func (s *MySQLStore) RotateSystemKey(publicKey, privateKey string) (int, error) {
	newSerial, err := RotateSystemKeyBun(s.bun, publicKey, privateKey)
	if err == nil {
		_ = s.LogAction("ROTATE_SYSTEM_KEY", fmt.Sprintf("new_serial: %d", newSerial))
	}
	return newSerial, err
}
func (s *MySQLStore) GetActiveSystemKey() (*model.SystemKey, error) {
	return GetActiveSystemKeyBun(s.bun)
}
func (s *MySQLStore) GetSystemKeyBySerial(serial int) (*model.SystemKey, error) {
	return GetSystemKeyBySerialBun(s.bun, serial)
}
func (s *MySQLStore) HasSystemKeys() (bool, error) {
	return HasSystemKeysBun(s.bun)
}

// Key<->Account assignment methods are now provided by the `KeyManager`.
// Store implementations keep Bun helpers in `bun_adapter.go` for use by
// the KeyManager adapter.

// SearchAccounts performs a fuzzy search for accounts using the centralized Bun helper.
func (s *MySQLStore) SearchAccounts(query string) ([]model.Account, error) {
	// Use Bun-backed AccountSearcher adapter for search logic.
	return NewBunAccountSearcher(s.bun).SearchAccounts(query)
}
func (s *MySQLStore) GetAllAuditLogEntries() ([]model.AuditLogEntry, error) {
	return GetAllAuditLogEntriesBun(s.bun)
}

func (s *MySQLStore) LogAction(action string, details string) error {
	return LogActionBun(s.bun, action, details)
}

// SaveBootstrapSession saves a bootstrap session to the database.
func (s *MySQLStore) SaveBootstrapSession(id, username, hostname, label, tags, tempPublicKey string, expiresAt time.Time, status string) error {
	return SaveBootstrapSessionBun(s.bun, id, username, hostname, label, tags, tempPublicKey, expiresAt, status)
}

// GetBootstrapSession retrieves a bootstrap session by ID.
func (s *MySQLStore) GetBootstrapSession(id string) (*model.BootstrapSession, error) {
	return GetBootstrapSessionBun(s.bun, id)
}

// DeleteBootstrapSession removes a bootstrap session from the database.
func (s *MySQLStore) DeleteBootstrapSession(id string) error {
	return DeleteBootstrapSessionBun(s.bun, id)
}

// UpdateBootstrapSessionStatus updates the status of a bootstrap session.
func (s *MySQLStore) UpdateBootstrapSessionStatus(id string, status string) error {
	return UpdateBootstrapSessionStatusBun(s.bun, id, status)
}

// GetExpiredBootstrapSessions returns all expired bootstrap sessions.
func (s *MySQLStore) GetExpiredBootstrapSessions() ([]*model.BootstrapSession, error) {
	return GetExpiredBootstrapSessionsBun(s.bun)
}

// GetOrphanedBootstrapSessions returns all orphaned bootstrap sessions.
func (s *MySQLStore) GetOrphanedBootstrapSessions() ([]*model.BootstrapSession, error) {
	return GetOrphanedBootstrapSessionsBun(s.bun)
}

// ExportDataForBackup retrieves all data from the database for a backup.
// It uses a transaction to ensure a consistent snapshot of the data.
func (s *MySQLStore) ExportDataForBackup() (*model.BackupData, error) {
	return ExportDataForBackupBun(s.bun)
}

// ImportDataFromBackup restores the database from a backup data structure.
// It performs a full wipe-and-replace within a single transaction to ensure atomicity.
func (s *MySQLStore) ImportDataFromBackup(backup *model.BackupData) error {
	return ImportDataFromBackupBun(s.bun, backup)
}

// IntegrateDataFromBackup restores data from a backup in a non-destructive way,
// skipping entries that already exist.
func (s *MySQLStore) IntegrateDataFromBackup(backup *model.BackupData) error {
	return IntegrateDataFromBackupBun(s.bun, backup)
}
