package db

import (
	"database/sql"
	"fmt"
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
	})

	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}
	return db, nil
}

// GetAllAccounts retrieves all accounts from the database.
func GetAllAccounts() ([]model.Account, error) {
	rows, err := db.Query("SELECT id, username, hostname, serial FROM accounts ORDER BY hostname, username")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []model.Account
	for rows.Next() {
		var acc model.Account
		if err := rows.Scan(&acc.ID, &acc.Username, &acc.Hostname, &acc.Serial); err != nil {
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

// IncrementAccountSerial increments the serial for a given account ID.
// This will be called after a successful deployment to an account's host.
func IncrementAccountSerial(id int) error {
	// We increment directly in SQL to avoid race conditions.
	_, err := db.Exec("UPDATE accounts SET serial = serial + 1 WHERE id = ?", id)
	return err
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
