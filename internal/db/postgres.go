// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// package db provides the data access layer for Keymaster.
// This file contains the PostgreSQL implementation of the database store.
// Note: This implementation is considered experimental.
package db // import "github.com/toeirei/keymaster/internal/db"

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // PostgreSQL driver
	"github.com/toeirei/keymaster/internal/model"
	"github.com/uptrace/bun"
)

// PostgresStore is the PostgreSQL implementation of the Store interface.
type PostgresStore struct {
	db  *sql.DB
	bun *bun.DB
}

// NewPostgresStore initializes the database connection and creates tables if they don't exist.
func NewPostgresStore(dataSourceName string) (*PostgresStore, error) {
	// This function is now a placeholder. The actual initialization happens in InitDB.
	// It's kept for potential future logic specific to the store's creation.
	s, ok := store.(*PostgresStore)
	if !ok {
		return nil, fmt.Errorf("internal error: store is not a *PostgresStore")
	}
	return s, nil
}

// --- Stubbed Methods ---

func (s *PostgresStore) GetAllAccounts() ([]model.Account, error) {
	return GetAllAccountsBun(s.bun)
}

func (s *PostgresStore) AddAccount(username, hostname, label, tags string) (int, error) {
	id, err := AddAccountBun(s.bun, username, hostname, label, tags)
	if err == nil {
		_ = s.LogAction("ADD_ACCOUNT", fmt.Sprintf("account: %s@%s", username, hostname))
	}
	return id, err
}

func (s *PostgresStore) DeleteAccount(id int) error {
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

func (s *PostgresStore) UpdateAccountSerial(id, serial int) error {
	return UpdateAccountSerialBun(s.bun, id, serial)
}

func (s *PostgresStore) ToggleAccountStatus(id int) error {
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

func (s *PostgresStore) UpdateAccountLabel(id int, label string) error {
	err := UpdateAccountLabelBun(s.bun, id, label)
	if err == nil {
		_ = s.LogAction("UPDATE_ACCOUNT_LABEL", fmt.Sprintf("account_id: %d, new_label: '%s'", id, label))
	}
	return err
}

func (s *PostgresStore) UpdateAccountHostname(id int, hostname string) error {
	return UpdateAccountHostnameBun(s.bun, id, hostname)
}

func (s *PostgresStore) UpdateAccountTags(id int, tags string) error {
	err := UpdateAccountTagsBun(s.bun, id, tags)
	if err == nil {
		_ = s.LogAction("UPDATE_ACCOUNT_TAGS", fmt.Sprintf("account_id: %d, new_tags: '%s'", id, tags))
	}
	return err
}

func (s *PostgresStore) GetAllActiveAccounts() ([]model.Account, error) {
	return GetAllActiveAccountsBun(s.bun)
}

func (s *PostgresStore) AddPublicKey(algorithm, keyData, comment string, isGlobal bool) error {
	err := AddPublicKeyBun(s.bun, algorithm, keyData, comment, isGlobal)
	if err == nil {
		_ = s.LogAction("ADD_PUBLIC_KEY", fmt.Sprintf("comment: %s", comment))
	}
	return err
}

func (s *PostgresStore) GetAllPublicKeys() ([]model.PublicKey, error) {
	return GetAllPublicKeysBun(s.bun)
}

func (s *PostgresStore) GetPublicKeyByComment(comment string) (*model.PublicKey, error) {
	return GetPublicKeyByCommentBun(s.bun, comment)
}

func (s *PostgresStore) AddPublicKeyAndGetModel(algorithm, keyData, comment string, isGlobal bool) (*model.PublicKey, error) {
	pk, err := AddPublicKeyAndGetModelBun(s.bun, algorithm, keyData, comment, isGlobal)
	if err == nil && pk != nil {
		_ = s.LogAction("ADD_PUBLIC_KEY", fmt.Sprintf("comment: %s", comment))
	}
	return pk, err
}

func (s *PostgresStore) TogglePublicKeyGlobal(id int) error {
	err := TogglePublicKeyGlobalBun(s.bun, id)
	if err == nil {
		_ = s.LogAction("TOGGLE_KEY_GLOBAL", fmt.Sprintf("key_id: %d", id))
	}
	return err
}

func (s *PostgresStore) GetGlobalPublicKeys() ([]model.PublicKey, error) {
	return GetGlobalPublicKeysBun(s.bun)
}

func (s *PostgresStore) DeletePublicKey(id int) error {
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
func (s *PostgresStore) GetKnownHostKey(hostname string) (string, error) {
	return GetKnownHostKeyBun(s.bun, hostname)
}

func (s *PostgresStore) AddKnownHostKey(hostname, key string) error {
	err := AddKnownHostKeyBun(s.bun, hostname, key)
	if err == nil {
		_ = s.LogAction("TRUST_HOST", fmt.Sprintf("hostname: %s", hostname))
	}
	return err
}

func (s *PostgresStore) CreateSystemKey(publicKey, privateKey string) (int, error) {
	newSerial, err := CreateSystemKeyBun(s.bun, publicKey, privateKey)
	if err == nil {
		_ = s.LogAction("CREATE_SYSTEM_KEY", fmt.Sprintf("serial: %d", newSerial))
	}
	return newSerial, err
}

func (s *PostgresStore) RotateSystemKey(publicKey, privateKey string) (int, error) {
	newSerial, err := RotateSystemKeyBun(s.bun, publicKey, privateKey)
	if err == nil {
		_ = s.LogAction("ROTATE_SYSTEM_KEY", fmt.Sprintf("new_serial: %d", newSerial))
	}
	return newSerial, err
}

func (s *PostgresStore) GetActiveSystemKey() (*model.SystemKey, error) {
	return GetActiveSystemKeyBun(s.bun)
}

func (s *PostgresStore) GetSystemKeyBySerial(serial int) (*model.SystemKey, error) {
	return GetSystemKeyBySerialBun(s.bun, serial)
}

func (s *PostgresStore) HasSystemKeys() (bool, error) {
	return HasSystemKeysBun(s.bun)
}

func (s *PostgresStore) AssignKeyToAccount(keyID, accountID int) error {
	err := AssignKeyToAccountBun(s.bun, keyID, accountID)
	if err == nil {
		var keyComment, accUser, accHost string
		_ = s.db.QueryRow("SELECT comment FROM public_keys WHERE id = $1", keyID).Scan(&keyComment)
		_ = s.db.QueryRow("SELECT username, hostname FROM accounts WHERE id = $1", accountID).Scan(&accUser, &accHost)
		details := fmt.Sprintf("key: '%s' to account: %s@%s", keyComment, accUser, accHost)
		_ = s.LogAction("ASSIGN_KEY", details)
	}
	return err
}

func (s *PostgresStore) UnassignKeyFromAccount(keyID, accountID int) error {
	var keyComment, accUser, accHost string
	_ = s.db.QueryRow("SELECT comment FROM public_keys WHERE id = $1", keyID).Scan(&keyComment)
	_ = s.db.QueryRow("SELECT username, hostname FROM accounts WHERE id = $1", accountID).Scan(&accUser, &accHost)
	details := fmt.Sprintf("key: '%s' from account: %s@%s", keyComment, accUser, accHost)
	err := UnassignKeyFromAccountBun(s.bun, keyID, accountID)
	if err == nil {
		_ = s.LogAction("UNASSIGN_KEY", details)
	}
	return err
}

func (s *PostgresStore) GetKeysForAccount(accountID int) ([]model.PublicKey, error) {
	return GetKeysForAccountBun(s.bun, accountID)
}

func (s *PostgresStore) GetAccountsForKey(keyID int) ([]model.Account, error) {
	return GetAccountsForKeyBun(s.bun, keyID)
}

func (s *PostgresStore) GetAllAuditLogEntries() ([]model.AuditLogEntry, error) {
	return GetAllAuditLogEntriesBun(s.bun)
}

func (s *PostgresStore) LogAction(action string, details string) error {
	// Delegate to Bun-backed helper which also derives current OS user.
	return LogActionBun(s.bun, action, details)
}

// SaveBootstrapSession saves a bootstrap session to the database.
func (s *PostgresStore) SaveBootstrapSession(id, username, hostname, label, tags, tempPublicKey string, expiresAt time.Time, status string) error {
	return SaveBootstrapSessionBun(s.bun, id, username, hostname, label, tags, tempPublicKey, expiresAt, status)
}

// GetBootstrapSession retrieves a bootstrap session by ID.
func (s *PostgresStore) GetBootstrapSession(id string) (*model.BootstrapSession, error) {
	return GetBootstrapSessionBun(s.bun, id)
}

// DeleteBootstrapSession removes a bootstrap session from the database.
func (s *PostgresStore) DeleteBootstrapSession(id string) error {
	return DeleteBootstrapSessionBun(s.bun, id)
}

// UpdateBootstrapSessionStatus updates the status of a bootstrap session.
func (s *PostgresStore) UpdateBootstrapSessionStatus(id string, status string) error {
	return UpdateBootstrapSessionStatusBun(s.bun, id, status)
}

// GetExpiredBootstrapSessions returns all expired bootstrap sessions.
func (s *PostgresStore) GetExpiredBootstrapSessions() ([]*model.BootstrapSession, error) {
	return GetExpiredBootstrapSessionsBun(s.bun)
}

// GetOrphanedBootstrapSessions returns all orphaned bootstrap sessions.
func (s *PostgresStore) GetOrphanedBootstrapSessions() ([]*model.BootstrapSession, error) {
	return GetOrphanedBootstrapSessionsBun(s.bun)
}

// ExportDataForBackup retrieves all data from the database for a backup.
// It uses a transaction to ensure a consistent snapshot of the data.
func (s *PostgresStore) ExportDataForBackup() (*model.BackupData, error) {
	return ExportDataForBackupBun(s.bun)
}

// ImportDataFromBackup restores the database from a backup data structure.
// It performs a full wipe-and-replace within a single transaction to ensure atomicity.
func (s *PostgresStore) ImportDataFromBackup(backup *model.BackupData) error {
	return ImportDataFromBackupBun(s.bun, backup)
}

// IntegrateDataFromBackup restores data from a backup in a non-destructive way,
// skipping entries that already exist.
func (s *PostgresStore) IntegrateDataFromBackup(backup *model.BackupData) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Rollback on any error.

	// Use "ON CONFLICT DO NOTHING" to skip duplicates based on unique constraints.

	// Accounts (UNIQUE on username, hostname)
	stmt, err := tx.Prepare("INSERT INTO accounts (id, username, hostname, label, tags, serial, is_active) VALUES ($1, $2, $3, $4, $5, $6, $7) ON CONFLICT (username, hostname) DO NOTHING")
	if err != nil {
		return fmt.Errorf("failed to prepare account insert: %w", err)
	}
	for _, acc := range backup.Accounts {
		if _, err := stmt.Exec(acc.ID, acc.Username, acc.Hostname, acc.Label, acc.Tags, acc.Serial, acc.IsActive); err != nil {
			return fmt.Errorf("failed to integrate account %d: %w", acc.ID, err)
		}
	}
	stmt.Close()

	// Public Keys (UNIQUE on comment)
	stmt, err = tx.Prepare("INSERT INTO public_keys (id, algorithm, key_data, comment, is_global) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (comment) DO NOTHING")
	if err != nil {
		return fmt.Errorf("failed to prepare public_key insert: %w", err)
	}
	for _, pk := range backup.PublicKeys {
		if _, err := stmt.Exec(pk.ID, pk.Algorithm, pk.KeyData, pk.Comment, pk.IsGlobal); err != nil {
			return fmt.Errorf("failed to integrate public key %d: %w", pk.ID, err)
		}
	}
	stmt.Close()

	// AccountKeys (PRIMARY KEY on key_id, account_id)
	stmt, err = tx.Prepare("INSERT INTO account_keys (key_id, account_id) VALUES ($1, $2) ON CONFLICT (key_id, account_id) DO NOTHING")
	if err != nil {
		return fmt.Errorf("failed to prepare account_key insert: %w", err)
	}
	for _, ak := range backup.AccountKeys {
		if _, err := stmt.Exec(ak.KeyID, ak.AccountID); err != nil {
			return fmt.Errorf("failed to integrate account_key for key %d and account %d: %w", ak.KeyID, ak.AccountID, err)
		}
	}
	stmt.Close()

	// System Keys (UNIQUE on serial)
	stmt, err = tx.Prepare("INSERT INTO system_keys (id, serial, public_key, private_key, is_active) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (serial) DO NOTHING")
	if err != nil {
		return fmt.Errorf("failed to prepare system_key insert: %w", err)
	}
	for _, sk := range backup.SystemKeys {
		if _, err := stmt.Exec(sk.ID, sk.Serial, sk.PublicKey, sk.PrivateKey, sk.IsActive); err != nil {
			return fmt.Errorf("failed to integrate system key %d: %w", sk.ID, err)
		}
	}
	stmt.Close()

	// Known Hosts (PRIMARY KEY on hostname)
	stmt, err = tx.Prepare(`INSERT INTO known_hosts (hostname, "key") VALUES ($1, $2) ON CONFLICT (hostname) DO NOTHING`)
	if err != nil {
		return fmt.Errorf("failed to prepare known_host insert: %w", err)
	}
	for _, kh := range backup.KnownHosts {
		if _, err := stmt.Exec(kh.Hostname, kh.Key); err != nil {
			return fmt.Errorf("failed to integrate known host %s: %w", kh.Hostname, err)
		}
	}
	stmt.Close()

	// Audit logs and bootstrap sessions are generally not integrated to avoid confusion.

	return tx.Commit()
}
