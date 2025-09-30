// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// package db provides the data access layer for Keymaster.
// This file contains the SQLite implementation of the database store.
package db // import "github.com/toeirei/keymaster/internal/db"

import (
	"database/sql"
	"fmt"
	"os/user"
	"strings"
	"time"

	"github.com/toeirei/keymaster/internal/model"
	_ "modernc.org/sqlite" // Pure Go SQLite driver
)

// SqliteStore is the SQLite implementation of the Store interface.
type SqliteStore struct {
	db *sql.DB
}

// NewSqliteStore initializes the database connection and creates tables if they don't exist.
func NewSqliteStore(dataSourceName string) (*SqliteStore, error) {
	// This function is now a placeholder. The actual initialization happens in InitDB.
	// It's kept for potential future logic specific to the store's creation.
	s, ok := store.(*SqliteStore)
	if !ok {
		return nil, fmt.Errorf("internal error: store is not a *SqliteStore")
	}
	return s, nil
}

// GetAllAccounts retrieves all accounts from the database.
func (s *SqliteStore) GetAllAccounts() ([]model.Account, error) {
	rows, err := s.db.Query("SELECT id, username, hostname, label, tags, serial, is_active FROM accounts ORDER BY label, hostname, username")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []model.Account
	for rows.Next() {
		var acc model.Account
		var label sql.NullString
		var tags sql.NullString
		if err := rows.Scan(&acc.ID, &acc.Username, &acc.Hostname, &label, &tags, &acc.Serial, &acc.IsActive); err != nil {
			return nil, err
		}
		if label.Valid {
			acc.Label = label.String
		}
		if tags.Valid {
			acc.Tags = tags.String
		}
		accounts = append(accounts, acc)
	}
	return accounts, nil
}

// AddAccount adds a new account to the database.
func (s *SqliteStore) AddAccount(username, hostname, label, tags string) (int, error) {
	result, err := s.db.Exec("INSERT INTO accounts(username, hostname, label, tags) VALUES(?, ?, ?, ?)", username, hostname, label, tags)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert ID: %w", err)
	}
	_ = s.LogAction("ADD_ACCOUNT", fmt.Sprintf("account: %s@%s", username, hostname))
	return int(id), err
}

// DeleteAccount removes an account from the database by its ID.
func (s *SqliteStore) DeleteAccount(id int) error {
	// Get account details before deleting for logging.
	var username, hostname string
	err := s.db.QueryRow("SELECT username, hostname FROM accounts WHERE id = ?", id).Scan(&username, &hostname)
	details := fmt.Sprintf("id: %d", id)
	if err == nil {
		details = fmt.Sprintf("account: %s@%s", username, hostname)
	}

	_, err = s.db.Exec("DELETE FROM accounts WHERE id = ?", id)
	if err == nil {
		_ = s.LogAction("DELETE_ACCOUNT", details)
	}
	return err
}

// UpdateAccountSerial sets the serial for a given account ID to a specific value.
func (s *SqliteStore) UpdateAccountSerial(id, serial int) error {
	_, err := s.db.Exec("UPDATE accounts SET serial = ? WHERE id = ?", serial, id)
	// This is called during deployment, which is logged at a higher level.
	// No need for a separate log action here.
	return err
}

// ToggleAccountStatus flips the active status of an account.
func (s *SqliteStore) ToggleAccountStatus(id int) error {
	// Get account details before toggling for logging.
	var username, hostname string
	var isActive bool
	err := s.db.QueryRow("SELECT username, hostname, is_active FROM accounts WHERE id = ?", id).Scan(&username, &hostname, &isActive)
	if err != nil {
		return err // If we can't find it, we can't toggle it.
	}

	_, err = s.db.Exec("UPDATE accounts SET is_active = NOT is_active WHERE id = ?", id)
	if err == nil {
		details := fmt.Sprintf("account: %s@%s, new_status: %t", username, hostname, !isActive)
		_ = s.LogAction("TOGGLE_ACCOUNT_STATUS", details)
	}
	return err
}

// UpdateAccountLabel updates the label for a given account.
func (s *SqliteStore) UpdateAccountLabel(id int, label string) error {
	_, err := s.db.Exec("UPDATE accounts SET label = ? WHERE id = ?", label, id)
	if err == nil {
		_ = s.LogAction("UPDATE_ACCOUNT_LABEL", fmt.Sprintf("account_id: %d, new_label: '%s'", id, label))
	}
	return err
}

// UpdateAccountHostname updates the hostname for a given account.
// This is primarily used for testing to point an account to a mock server.
func (s *SqliteStore) UpdateAccountHostname(id int, hostname string) error {
	_, err := s.db.Exec("UPDATE accounts SET hostname = ? WHERE id = ?", hostname, id)
	return err
}

// UpdateAccountTags updates the tags for a given account.
func (s *SqliteStore) UpdateAccountTags(id int, tags string) error {
	_, err := s.db.Exec("UPDATE accounts SET tags = ? WHERE id = ?", tags, id)
	if err == nil {
		_ = s.LogAction("UPDATE_ACCOUNT_TAGS", fmt.Sprintf("account_id: %d, new_tags: '%s'", id, tags))
	}
	return err
}

// GetAllActiveAccounts retrieves all active accounts from the database.
func (s *SqliteStore) GetAllActiveAccounts() ([]model.Account, error) {
	rows, err := s.db.Query("SELECT id, username, hostname, label, tags, serial, is_active FROM accounts WHERE is_active = 1 ORDER BY label, hostname, username")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []model.Account
	for rows.Next() {
		var acc model.Account
		var label sql.NullString
		var tags sql.NullString
		if err := rows.Scan(&acc.ID, &acc.Username, &acc.Hostname, &label, &tags, &acc.Serial, &acc.IsActive); err != nil {
			return nil, err
		}
		if label.Valid {
			acc.Label = label.String
		}
		if tags.Valid {
			acc.Tags = tags.String
		}
		accounts = append(accounts, acc)
	}
	return accounts, nil
}

// AddPublicKey adds a new public key to the database.
func (s *SqliteStore) AddPublicKey(algorithm, keyData, comment string, isGlobal bool) error {
	_, err := s.db.Exec("INSERT INTO public_keys(algorithm, key_data, comment, is_global) VALUES(?, ?, ?, ?)", algorithm, keyData, comment, isGlobal)
	if err == nil {
		_ = s.LogAction("ADD_PUBLIC_KEY", fmt.Sprintf("comment: %s", comment))
	}
	return err
}

// GetAllPublicKeys retrieves all public keys from the database.
func (s *SqliteStore) GetAllPublicKeys() ([]model.PublicKey, error) {
	rows, err := s.db.Query("SELECT id, algorithm, key_data, comment, is_global FROM public_keys ORDER BY comment")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []model.PublicKey
	for rows.Next() {
		var key model.PublicKey
		if err := rows.Scan(&key.ID, &key.Algorithm, &key.KeyData, &key.Comment, &key.IsGlobal); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, nil
}

// GetPublicKeyByComment retrieves a single public key by its unique comment.
func (s *SqliteStore) GetPublicKeyByComment(comment string) (*model.PublicKey, error) {
	row := s.db.QueryRow("SELECT id, algorithm, key_data, comment, is_global FROM public_keys WHERE comment = ?", comment)
	var key model.PublicKey
	err := row.Scan(&key.ID, &key.Algorithm, &key.KeyData, &key.Comment, &key.IsGlobal)
	if err != nil {
		return nil, err // This will be sql.ErrNoRows if not found
	}
	return &key, nil
}

// AddPublicKeyAndGetModel adds a public key to the database if it doesn't already
// exist (based on the comment) and returns the full key model.
// It returns (nil, nil) if the key is a duplicate.
func (s *SqliteStore) AddPublicKeyAndGetModel(algorithm, keyData, comment string, isGlobal bool) (*model.PublicKey, error) {
	// First, check if it exists to avoid constraint errors.
	existing, err := s.GetPublicKeyByComment(comment)
	if err != nil && err != sql.ErrNoRows {
		return nil, err // A real DB error
	}
	if existing != nil {
		return nil, nil // Key already exists, return nil model and nil error
	}

	result, err := s.db.Exec("INSERT INTO public_keys (algorithm, key_data, comment, is_global) VALUES (?, ?, ?, ?)", algorithm, keyData, comment, isGlobal)
	if err != nil {
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	_ = s.LogAction("ADD_PUBLIC_KEY", fmt.Sprintf("comment: %s", comment))

	return &model.PublicKey{ID: int(id), Algorithm: algorithm, KeyData: keyData, Comment: comment, IsGlobal: isGlobal}, nil
}

// TogglePublicKeyGlobal flips the 'is_global' status of a public key.
func (s *SqliteStore) TogglePublicKeyGlobal(id int) error {
	_, err := s.db.Exec("UPDATE public_keys SET is_global = NOT is_global WHERE id = ?", id)
	if err == nil {
		_ = s.LogAction("TOGGLE_KEY_GLOBAL", fmt.Sprintf("key_id: %d", id))
	}
	return err
}

// GetGlobalPublicKeys retrieves all keys marked as global.
func (s *SqliteStore) GetGlobalPublicKeys() ([]model.PublicKey, error) {
	rows, err := s.db.Query("SELECT id, algorithm, key_data, comment, is_global FROM public_keys WHERE is_global = 1 ORDER BY comment")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []model.PublicKey
	for rows.Next() {
		var key model.PublicKey
		if err := rows.Scan(&key.ID, &key.Algorithm, &key.KeyData, &key.Comment, &key.IsGlobal); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, nil
}

// GetKnownHostKey retrieves the trusted public key for a given hostname.
func (s *SqliteStore) GetKnownHostKey(hostname string) (string, error) {
	var key string
	err := s.db.QueryRow("SELECT key FROM known_hosts WHERE hostname = ?", hostname).Scan(&key)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil // No key found is not an error, it's a state.
		}
		return "", err
	}
	return key, nil
}

// AddKnownHostKey adds a new trusted host key to the database.
func (s *SqliteStore) AddKnownHostKey(hostname, key string) error {
	// INSERT OR REPLACE will add the key if it doesn't exist, or update it if it does.
	// This is useful if a host is legitimately re-provisioned.
	_, err := s.db.Exec("INSERT OR REPLACE INTO known_hosts (hostname, key) VALUES (?, ?)", hostname, key)
	if err == nil {
		_ = s.LogAction("TRUST_HOST", fmt.Sprintf("hostname: %s", hostname))
	}
	return err
}

// CreateSystemKey adds a new system key to the database. It determines the correct serial automatically.
func (s *SqliteStore) CreateSystemKey(publicKey, privateKey string) (int, error) {
	var maxSerial sql.NullInt64
	err := s.db.QueryRow("SELECT MAX(serial) FROM system_keys").Scan(&maxSerial)
	if err != nil {
		return 0, err
	}

	newSerial := 1
	if maxSerial.Valid {
		newSerial = int(maxSerial.Int64) + 1
	}

	// In a real rotation, we would first set all other keys to inactive.
	// For initial generation, this is fine.
	_, err = s.db.Exec(
		"INSERT INTO system_keys(serial, public_key, private_key, is_active) VALUES(?, ?, ?, ?)",
		newSerial, publicKey, privateKey, true,
	)
	if err != nil {
		return 0, err
	}
	_ = s.LogAction("CREATE_SYSTEM_KEY", fmt.Sprintf("serial: %d", newSerial))

	return newSerial, nil
}

// RotateSystemKey deactivates all current system keys and adds a new one as active.
// This should be performed within a transaction to ensure atomicity.
func (s *SqliteStore) RotateSystemKey(publicKey, privateKey string) (int, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	// Defer a rollback in case anything fails.
	defer tx.Rollback()

	// Deactivate all existing keys.
	if _, err := tx.Exec("UPDATE system_keys SET is_active = 0"); err != nil {
		return 0, fmt.Errorf("failed to deactivate old system keys: %w", err)
	}

	// Get the next serial number.
	var maxSerial sql.NullInt64
	err = tx.QueryRow("SELECT MAX(serial) FROM system_keys").Scan(&maxSerial)
	if err != nil {
		return 0, err
	}
	newSerial := int(maxSerial.Int64) + 1

	// Insert the new active key.
	_, err = tx.Exec(
		"INSERT INTO system_keys(serial, public_key, private_key, is_active) VALUES(?, ?, ?, ?)",
		newSerial, publicKey, privateKey, true,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to insert new system key: %w", err)
	}

	// Commit the transaction.
	err = tx.Commit()
	if err == nil {
		_ = s.LogAction("ROTATE_SYSTEM_KEY", fmt.Sprintf("new_serial: %d", newSerial))
	}

	return newSerial, err
}

// GetActiveSystemKey retrieves the currently active system key for deployments.
func (s *SqliteStore) GetActiveSystemKey() (*model.SystemKey, error) {
	row := s.db.QueryRow("SELECT id, serial, public_key, private_key, is_active FROM system_keys WHERE is_active = 1")

	var key model.SystemKey
	err := row.Scan(&key.ID, &key.Serial, &key.PublicKey, &key.PrivateKey, &key.IsActive)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No active key found, not necessarily an error.
		}
		return nil, err
	}
	return &key, nil
}

// GetSystemKeyBySerial retrieves a system key by its serial number.
func (s *SqliteStore) GetSystemKeyBySerial(serial int) (*model.SystemKey, error) {
	row := s.db.QueryRow("SELECT id, serial, public_key, private_key, is_active FROM system_keys WHERE serial = ?", serial)

	var key model.SystemKey
	err := row.Scan(&key.ID, &key.Serial, &key.PublicKey, &key.PrivateKey, &key.IsActive)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No key found with that serial.
		}
		return nil, err
	}
	return &key, nil
}

// HasSystemKeys checks if any system keys exist in the database.
func (s *SqliteStore) HasSystemKeys() (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(id) FROM system_keys").Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// DeletePublicKey removes a public key and all its associations.
// The ON DELETE CASCADE constraint handles the associations in account_keys.
func (s *SqliteStore) DeletePublicKey(id int) error {
	// Get key comment before deleting for logging.
	var comment string
	err := s.db.QueryRow("SELECT comment FROM public_keys WHERE id = ?", id).Scan(&comment)
	details := fmt.Sprintf("id: %d", id)
	if err == nil {
		details = fmt.Sprintf("comment: %s", comment)
	}

	_, err = s.db.Exec("DELETE FROM public_keys WHERE id = ?", id)
	if err == nil {
		_ = s.LogAction("DELETE_PUBLIC_KEY", details)
	}
	return err
}

// AssignKeyToAccount creates an association between a key and an account.
func (s *SqliteStore) AssignKeyToAccount(keyID, accountID int) error {
	_, err := s.db.Exec("INSERT INTO account_keys(key_id, account_id) VALUES(?, ?)", keyID, accountID)
	if err == nil {
		// Get details for logging, ignoring errors as this is best-effort.
		var keyComment, accUser, accHost string
		_ = s.db.QueryRow("SELECT comment FROM public_keys WHERE id = ?", keyID).Scan(&keyComment)
		_ = s.db.QueryRow("SELECT username, hostname FROM accounts WHERE id = ?", accountID).Scan(&accUser, &accHost)
		details := fmt.Sprintf("key: '%s' to account: %s@%s", keyComment, accUser, accHost)
		_ = s.LogAction("ASSIGN_KEY", details)
	}
	return err
}

// UnassignKeyFromAccount removes an association between a key and an account.
func (s *SqliteStore) UnassignKeyFromAccount(keyID, accountID int) error {
	// Get details before unassigning for logging.
	var keyComment, accUser, accHost string
	_ = s.db.QueryRow("SELECT comment FROM public_keys WHERE id = ?", keyID).Scan(&keyComment)
	_ = s.db.QueryRow("SELECT username, hostname FROM accounts WHERE id = ?", accountID).Scan(&accUser, &accHost)
	details := fmt.Sprintf("key: '%s' from account: %s@%s", keyComment, accUser, accHost)

	_, err := s.db.Exec("DELETE FROM account_keys WHERE key_id = ? AND account_id = ?", keyID, accountID)
	if err == nil {
		_ = s.LogAction("UNASSIGN_KEY", details)
	}
	return err
}

// GetKeysForAccount retrieves all public keys assigned to a specific account.
func (s *SqliteStore) GetKeysForAccount(accountID int) ([]model.PublicKey, error) {
	query := `
		SELECT pk.id, pk.algorithm, pk.key_data, pk.comment
		FROM public_keys pk
		JOIN account_keys ak ON pk.id = ak.key_id
		WHERE ak.account_id = ?
		ORDER BY pk.comment`
	rows, err := s.db.Query(query, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []model.PublicKey
	for rows.Next() {
		var key model.PublicKey
		if err := rows.Scan(&key.ID, &key.Algorithm, &key.KeyData, &key.Comment); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, nil
}

// GetAccountsForKey retrieves all accounts that have a specific public key assigned.
func (s *SqliteStore) GetAccountsForKey(keyID int) ([]model.Account, error) {
	query := `
		SELECT a.id, a.username, a.hostname, a.label, a.tags, a.serial, a.is_active
		FROM accounts a
		JOIN account_keys ak ON a.id = ak.account_id
		WHERE ak.key_id = ?
		ORDER BY a.label, a.hostname, a.username`
	rows, err := s.db.Query(query, keyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []model.Account
	for rows.Next() {
		var acc model.Account
		var label sql.NullString
		var tags sql.NullString
		if err := rows.Scan(&acc.ID, &acc.Username, &acc.Hostname, &label, &tags, &acc.Serial, &acc.IsActive); err != nil {
			return nil, err
		}
		if label.Valid {
			acc.Label = label.String
		}
		if tags.Valid {
			acc.Tags = tags.String
		}
		accounts = append(accounts, acc)
	}
	return accounts, nil
}

// GetAllAuditLogEntries retrieves all entries from the audit log, most recent first.
func (s *SqliteStore) GetAllAuditLogEntries() ([]model.AuditLogEntry, error) {
	rows, err := s.db.Query("SELECT id, timestamp, username, action, details FROM audit_log ORDER BY timestamp DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []model.AuditLogEntry
	for rows.Next() {
		var entry model.AuditLogEntry
		if err := rows.Scan(&entry.ID, &entry.Timestamp, &entry.Username, &entry.Action, &entry.Details); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// LogAction records an audit trail event.
func (s *SqliteStore) LogAction(action string, details string) error {
	// Get current OS user
	currentUser, err := user.Current()
	username := "unknown"
	if err == nil {
		// On Windows, username might be "domain\user", let's just take the user part.
		if parts := strings.Split(currentUser.Username, `\`); len(parts) > 1 {
			username = parts[1]
		} else {
			username = currentUser.Username
		}
	}

	_, err = s.db.Exec("INSERT INTO audit_log (username, action, details) VALUES (?, ?, ?)", username, action, details)
	return err
}

// SaveBootstrapSession saves a bootstrap session to the database.
func (s *SqliteStore) SaveBootstrapSession(id, username, hostname, label, tags, tempPublicKey string, expiresAt time.Time, status string) error {
	_, err := s.db.Exec(`INSERT INTO bootstrap_sessions (id, username, hostname, label, tags, temp_public_key, expires_at, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, username, hostname, label, tags, tempPublicKey, expiresAt, status)
	return err
}

// GetBootstrapSession retrieves a bootstrap session by ID.
func (s *SqliteStore) GetBootstrapSession(id string) (*model.BootstrapSession, error) {
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
func (s *SqliteStore) DeleteBootstrapSession(id string) error {
	_, err := s.db.Exec("DELETE FROM bootstrap_sessions WHERE id = ?", id)
	return err
}

// UpdateBootstrapSessionStatus updates the status of a bootstrap session.
func (s *SqliteStore) UpdateBootstrapSessionStatus(id string, status string) error {
	_, err := s.db.Exec("UPDATE bootstrap_sessions SET status = ? WHERE id = ?", status, id)
	return err
}

// GetExpiredBootstrapSessions returns all expired bootstrap sessions.
func (s *SqliteStore) GetExpiredBootstrapSessions() ([]*model.BootstrapSession, error) {
	rows, err := s.db.Query(`SELECT id, username, hostname, label, tags, temp_public_key, created_at, expires_at, status
		FROM bootstrap_sessions WHERE expires_at < datetime('now')`)
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
func (s *SqliteStore) GetOrphanedBootstrapSessions() ([]*model.BootstrapSession, error) {
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
func (s *SqliteStore) ExportDataForBackup() (*model.BackupData, error) {
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
	rows, err = tx.Query("SELECT hostname, key FROM known_hosts")
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
