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
	"os/user"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // PostgreSQL driver
	"github.com/toeirei/keymaster/internal/model"
)

// PostgresStore is the PostgreSQL implementation of the Store interface.
type PostgresStore struct {
	db *sql.DB
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

func (s *PostgresStore) AddAccount(username, hostname, label, tags string) (int, error) {
	// Postgres uses $1, $2, etc. for placeholders.
	var id int
	err := s.db.QueryRow("INSERT INTO accounts(username, hostname, label, tags) VALUES($1, $2, $3, $4) RETURNING id", username, hostname, label, tags).Scan(&id)
	if err != nil {
		return 0, err
	}
	_ = s.LogAction("ADD_ACCOUNT", fmt.Sprintf("account: %s@%s", username, hostname))
	return id, nil
}

func (s *PostgresStore) DeleteAccount(id int) error {
	// Get account details before deleting for logging.
	var username, hostname string
	err := s.db.QueryRow("SELECT username, hostname FROM accounts WHERE id = $1", id).Scan(&username, &hostname)
	details := fmt.Sprintf("id: %d", id)
	if err == nil {
		details = fmt.Sprintf("account: %s@%s", username, hostname)
	}

	_, err = s.db.Exec("DELETE FROM accounts WHERE id = $1", id)
	if err == nil {
		_ = s.LogAction("DELETE_ACCOUNT", details)
	}
	return err
}

func (s *PostgresStore) UpdateAccountSerial(id, serial int) error {
	_, err := s.db.Exec("UPDATE accounts SET serial = $1 WHERE id = $2", serial, id)
	// This is called during deployment, which is logged at a higher level.
	// No need for a separate log action here.
	return err
}

func (s *PostgresStore) ToggleAccountStatus(id int) error {
	// Get account details before toggling for logging.
	var username, hostname string
	var isActive bool
	err := s.db.QueryRow("SELECT username, hostname, is_active FROM accounts WHERE id = $1", id).Scan(&username, &hostname, &isActive)
	if err != nil {
		return err // If we can't find it, we can't toggle it.
	}

	_, err = s.db.Exec("UPDATE accounts SET is_active = NOT is_active WHERE id = $1", id)
	if err == nil {
		details := fmt.Sprintf("account: %s@%s, new_status: %t", username, hostname, !isActive)
		_ = s.LogAction("TOGGLE_ACCOUNT_STATUS", details)
	}
	return err
}

func (s *PostgresStore) UpdateAccountLabel(id int, label string) error {
	_, err := s.db.Exec("UPDATE accounts SET label = $1 WHERE id = $2", label, id)
	if err == nil {
		_ = s.LogAction("UPDATE_ACCOUNT_LABEL", fmt.Sprintf("account_id: %d, new_label: '%s'", id, label))
	}
	return err
}

func (s *PostgresStore) UpdateAccountHostname(id int, hostname string) error {
	// This is primarily used for testing to point an account to a mock server.
	_, err := s.db.Exec("UPDATE accounts SET hostname = $1 WHERE id = $2", hostname, id)
	return err
}

func (s *PostgresStore) UpdateAccountTags(id int, tags string) error {
	_, err := s.db.Exec("UPDATE accounts SET tags = $1 WHERE id = $2", tags, id)
	if err == nil {
		_ = s.LogAction("UPDATE_ACCOUNT_TAGS", fmt.Sprintf("account_id: %d, new_tags: '%s'", id, tags))
	}
	return err
}

func (s *PostgresStore) GetAllActiveAccounts() ([]model.Account, error) {
	rows, err := s.db.Query("SELECT id, username, hostname, label, tags, serial, is_active FROM accounts WHERE is_active = TRUE ORDER BY label, hostname, username")
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

func (s *PostgresStore) AddPublicKey(algorithm, keyData, comment string, isGlobal bool) error {
	_, err := s.db.Exec("INSERT INTO public_keys(algorithm, key_data, comment, is_global) VALUES($1, $2, $3, $4)", algorithm, keyData, comment, isGlobal)
	if err == nil {
		_ = s.LogAction("ADD_PUBLIC_KEY", fmt.Sprintf("comment: %s", comment))
	}
	return err
}

func (s *PostgresStore) GetAllPublicKeys() ([]model.PublicKey, error) {
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

func (s *PostgresStore) GetPublicKeyByComment(comment string) (*model.PublicKey, error) {
	row := s.db.QueryRow("SELECT id, algorithm, key_data, comment, is_global FROM public_keys WHERE comment = $1", comment)
	var key model.PublicKey
	err := row.Scan(&key.ID, &key.Algorithm, &key.KeyData, &key.Comment, &key.IsGlobal)
	if err != nil {
		return nil, err // This will be sql.ErrNoRows if not found
	}
	return &key, nil
}

func (s *PostgresStore) AddPublicKeyAndGetModel(algorithm, keyData, comment string, isGlobal bool) (*model.PublicKey, error) {
	// First, check if it exists to avoid constraint errors.
	existing, err := s.GetPublicKeyByComment(comment)
	if err != nil && err != sql.ErrNoRows {
		return nil, err // A real DB error
	}
	if existing != nil {
		return nil, nil // Key already exists, return nil model and nil error
	}

	var id int
	err = s.db.QueryRow(
		"INSERT INTO public_keys (algorithm, key_data, comment, is_global) VALUES ($1, $2, $3, $4) RETURNING id",
		algorithm, keyData, comment, isGlobal,
	).Scan(&id)
	if err != nil {
		return nil, err
	}
	_ = s.LogAction("ADD_PUBLIC_KEY", fmt.Sprintf("comment: %s", comment))

	return &model.PublicKey{ID: int(id), Algorithm: algorithm, KeyData: keyData, Comment: comment, IsGlobal: isGlobal}, nil
}

func (s *PostgresStore) TogglePublicKeyGlobal(id int) error {
	_, err := s.db.Exec("UPDATE public_keys SET is_global = NOT is_global WHERE id = $1", id)
	if err == nil {
		_ = s.LogAction("TOGGLE_KEY_GLOBAL", fmt.Sprintf("key_id: %d", id))
	}
	return err
}

func (s *PostgresStore) GetGlobalPublicKeys() ([]model.PublicKey, error) {
	rows, err := s.db.Query("SELECT id, algorithm, key_data, comment, is_global FROM public_keys WHERE is_global = TRUE ORDER BY comment")
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

func (s *PostgresStore) DeletePublicKey(id int) error {
	// Get key comment before deleting for logging.
	var comment string
	err := s.db.QueryRow("SELECT comment FROM public_keys WHERE id = $1", id).Scan(&comment)
	details := fmt.Sprintf("id: %d", id)
	if err == nil {
		details = fmt.Sprintf("comment: %s", comment)
	}

	_, err = s.db.Exec("DELETE FROM public_keys WHERE id = $1", id)
	if err == nil {
		_ = s.LogAction("DELETE_PUBLIC_KEY", details)
	}
	return err
}
func (s *PostgresStore) GetKnownHostKey(hostname string) (string, error) {
	var key string
	// Note the double quotes around "key" because it's a reserved keyword in some contexts.
	err := s.db.QueryRow(`SELECT "key" FROM known_hosts WHERE hostname = $1`, hostname).Scan(&key)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil // No key found is not an error, it's a state.
		}
		return "", err
	}
	return key, nil
}

func (s *PostgresStore) AddKnownHostKey(hostname, key string) error {
	// Use Postgres's ON CONFLICT for "UPSERT" behavior.
	_, err := s.db.Exec(`
		INSERT INTO known_hosts (hostname, "key") VALUES ($1, $2)
		ON CONFLICT (hostname) DO UPDATE SET "key" = EXCLUDED.key`,
		hostname, key)

	if err == nil {
		_ = s.LogAction("TRUST_HOST", fmt.Sprintf("hostname: %s", hostname))
	}
	return err
}

func (s *PostgresStore) CreateSystemKey(publicKey, privateKey string) (int, error) {
	var maxSerial sql.NullInt64
	err := s.db.QueryRow("SELECT MAX(serial) FROM system_keys").Scan(&maxSerial)
	if err != nil && err != sql.ErrNoRows {
		return 0, err
	}

	newSerial := 1
	if maxSerial.Valid {
		newSerial = int(maxSerial.Int64) + 1
	}

	_, err = s.db.Exec(
		"INSERT INTO system_keys(serial, public_key, private_key, is_active) VALUES($1, $2, $3, $4)",
		newSerial, publicKey, privateKey, true,
	)
	if err != nil {
		return 0, err
	}
	_ = s.LogAction("CREATE_SYSTEM_KEY", fmt.Sprintf("serial: %d", newSerial))

	return newSerial, nil
}

func (s *PostgresStore) RotateSystemKey(publicKey, privateKey string) (int, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("UPDATE system_keys SET is_active = FALSE"); err != nil {
		return 0, fmt.Errorf("failed to deactivate old system keys: %w", err)
	}

	var maxSerial sql.NullInt64
	err = tx.QueryRow("SELECT MAX(serial) FROM system_keys").Scan(&maxSerial)
	if err != nil && err != sql.ErrNoRows {
		return 0, err
	}
	newSerial := int(maxSerial.Int64) + 1

	_, err = tx.Exec(
		"INSERT INTO system_keys(serial, public_key, private_key, is_active) VALUES($1, $2, $3, $4)",
		newSerial, publicKey, privateKey, true,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to insert new system key: %w", err)
	}

	err = tx.Commit()
	if err == nil {
		_ = s.LogAction("ROTATE_SYSTEM_KEY", fmt.Sprintf("new_serial: %d", newSerial))
	}

	return newSerial, err
}

func (s *PostgresStore) GetActiveSystemKey() (*model.SystemKey, error) {
	row := s.db.QueryRow("SELECT id, serial, public_key, private_key, is_active FROM system_keys WHERE is_active = TRUE")

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

func (s *PostgresStore) GetSystemKeyBySerial(serial int) (*model.SystemKey, error) {
	row := s.db.QueryRow("SELECT id, serial, public_key, private_key, is_active FROM system_keys WHERE serial = $1", serial)

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

func (s *PostgresStore) HasSystemKeys() (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(id) FROM system_keys").Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *PostgresStore) AssignKeyToAccount(keyID, accountID int) error {
	_, err := s.db.Exec("INSERT INTO account_keys(key_id, account_id) VALUES($1, $2)", keyID, accountID)
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

	_, err := s.db.Exec("DELETE FROM account_keys WHERE key_id = $1 AND account_id = $2", keyID, accountID)
	if err == nil {
		_ = s.LogAction("UNASSIGN_KEY", details)
	}
	return err
}

func (s *PostgresStore) GetKeysForAccount(accountID int) ([]model.PublicKey, error) {
	query := `
		SELECT pk.id, pk.algorithm, pk.key_data, pk.comment
		FROM public_keys pk
		JOIN account_keys ak ON pk.id = ak.key_id
		WHERE ak.account_id = $1
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

func (s *PostgresStore) GetAccountsForKey(keyID int) ([]model.Account, error) {
	query := `
		SELECT a.id, a.username, a.hostname, a.label, a.tags, a.serial, a.is_active
		FROM accounts a
		JOIN account_keys ak ON a.id = ak.account_id
		WHERE ak.key_id = $1
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

func (s *PostgresStore) GetAllAuditLogEntries() ([]model.AuditLogEntry, error) {
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

func (s *PostgresStore) LogAction(action string, details string) error {
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

	_, err = s.db.Exec("INSERT INTO audit_log (username, action, details) VALUES ($1, $2, $3)", username, action, details)
	return err
}

// SaveBootstrapSession saves a bootstrap session to the database.
func (s *PostgresStore) SaveBootstrapSession(id, username, hostname, label, tags, tempPublicKey string, expiresAt time.Time, status string) error {
	_, err := s.db.Exec(`INSERT INTO bootstrap_sessions (id, username, hostname, label, tags, temp_public_key, expires_at, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		id, username, hostname, label, tags, tempPublicKey, expiresAt, status)
	return err
}

// GetBootstrapSession retrieves a bootstrap session by ID.
func (s *PostgresStore) GetBootstrapSession(id string) (*model.BootstrapSession, error) {
	var session model.BootstrapSession
	var label, tags sql.NullString

	err := s.db.QueryRow(`SELECT id, username, hostname, label, tags, temp_public_key, created_at, expires_at, status
		FROM bootstrap_sessions WHERE id = $1`, id).Scan(
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
func (s *PostgresStore) DeleteBootstrapSession(id string) error {
	_, err := s.db.Exec("DELETE FROM bootstrap_sessions WHERE id = $1", id)
	return err
}

// UpdateBootstrapSessionStatus updates the status of a bootstrap session.
func (s *PostgresStore) UpdateBootstrapSessionStatus(id string, status string) error {
	_, err := s.db.Exec("UPDATE bootstrap_sessions SET status = $1 WHERE id = $2", status, id)
	return err
}

// GetExpiredBootstrapSessions returns all expired bootstrap sessions.
func (s *PostgresStore) GetExpiredBootstrapSessions() ([]*model.BootstrapSession, error) {
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
func (s *PostgresStore) GetOrphanedBootstrapSessions() ([]*model.BootstrapSession, error) {
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
func (s *PostgresStore) ExportDataForBackup() (*model.BackupData, error) {
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
	rows, err = tx.Query(`SELECT hostname, "key" FROM known_hosts`)
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
func (s *PostgresStore) ImportDataFromBackup(backup *model.BackupData) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Rollback on any error.

	// --- 1. DB Wipe ---
	// TRUNCATE is faster than DELETE and also resets auto-incrementing counters.
	// RESTART IDENTITY resets sequences, and CASCADE handles foreign keys automatically.
	tables := []string{
		"accounts",
		"public_keys",
		"system_keys",
		"known_hosts",
		"audit_log",
		"bootstrap_sessions",
	}
	truncateQuery := fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE", strings.Join(tables, ", "))
	if _, err := tx.Exec(truncateQuery); err != nil {
		return fmt.Errorf("failed to wipe tables: %w", err)
	}

	// --- 2. DB Integration (Insertion) ---

	// Accounts
	stmt, err := tx.Prepare("INSERT INTO accounts (id, username, hostname, label, tags, serial, is_active) VALUES ($1, $2, $3, $4, $5, $6, $7)")
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
	stmt, err = tx.Prepare("INSERT INTO public_keys (id, algorithm, key_data, comment, is_global) VALUES ($1, $2, $3, $4, $5)")
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
	stmt, err = tx.Prepare("INSERT INTO account_keys (key_id, account_id) VALUES ($1, $2)")
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
	stmt, err = tx.Prepare("INSERT INTO system_keys (id, serial, public_key, private_key, is_active) VALUES ($1, $2, $3, $4, $5)")
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
	stmt, err = tx.Prepare(`INSERT INTO known_hosts (hostname, "key") VALUES ($1, $2)`)
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
	stmt, err = tx.Prepare("INSERT INTO audit_log (id, timestamp, username, action, details) VALUES ($1, $2, $3, $4, $5)")
	if err != nil {
		return fmt.Errorf("failed to prepare audit_log insert: %w", err)
	}
	for _, ale := range backup.AuditLogEntries {
		if _, err := stmt.Exec(ale.ID, ale.Timestamp, ale.Username, ale.Action, ale.Details); err != nil {
			return fmt.Errorf("failed to insert audit log %d: %w", ale.ID, err)
		}
	}
	stmt.Close()

	// After all data is inserted, we need to manually update the sequences for the tables
	// where we inserted explicit IDs, so that new inserts don't have conflicting IDs.
	if _, err := tx.Exec("SELECT setval('accounts_id_seq', (SELECT MAX(id) FROM accounts))"); err != nil {
		return fmt.Errorf("failed to update accounts sequence: %w", err)
	}
	if _, err := tx.Exec("SELECT setval('public_keys_id_seq', (SELECT MAX(id) FROM public_keys))"); err != nil {
		return fmt.Errorf("failed to update public_keys sequence: %w", err)
	}

	return tx.Commit()
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
