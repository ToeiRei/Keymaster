package db

import (
	"database/sql"
	"fmt"
	"os/user"
	"strings"
	"sync"

	"github.com/toeirei/keymaster/internal/model"
	_ "modernc.org/sqlite" // Pure Go SQLite driver
)

var (
	db   *sql.DB
	once sync.Once
	err  error
)

// InitDB initializes the database connection and creates tables if they don't exist.
// It's safe to call this multiple times.
func InitDB(dataSourceName string) (*sql.DB, error) {
	once.Do(func() {
		db, err = sql.Open("sqlite", dataSourceName)
		if err != nil {
			return
		}

		// Enable foreign key support, which is required for ON DELETE CASCADE.
		if _, err = db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
			return
		}

		accountsTableSQL := `
		CREATE TABLE IF NOT EXISTS accounts (
			id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL,
			hostname TEXT NOT NULL,
			label TEXT,
			serial INTEGER NOT NULL DEFAULT 0,
			is_active BOOLEAN NOT NULL DEFAULT 1,
			UNIQUE(username, hostname)
		);`
		if _, err = db.Exec(accountsTableSQL); err != nil {
			return
		}

		publicKeysTableSQL := `
		CREATE TABLE IF NOT EXISTS public_keys (
			id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			algorithm TEXT NOT NULL,
			key_data TEXT NOT NULL,
			comment TEXT NOT NULL UNIQUE
		);`
		if _, err = db.Exec(publicKeysTableSQL); err != nil {
			return
		}

		accountKeysTableSQL := `
		CREATE TABLE IF NOT EXISTS account_keys (
			account_id INTEGER NOT NULL,
			key_id INTEGER NOT NULL,
			PRIMARY KEY (account_id, key_id),
			FOREIGN KEY (account_id) REFERENCES accounts (id) ON DELETE CASCADE,
			FOREIGN KEY (key_id) REFERENCES public_keys (id) ON DELETE CASCADE
		);`
		if _, err = db.Exec(accountKeysTableSQL); err != nil {
			return
		}

		systemKeysTableSQL := `
		CREATE TABLE IF NOT EXISTS system_keys (
			id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			serial INTEGER NOT NULL UNIQUE,
			public_key TEXT NOT NULL,
			private_key TEXT NOT NULL,
			is_active BOOLEAN NOT NULL DEFAULT 0
		);`
		if _, err = db.Exec(systemKeysTableSQL); err != nil {
			return
		}

		knownHostsTableSQL := `
		CREATE TABLE IF NOT EXISTS known_hosts (
			hostname TEXT NOT NULL PRIMARY KEY,
			key TEXT NOT NULL
		);`
		if _, err = db.Exec(knownHostsTableSQL); err != nil {
			return
		}

		auditLogTableSQL := `
		CREATE TABLE IF NOT EXISTS audit_log (
			id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			timestamp DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			username TEXT NOT NULL,
			action TEXT NOT NULL,
			details TEXT
		);`
		if _, err = db.Exec(auditLogTableSQL); err != nil {
			return
		}

		// --- Simple Migration ---
		// This block ensures that older databases are updated with the new is_active column
		// without requiring the user to delete their database file.
		migrationErr := func() error {
			_, alterErr := db.Exec("ALTER TABLE accounts ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT 1;")
			if alterErr != nil {
				// If the error indicates the column already exists, we can safely ignore it.
				if strings.Contains(alterErr.Error(), "duplicate column name") {
					return nil
				}
				return alterErr
			}
			return nil
		}()
		if migrationErr != nil {
			err = migrationErr
			return
		}

		// Migration for the 'label' column
		labelMigrationErr := func() error {
			_, alterErr := db.Exec("ALTER TABLE accounts ADD COLUMN label TEXT;")
			if alterErr != nil {
				if strings.Contains(alterErr.Error(), "duplicate column name") {
					return nil
				}
				return alterErr
			}
			return nil
		}()
		if labelMigrationErr != nil {
			err = labelMigrationErr
			return
		}
	})

	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}
	return db, nil
}

// GetAllAccounts retrieves all accounts from the database.
func GetAllAccounts() ([]model.Account, error) {
	rows, err := db.Query("SELECT id, username, hostname, label, serial, is_active FROM accounts ORDER BY label, hostname, username")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []model.Account
	for rows.Next() {
		var acc model.Account
		var label sql.NullString
		if err := rows.Scan(&acc.ID, &acc.Username, &acc.Hostname, &label, &acc.Serial, &acc.IsActive); err != nil {
			return nil, err
		}
		if label.Valid {
			acc.Label = label.String
		}
		accounts = append(accounts, acc)
	}
	return accounts, nil
}

// AddAccount adds a new account to the database.
func AddAccount(username, hostname, label string) error {
	_, err := db.Exec("INSERT INTO accounts(username, hostname, label) VALUES(?, ?, ?)", username, hostname, label)
	if err == nil {
		_ = LogAction("ADD_ACCOUNT", fmt.Sprintf("account: %s@%s", username, hostname))
	}
	return err
}

// DeleteAccount removes an account from the database by its ID.
func DeleteAccount(id int) error {
	// Get account details before deleting for logging.
	var username, hostname string
	err := db.QueryRow("SELECT username, hostname FROM accounts WHERE id = ?", id).Scan(&username, &hostname)
	details := fmt.Sprintf("id: %d", id)
	if err == nil {
		details = fmt.Sprintf("account: %s@%s", username, hostname)
	}

	_, err = db.Exec("DELETE FROM accounts WHERE id = ?", id)
	if err == nil {
		_ = LogAction("DELETE_ACCOUNT", details)
	}
	return err
}

// UpdateAccountSerial sets the serial for a given account ID to a specific value.
func UpdateAccountSerial(id, serial int) error {
	_, err := db.Exec("UPDATE accounts SET serial = ? WHERE id = ?", serial, id)
	// This is called during deployment, which is logged at a higher level.
	// No need for a separate log action here.
	return err
}

// ToggleAccountStatus flips the active status of an account.
func ToggleAccountStatus(id int) error {
	// Get account details before toggling for logging.
	var username, hostname string
	var isActive bool
	err := db.QueryRow("SELECT username, hostname, is_active FROM accounts WHERE id = ?", id).Scan(&username, &hostname, &isActive)
	if err != nil {
		return err // If we can't find it, we can't toggle it.
	}

	_, err = db.Exec("UPDATE accounts SET is_active = NOT is_active WHERE id = ?", id)
	if err == nil {
		details := fmt.Sprintf("account: %s@%s, new_status: %t", username, hostname, !isActive)
		_ = LogAction("TOGGLE_ACCOUNT_STATUS", details)
	}
	return err
}

// GetAllActiveAccounts retrieves all active accounts from the database.
func GetAllActiveAccounts() ([]model.Account, error) {
	rows, err := db.Query("SELECT id, username, hostname, label, serial, is_active FROM accounts WHERE is_active = 1 ORDER BY label, hostname, username")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []model.Account
	for rows.Next() {
		var acc model.Account
		var label sql.NullString
		if err := rows.Scan(&acc.ID, &acc.Username, &acc.Hostname, &label, &acc.Serial, &acc.IsActive); err != nil {
			return nil, err
		}
		if label.Valid {
			acc.Label = label.String
		}
		accounts = append(accounts, acc)
	}
	return accounts, nil
}

// AddPublicKey adds a new public key to the database.
func AddPublicKey(algorithm, keyData, comment string) error {
	_, err := db.Exec("INSERT INTO public_keys(algorithm, key_data, comment) VALUES(?, ?, ?)", algorithm, keyData, comment)
	if err == nil {
		_ = LogAction("ADD_PUBLIC_KEY", fmt.Sprintf("comment: %s", comment))
	}
	return err
}

// GetAllPublicKeys retrieves all public keys from the database.
func GetAllPublicKeys() ([]model.PublicKey, error) {
	rows, err := db.Query("SELECT id, algorithm, key_data, comment FROM public_keys ORDER BY comment")
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

// GetPublicKeyByComment retrieves a single public key by its unique comment.
func GetPublicKeyByComment(comment string) (*model.PublicKey, error) {
	row := db.QueryRow("SELECT id, algorithm, key_data, comment FROM public_keys WHERE comment = ?", comment)
	var key model.PublicKey
	err := row.Scan(&key.ID, &key.Algorithm, &key.KeyData, &key.Comment)
	if err != nil {
		return nil, err // This will be sql.ErrNoRows if not found
	}
	return &key, nil
}

// AddPublicKeyAndGetModel adds a public key to the database if it doesn't already
// exist (based on the comment) and returns the full key model.
// It returns (nil, nil) if the key is a duplicate.
func AddPublicKeyAndGetModel(algorithm, keyData, comment string) (*model.PublicKey, error) {
	// First, check if it exists to avoid constraint errors.
	existing, err := GetPublicKeyByComment(comment)
	if err != nil && err != sql.ErrNoRows {
		return nil, err // A real DB error
	}
	if existing != nil {
		return nil, nil // Key already exists, return nil model and nil error
	}

	result, err := db.Exec("INSERT INTO public_keys (algorithm, key_data, comment) VALUES (?, ?, ?)", algorithm, keyData, comment)
	if err != nil {
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	_ = LogAction("ADD_PUBLIC_KEY", fmt.Sprintf("comment: %s", comment))

	return &model.PublicKey{ID: int(id), Algorithm: algorithm, KeyData: keyData, Comment: comment}, nil
}

// GetKnownHostKey retrieves the trusted public key for a given hostname.
func GetKnownHostKey(hostname string) (string, error) {
	var key string
	err := db.QueryRow("SELECT key FROM known_hosts WHERE hostname = ?", hostname).Scan(&key)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil // No key found is not an error, it's a state.
		}
		return "", err
	}
	return key, nil
}

// AddKnownHostKey adds a new trusted host key to the database.
func AddKnownHostKey(hostname, key string) error {
	// INSERT OR REPLACE will add the key if it doesn't exist, or update it if it does.
	// This is useful if a host is legitimately re-provisioned.
	_, err := db.Exec("INSERT OR REPLACE INTO known_hosts (hostname, key) VALUES (?, ?)", hostname, key)
	if err == nil {
		_ = LogAction("TRUST_HOST", fmt.Sprintf("hostname: %s", hostname))
	}
	return err
}

// CreateSystemKey adds a new system key to the database. It determines the correct serial automatically.
func CreateSystemKey(publicKey, privateKey string) (int, error) {
	var maxSerial sql.NullInt64
	err := db.QueryRow("SELECT MAX(serial) FROM system_keys").Scan(&maxSerial)
	if err != nil {
		return 0, err
	}

	newSerial := 1
	if maxSerial.Valid {
		newSerial = int(maxSerial.Int64) + 1
	}

	// In a real rotation, we would first set all other keys to inactive.
	// For initial generation, this is fine.
	_, err = db.Exec(
		"INSERT INTO system_keys(serial, public_key, private_key, is_active) VALUES(?, ?, ?, ?)",
		newSerial, publicKey, privateKey, true,
	)
	if err != nil {
		return 0, err
	}
	_ = LogAction("CREATE_SYSTEM_KEY", fmt.Sprintf("serial: %d", newSerial))

	return newSerial, nil
}

// RotateSystemKey deactivates all current system keys and adds a new one as active.
// This should be performed within a transaction to ensure atomicity.
func RotateSystemKey(publicKey, privateKey string) (int, error) {
	tx, err := db.Begin()
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
		_ = LogAction("ROTATE_SYSTEM_KEY", fmt.Sprintf("new_serial: %d", newSerial))
	}

	return newSerial, err
}

// GetActiveSystemKey retrieves the currently active system key for deployments.
func GetActiveSystemKey() (*model.SystemKey, error) {
	row := db.QueryRow("SELECT id, serial, public_key, private_key, is_active FROM system_keys WHERE is_active = 1")

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
func GetSystemKeyBySerial(serial int) (*model.SystemKey, error) {
	row := db.QueryRow("SELECT id, serial, public_key, private_key, is_active FROM system_keys WHERE serial = ?", serial)

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
func HasSystemKeys() (bool, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(id) FROM system_keys").Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// DeletePublicKey removes a public key and all its associations.
// The ON DELETE CASCADE constraint handles the associations in account_keys.
func DeletePublicKey(id int) error {
	// Get key comment before deleting for logging.
	var comment string
	err := db.QueryRow("SELECT comment FROM public_keys WHERE id = ?", id).Scan(&comment)
	details := fmt.Sprintf("id: %d", id)
	if err == nil {
		details = fmt.Sprintf("comment: %s", comment)
	}

	_, err = db.Exec("DELETE FROM public_keys WHERE id = ?", id)
	if err == nil {
		_ = LogAction("DELETE_PUBLIC_KEY", details)
	}
	return err
}

// AssignKeyToAccount creates an association between a key and an account.
func AssignKeyToAccount(keyID, accountID int) error {
	_, err := db.Exec("INSERT INTO account_keys(key_id, account_id) VALUES(?, ?)", keyID, accountID)
	if err == nil {
		// Get details for logging, ignoring errors as this is best-effort.
		var keyComment, accUser, accHost string
		_ = db.QueryRow("SELECT comment FROM public_keys WHERE id = ?", keyID).Scan(&keyComment)
		_ = db.QueryRow("SELECT username, hostname FROM accounts WHERE id = ?", accountID).Scan(&accUser, &accHost)
		details := fmt.Sprintf("key: '%s' to account: %s@%s", keyComment, accUser, accHost)
		_ = LogAction("ASSIGN_KEY", details)
	}
	return err
}

// UnassignKeyFromAccount removes an association between a key and an account.
func UnassignKeyFromAccount(keyID, accountID int) error {
	// Get details before unassigning for logging.
	var keyComment, accUser, accHost string
	_ = db.QueryRow("SELECT comment FROM public_keys WHERE id = ?", keyID).Scan(&keyComment)
	_ = db.QueryRow("SELECT username, hostname FROM accounts WHERE id = ?", accountID).Scan(&accUser, &accHost)
	details := fmt.Sprintf("key: '%s' from account: %s@%s", keyComment, accUser, accHost)

	_, err := db.Exec("DELETE FROM account_keys WHERE key_id = ? AND account_id = ?", keyID, accountID)
	if err == nil {
		_ = LogAction("UNASSIGN_KEY", details)
	}
	return err
}

// GetKeysForAccount retrieves all public keys assigned to a specific account.
func GetKeysForAccount(accountID int) ([]model.PublicKey, error) {
	query := `
		SELECT pk.id, pk.algorithm, pk.key_data, pk.comment
		FROM public_keys pk
		JOIN account_keys ak ON pk.id = ak.key_id
		WHERE ak.account_id = ?
		ORDER BY pk.comment`
	rows, err := db.Query(query, accountID)
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

// GetAllAuditLogEntries retrieves all entries from the audit log, most recent first.
func GetAllAuditLogEntries() ([]model.AuditLogEntry, error) {
	rows, err := db.Query("SELECT id, timestamp, username, action, details FROM audit_log ORDER BY timestamp DESC")
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
func LogAction(action string, details string) error {
	// Get current OS user
	currentUser, err := user.Current()
	username := "unknown"
	if err == nil {
		username = currentUser.Username
	}

	_, err = db.Exec("INSERT INTO audit_log (username, action, details) VALUES (?, ?, ?)", username, action, details)
	return err
}
