package db

import (
	"database/sql"
	"fmt"
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
	})

	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}
	return db, nil
}

// GetAllAccounts retrieves all accounts from the database.
func GetAllAccounts() ([]model.Account, error) {
	rows, err := db.Query("SELECT id, username, hostname, serial, is_active FROM accounts ORDER BY hostname, username")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []model.Account
	for rows.Next() {
		var acc model.Account
		if err := rows.Scan(&acc.ID, &acc.Username, &acc.Hostname, &acc.Serial, &acc.IsActive); err != nil {
			return nil, err
		}
		accounts = append(accounts, acc)
	}
	return accounts, nil
}

// AddAccount adds a new account to the database.
func AddAccount(username, hostname string) error {
	_, err := db.Exec("INSERT INTO accounts(username, hostname) VALUES(?, ?)", username, hostname)
	// This will return an error if the UNIQUE constraint is violated, which is what we want.
	return err
}

// DeleteAccount removes an account from the database by its ID.
func DeleteAccount(id int) error {
	_, err := db.Exec("DELETE FROM accounts WHERE id = ?", id)
	return err
}

// UpdateAccountSerial sets the serial for a given account ID to a specific value.
func UpdateAccountSerial(id, serial int) error {
	_, err := db.Exec("UPDATE accounts SET serial = ? WHERE id = ?", serial, id)
	return err
}

// ToggleAccountStatus flips the active status of an account.
func ToggleAccountStatus(id int) error {
	// SQLite uses 0 and 1 for booleans. `NOT` works as expected.
	_, err := db.Exec("UPDATE accounts SET is_active = NOT is_active WHERE id = ?", id)
	return err
}

// GetAllActiveAccounts retrieves all active accounts from the database.
func GetAllActiveAccounts() ([]model.Account, error) {
	rows, err := db.Query("SELECT id, username, hostname, serial, is_active FROM accounts WHERE is_active = 1 ORDER BY hostname, username")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []model.Account
	for rows.Next() {
		var acc model.Account
		if err := rows.Scan(&acc.ID, &acc.Username, &acc.Hostname, &acc.Serial, &acc.IsActive); err != nil {
			return nil, err
		}
		accounts = append(accounts, acc)
	}
	return accounts, nil
}

// AddPublicKey adds a new public key to the database.
func AddPublicKey(algorithm, keyData, comment string) error {
	_, err := db.Exec("INSERT INTO public_keys(algorithm, key_data, comment) VALUES(?, ?, ?)", algorithm, keyData, comment)
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
	return newSerial, tx.Commit()
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
	_, err := db.Exec("DELETE FROM public_keys WHERE id = ?", id)
	return err
}

// AssignKeyToAccount creates an association between a key and an account.
func AssignKeyToAccount(keyID, accountID int) error {
	_, err := db.Exec("INSERT INTO account_keys(key_id, account_id) VALUES(?, ?)", keyID, accountID)
	return err
}

// UnassignKeyFromAccount removes an association between a key and an account.
func UnassignKeyFromAccount(keyID, accountID int) error {
	_, err := db.Exec("DELETE FROM account_keys WHERE key_id = ? AND account_id = ?", keyID, accountID)
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
