package db

import (
	"database/sql"
	"fmt"
	"sync"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
	"github.com/toeirei/keymaster/internal/model"
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
		db, err = sql.Open("sqlite3", dataSourceName)
		if err != nil {
			return
		}

		createTableSQL := `
		CREATE TABLE IF NOT EXISTS accounts (
			id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL,
			hostname TEXT NOT NULL,
			serial INTEGER NOT NULL DEFAULT 0,
			UNIQUE(username, hostname)
		);`
		// TODO: Add tables for keys and associations later.

		_, err = db.Exec(createTableSQL)
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
