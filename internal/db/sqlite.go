package db

import (
	"database/sql"
	"fmt"
	"os/user"
	"strings"

	"github.com/toeirei/keymaster/internal/model"
	_ "modernc.org/sqlite" // Pure Go SQLite driver
)

// SqliteStore is the SQLite implementation of the Store interface.
type SqliteStore struct {
	db *sql.DB
}

// NewSqliteStore initializes the database connection and creates tables if they don't exist.
func NewSqliteStore(dataSourceName string) (*SqliteStore, error) {
	db, err := sql.Open("sqlite", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable foreign key support, which is required for ON DELETE CASCADE.
	if _, err = db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("database migration failed: %w", err)
	}

	return &SqliteStore{db: db}, nil
}

func runMigrations(db *sql.DB) error {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS accounts (
			id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL,
			hostname TEXT NOT NULL,
			label TEXT,
			tags TEXT,
			serial INTEGER NOT NULL DEFAULT 0,
			is_active BOOLEAN NOT NULL DEFAULT 1,
			UNIQUE(username, hostname)
		);`,
		`CREATE TABLE IF NOT EXISTS public_keys (
			id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			algorithm TEXT NOT NULL,
			key_data TEXT NOT NULL,
			comment TEXT NOT NULL UNIQUE,
			is_global BOOLEAN NOT NULL DEFAULT 0
		);`,
		`CREATE TABLE IF NOT EXISTS account_keys (
			account_id INTEGER NOT NULL,
			key_id INTEGER NOT NULL,
			PRIMARY KEY (account_id, key_id),
			FOREIGN KEY (account_id) REFERENCES accounts (id) ON DELETE CASCADE,
			FOREIGN KEY (key_id) REFERENCES public_keys (id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS system_keys (
			id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			serial INTEGER NOT NULL UNIQUE,
			public_key TEXT NOT NULL,
			private_key TEXT NOT NULL,
			is_active BOOLEAN NOT NULL DEFAULT 0
		);`,
		`CREATE TABLE IF NOT EXISTS known_hosts (
			hostname TEXT NOT NULL PRIMARY KEY,
			key TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS audit_log (
			id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			timestamp DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			username TEXT NOT NULL,
			action TEXT NOT NULL,
			details TEXT
		);`,
	}

	for _, tableSQL := range tables {
		if _, err := db.Exec(tableSQL); err != nil {
			return err
		}
	}

	// --- Simple Alter Table Migrations ---
	migrations := []string{
		"ALTER TABLE accounts ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT 1;",
		"ALTER TABLE accounts ADD COLUMN label TEXT;",
		"ALTER TABLE accounts ADD COLUMN tags TEXT;",
		"ALTER TABLE public_keys ADD COLUMN is_global BOOLEAN NOT NULL DEFAULT 0;",
	}

	for _, migrationSQL := range migrations {
		_, err := db.Exec(migrationSQL)
		if err != nil {
			// If the error indicates the column already exists, we can safely ignore it.
			if !strings.Contains(err.Error(), "duplicate column name") {
				return err
			}
		}
	}

	return nil
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
func (s *SqliteStore) AddAccount(username, hostname, label, tags string) error {
	_, err := s.db.Exec("INSERT INTO accounts(username, hostname, label, tags) VALUES(?, ?, ?, ?)", username, hostname, label, tags)
	if err == nil {
		_ = s.LogAction("ADD_ACCOUNT", fmt.Sprintf("account: %s@%s", username, hostname))
	}
	return err
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
