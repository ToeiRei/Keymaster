// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// package db provides the data access layer for Keymaster.
// This file contains the MySQL implementation of the database store.
// Note: This implementation is considered experimental.
package db // import "github.com/toeirei/keymaster/internal/db"

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql" // MySQL driver
	"github.com/toeirei/keymaster/internal/model"
	"github.com/uptrace/bun"
)

// MySQLStore is the MySQL implementation of the Store interface.
type MySQLStore struct {
	db  *sql.DB
	bun *bun.DB
}

// NewMySQLStore initializes the database connection and creates tables if they don't exist.
func NewMySQLStore(dataSourceName string) (*MySQLStore, error) {
	// The MySQL driver requires a DSN format like: "user:password@tcp(host:port)/dbname"
	// It's good practice to add `?parseTime=true` to handle DATETIME columns correctly.
	// This function is now a placeholder. The actual initialization happens in InitDB.
	// It's kept for potential future logic specific to the store's creation.
	s, ok := store.(*MySQLStore)
	if !ok {
		return nil, fmt.Errorf("internal error: store is not a *MySQLStore")
	}
	return s, nil
}

// --- Stubbed Methods ---

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
	// This is primarily used for testing to point an account to a mock server.
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
func (s *MySQLStore) GetAllPublicKeys() ([]model.PublicKey, error) {
	return GetAllPublicKeysBun(s.bun)
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
func (s *MySQLStore) AssignKeyToAccount(keyID, accountID int) error {
	err := AssignKeyToAccountBun(s.bun, keyID, accountID)
	if err == nil {
		var keyComment, accUser, accHost string
		_ = s.db.QueryRow("SELECT comment FROM public_keys WHERE id = ?", keyID).Scan(&keyComment)
		_ = s.db.QueryRow("SELECT username, hostname FROM accounts WHERE id = ?", accountID).Scan(&accUser, &accHost)
		details := fmt.Sprintf("key: '%s' to account: %s@%s", keyComment, accUser, accHost)
		_ = s.LogAction("ASSIGN_KEY", details)
	}
	return err
}
func (s *MySQLStore) UnassignKeyFromAccount(keyID, accountID int) error {
	var keyComment, accUser, accHost string
	_ = s.db.QueryRow("SELECT comment FROM public_keys WHERE id = ?", keyID).Scan(&keyComment)
	_ = s.db.QueryRow("SELECT username, hostname FROM accounts WHERE id = ?", accountID).Scan(&accUser, &accHost)
	details := fmt.Sprintf("key: '%s' from account: %s@%s", keyComment, accUser, accHost)

	err := UnassignKeyFromAccountBun(s.bun, keyID, accountID)
	if err == nil {
		_ = s.LogAction("UNASSIGN_KEY", details)
	}
	return err
}
func (s *MySQLStore) GetKeysForAccount(accountID int) ([]model.PublicKey, error) {
	return GetKeysForAccountBun(s.bun, accountID)
}
func (s *MySQLStore) GetAccountsForKey(keyID int) ([]model.Account, error) {
	return GetAccountsForKeyBun(s.bun, keyID)
}
func (s *MySQLStore) GetAllAuditLogEntries() ([]model.AuditLogEntry, error) {
	return GetAllAuditLogEntriesBun(s.bun)
}

func (s *MySQLStore) LogAction(action string, details string) error {
	return LogActionBun(s.bun, action, details)
}

// SaveBootstrapSession saves a bootstrap session to the database.
func (s *MySQLStore) SaveBootstrapSession(id, username, hostname, label, tags, tempPublicKey string, expiresAt time.Time, status string) error {
	_, err := s.db.Exec(`INSERT INTO bootstrap_sessions (id, username, hostname, label, tags, temp_public_key, expires_at, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, username, hostname, label, tags, tempPublicKey, expiresAt, status)
	return err
}

// GetBootstrapSession retrieves a bootstrap session by ID.
func (s *MySQLStore) GetBootstrapSession(id string) (*model.BootstrapSession, error) {
	var session model.BootstrapSession
	var label, tags sql.NullString

	err := s.db.QueryRow(`SELECT id, username, hostname, label, tags, temp_public_key, created_at, expires_at, status
		FROM bootstrap_sessions WHERE id = ?`, id).Scan(
		&session.ID, &session.Username, &session.Hostname, &label, &tags,
		&session.TempPublicKey, &session.CreatedAt, &session.ExpiresAt, &session.Status)

	if err != nil {
		return nil, err
	}

	if label.Valid {
		session.Label = label.String
	}
	if tags.Valid {
		session.Tags = tags.String
	}

	return &session, nil
}

// DeleteBootstrapSession removes a bootstrap session from the database.
func (s *MySQLStore) DeleteBootstrapSession(id string) error {
	_, err := s.db.Exec("DELETE FROM bootstrap_sessions WHERE id = ?", id)
	return err
}

// UpdateBootstrapSessionStatus updates the status of a bootstrap session.
func (s *MySQLStore) UpdateBootstrapSessionStatus(id string, status string) error {
	_, err := s.db.Exec("UPDATE bootstrap_sessions SET status = ? WHERE id = ?", status, id)
	return err
}

// GetExpiredBootstrapSessions returns all expired bootstrap sessions.
func (s *MySQLStore) GetExpiredBootstrapSessions() ([]*model.BootstrapSession, error) {
	rows, err := s.db.Query(`SELECT id, username, hostname, label, tags, temp_public_key, created_at, expires_at, status
		FROM bootstrap_sessions WHERE expires_at < NOW()`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*model.BootstrapSession
	for rows.Next() {
		var session model.BootstrapSession
		var label, tags sql.NullString

		if err := rows.Scan(&session.ID, &session.Username, &session.Hostname, &label, &tags,
			&session.TempPublicKey, &session.CreatedAt, &session.ExpiresAt, &session.Status); err != nil {
			return nil, err
		}

		if label.Valid {
			session.Label = label.String
		}
		if tags.Valid {
			session.Tags = tags.String
		}

		sessions = append(sessions, &session)
	}

	return sessions, nil
}

// GetOrphanedBootstrapSessions returns all orphaned bootstrap sessions.
func (s *MySQLStore) GetOrphanedBootstrapSessions() ([]*model.BootstrapSession, error) {
	rows, err := s.db.Query(`SELECT id, username, hostname, label, tags, temp_public_key, created_at, expires_at, status
		FROM bootstrap_sessions WHERE status = 'orphaned'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*model.BootstrapSession
	for rows.Next() {
		var session model.BootstrapSession
		var label, tags sql.NullString

		if err := rows.Scan(&session.ID, &session.Username, &session.Hostname, &label, &tags,
			&session.TempPublicKey, &session.CreatedAt, &session.ExpiresAt, &session.Status); err != nil {
			return nil, err
		}

		if label.Valid {
			session.Label = label.String
		}
		if tags.Valid {
			session.Tags = tags.String
		}

		sessions = append(sessions, &session)
	}

	return sessions, nil
}

// ExportDataForBackup retrieves all data from the database for a backup.
// It uses a transaction to ensure a consistent snapshot of the data.
func (s *MySQLStore) ExportDataForBackup() (*model.BackupData, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Rollback on any error.

	backup := &model.BackupData{
		SchemaVersion: 1, // Set a schema version for future migrations.
	}

	// --- Export Accounts ---
	rows, err := tx.Query("SELECT id, username, hostname, label, tags, serial, is_active FROM accounts")
	if err != nil {
		return nil, fmt.Errorf("failed to export accounts: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var acc model.Account
		var label, tags sql.NullString
		if err := rows.Scan(&acc.ID, &acc.Username, &acc.Hostname, &label, &tags, &acc.Serial, &acc.IsActive); err != nil {
			return nil, fmt.Errorf("failed to scan account: %w", err)
		}
		acc.Label = label.String
		acc.Tags = tags.String
		backup.Accounts = append(backup.Accounts, acc)
	}

	// --- Export Public Keys ---
	rows, err = tx.Query("SELECT id, algorithm, key_data, comment, is_global FROM public_keys")
	if err != nil {
		return nil, fmt.Errorf("failed to export public keys: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var pk model.PublicKey
		if err := rows.Scan(&pk.ID, &pk.Algorithm, &pk.KeyData, &pk.Comment, &pk.IsGlobal); err != nil {
			return nil, fmt.Errorf("failed to scan public key: %w", err)
		}
		backup.PublicKeys = append(backup.PublicKeys, pk)
	}

	// --- Export AccountKeys (many-to-many) ---
	rows, err = tx.Query("SELECT key_id, account_id FROM account_keys")
	if err != nil {
		return nil, fmt.Errorf("failed to export account_keys: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var ak model.AccountKey
		if err := rows.Scan(&ak.KeyID, &ak.AccountID); err != nil {
			return nil, fmt.Errorf("failed to scan account_key: %w", err)
		}
		backup.AccountKeys = append(backup.AccountKeys, ak)
	}

	// --- Export System Keys ---
	rows, err = tx.Query("SELECT id, serial, public_key, private_key, is_active FROM system_keys")
	if err != nil {
		return nil, fmt.Errorf("failed to export system keys: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var sk model.SystemKey
		if err := rows.Scan(&sk.ID, &sk.Serial, &sk.PublicKey, &sk.PrivateKey, &sk.IsActive); err != nil {
			return nil, fmt.Errorf("failed to scan system key: %w", err)
		}
		backup.SystemKeys = append(backup.SystemKeys, sk)
	}

	// --- Export Known Hosts ---
	rows, err = tx.Query("SELECT hostname, `key` FROM known_hosts")
	if err != nil {
		return nil, fmt.Errorf("failed to export known hosts: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var kh model.KnownHost
		if err := rows.Scan(&kh.Hostname, &kh.Key); err != nil {
			return nil, fmt.Errorf("failed to scan known host: %w", err)
		}
		backup.KnownHosts = append(backup.KnownHosts, kh)
	}

	// --- Export Audit Log Entries ---
	rows, err = tx.Query("SELECT id, timestamp, username, action, details FROM audit_log")
	if err != nil {
		return nil, fmt.Errorf("failed to export audit log: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var ale model.AuditLogEntry
		if err := rows.Scan(&ale.ID, &ale.Timestamp, &ale.Username, &ale.Action, &ale.Details); err != nil {
			return nil, fmt.Errorf("failed to scan audit log entry: %w", err)
		}
		backup.AuditLogEntries = append(backup.AuditLogEntries, ale)
	}

	// --- Export Bootstrap Sessions ---
	rows, err = tx.Query("SELECT id, username, hostname, label, tags, temp_public_key, created_at, expires_at, status FROM bootstrap_sessions")
	if err != nil {
		return nil, fmt.Errorf("failed to export bootstrap sessions: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var bs model.BootstrapSession
		var label, tags sql.NullString
		if err := rows.Scan(&bs.ID, &bs.Username, &bs.Hostname, &label, &tags, &bs.TempPublicKey, &bs.CreatedAt, &bs.ExpiresAt, &bs.Status); err != nil {
			return nil, fmt.Errorf("failed to scan bootstrap session: %w", err)
		}
		bs.Label = label.String
		bs.Tags = tags.String
		backup.BootstrapSessions = append(backup.BootstrapSessions, bs)
	}

	// If we got here, all queries were successful.
	return backup, tx.Commit()
}

// ImportDataFromBackup restores the database from a backup data structure.
// It performs a full wipe-and-replace within a single transaction to ensure atomicity.
func (s *MySQLStore) ImportDataFromBackup(backup *model.BackupData) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Rollback on any error.

	// --- 1. DB Wipe ---
	// Temporarily disable foreign key checks to allow truncating tables in any order.
	if _, err := tx.Exec("SET FOREIGN_KEY_CHECKS = 0"); err != nil {
		return fmt.Errorf("failed to disable foreign key checks: %w", err)
	}

	tables := []string{
		"accounts",
		"public_keys",
		"account_keys",
		"system_keys",
		"known_hosts",
		"audit_log",
		"bootstrap_sessions",
	}
	for _, table := range tables {
		if _, err := tx.Exec(fmt.Sprintf("TRUNCATE TABLE %s", table)); err != nil {
			return fmt.Errorf("failed to wipe table %s: %w", table, err)
		}
	}

	// Re-enable foreign key checks.
	defer func() {
		// This will run even if the main function returns early.
		if _, err := tx.Exec("SET FOREIGN_KEY_CHECKS = 1"); err != nil {
			// We can't do much here besides log it, as we might already be in an error path.
		}
	}()

	// --- 2. DB Integration (Insertion) ---

	// Accounts
	stmt, err := tx.Prepare("INSERT INTO accounts (id, username, hostname, label, tags, serial, is_active) VALUES (?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return fmt.Errorf("failed to prepare account insert: %w", err)
	}
	for _, acc := range backup.Accounts {
		if _, err := stmt.Exec(acc.ID, acc.Username, acc.Hostname, acc.Label, acc.Tags, acc.Serial, acc.IsActive); err != nil {
			return fmt.Errorf("failed to insert account %d: %w", acc.ID, err)
		}
	}
	stmt.Close()

	// Public Keys
	stmt, err = tx.Prepare("INSERT INTO public_keys (id, algorithm, key_data, comment, is_global) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return fmt.Errorf("failed to prepare public_key insert: %w", err)
	}
	for _, pk := range backup.PublicKeys {
		if _, err := stmt.Exec(pk.ID, pk.Algorithm, pk.KeyData, pk.Comment, pk.IsGlobal); err != nil {
			return fmt.Errorf("failed to insert public key %d: %w", pk.ID, err)
		}
	}
	stmt.Close()

	// AccountKeys
	stmt, err = tx.Prepare("INSERT INTO account_keys (key_id, account_id) VALUES (?, ?)")
	if err != nil {
		return fmt.Errorf("failed to prepare account_key insert: %w", err)
	}
	for _, ak := range backup.AccountKeys {
		if _, err := stmt.Exec(ak.KeyID, ak.AccountID); err != nil {
			return fmt.Errorf("failed to insert account_key for key %d and account %d: %w", ak.KeyID, ak.AccountID, err)
		}
	}
	stmt.Close()

	// System Keys
	stmt, err = tx.Prepare("INSERT INTO system_keys (id, serial, public_key, private_key, is_active) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return fmt.Errorf("failed to prepare system_key insert: %w", err)
	}
	for _, sk := range backup.SystemKeys {
		if _, err := stmt.Exec(sk.ID, sk.Serial, sk.PublicKey, sk.PrivateKey, sk.IsActive); err != nil {
			return fmt.Errorf("failed to insert system key %d: %w", sk.ID, err)
		}
	}
	stmt.Close()

	// Known Hosts
	stmt, err = tx.Prepare("INSERT INTO known_hosts (hostname, `key`) VALUES (?, ?)")
	if err != nil {
		return fmt.Errorf("failed to prepare known_host insert: %w", err)
	}
	for _, kh := range backup.KnownHosts {
		if _, err := stmt.Exec(kh.Hostname, kh.Key); err != nil {
			return fmt.Errorf("failed to insert known host %s: %w", kh.Hostname, err)
		}
	}
	stmt.Close()

	// Audit Log Entries
	stmt, err = tx.Prepare("INSERT INTO audit_log (id, timestamp, username, action, details) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return fmt.Errorf("failed to prepare audit_log insert: %w", err)
	}
	for _, ale := range backup.AuditLogEntries {
		// The timestamp from the JSON backup is in RFC3339 format ("2006-01-02T15:04:05Z").
		// We need to parse it into a time.Time object first, then format it into the
		// 'YYYY-MM-DD HH:MM:SS' format that MySQL's DATETIME type expects.
		t, err := time.Parse(time.RFC3339, ale.Timestamp)
		if err != nil {
			return fmt.Errorf("failed to parse timestamp for audit log %d: %w", ale.ID, err)
		}
		if _, err := stmt.Exec(ale.ID, t.Format("2006-01-02 15:04:05"), ale.Username, ale.Action, ale.Details); err != nil {
			return fmt.Errorf("failed to insert audit log %d: %w", ale.ID, err)
		}
	}
	stmt.Close()

	return tx.Commit()
}

// IntegrateDataFromBackup restores data from a backup in a non-destructive way,
// skipping entries that already exist.
func (s *MySQLStore) IntegrateDataFromBackup(backup *model.BackupData) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Rollback on any error.

	// Use "INSERT IGNORE" to skip duplicates based on unique constraints.

	// Accounts (UNIQUE on username, hostname)
	stmt, err := tx.Prepare("INSERT IGNORE INTO accounts (id, username, hostname, label, tags, serial, is_active) VALUES (?, ?, ?, ?, ?, ?, ?)")
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
	stmt, err = tx.Prepare("INSERT IGNORE INTO public_keys (id, algorithm, key_data, comment, is_global) VALUES (?, ?, ?, ?, ?)")
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
	stmt, err = tx.Prepare("INSERT IGNORE INTO account_keys (key_id, account_id) VALUES (?, ?)")
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
	stmt, err = tx.Prepare("INSERT IGNORE INTO system_keys (id, serial, public_key, private_key, is_active) VALUES (?, ?, ?, ?, ?)")
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
	stmt, err = tx.Prepare("INSERT IGNORE INTO known_hosts (hostname, `key`) VALUES (?, ?)")
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
