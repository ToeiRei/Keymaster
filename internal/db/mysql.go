package db

import (
	"database/sql"
	"fmt"

	"github.com/go-sql-driver/mysql" // MySQL driver
	"github.com/toeirei/keymaster/internal/model"
)

// MySQLStore is the MySQL implementation of the Store interface.
type MySQLStore struct {
	db *sql.DB
}

// NewMySQLStore initializes the database connection and creates tables if they don't exist.
func NewMySQLStore(dataSourceName string) (*MySQLStore, error) {
	// The MySQL driver requires a DSN format like: "user:password@tcp(host:port)/dbname"
	// It's good practice to add `?parseTime=true` to handle DATETIME columns correctly.
	db, err := sql.Open("mysql", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := runMySQLMigrations(db); err != nil {
		return nil, fmt.Errorf("database migration failed: %w", err)
	}

	return &MySQLStore{db: db}, nil
}

func runMySQLMigrations(db *sql.DB) error {
	// In MySQL, it's better to use VARCHAR for indexed columns and specify lengths.
	tables := []string{
		`CREATE TABLE IF NOT EXISTS accounts (
			id INTEGER NOT NULL PRIMARY KEY AUTO_INCREMENT,
			username VARCHAR(255) NOT NULL,
			hostname VARCHAR(255) NOT NULL,
			label VARCHAR(255),
			serial INTEGER NOT NULL DEFAULT 0,
			is_active BOOLEAN NOT NULL DEFAULT TRUE,
			UNIQUE(username, hostname)
		);`,
		`CREATE TABLE IF NOT EXISTS public_keys (
			id INTEGER NOT NULL PRIMARY KEY AUTO_INCREMENT,
			algorithm VARCHAR(255) NOT NULL,
			key_data TEXT NOT NULL,
			comment VARCHAR(255) NOT NULL UNIQUE
		);`,
		`CREATE TABLE IF NOT EXISTS account_keys (
			account_id INTEGER NOT NULL,
			key_id INTEGER NOT NULL,
			PRIMARY KEY (account_id, key_id),
			FOREIGN KEY (account_id) REFERENCES accounts (id) ON DELETE CASCADE,
			FOREIGN KEY (key_id) REFERENCES public_keys (id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS system_keys (
			id INTEGER NOT NULL PRIMARY KEY AUTO_INCREMENT,
			serial INTEGER NOT NULL UNIQUE,
			public_key TEXT NOT NULL,
			private_key TEXT NOT NULL,
			is_active BOOLEAN NOT NULL DEFAULT FALSE
		);`,
		`CREATE TABLE IF NOT EXISTS known_hosts (
			hostname VARCHAR(255) NOT NULL PRIMARY KEY,
			key TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS audit_log (
			id INTEGER NOT NULL PRIMARY KEY AUTO_INCREMENT,
			timestamp DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			username VARCHAR(255) NOT NULL,
			action VARCHAR(255) NOT NULL,
			details TEXT
		);`,
	}

	for _, tableSQL := range tables {
		if _, err := db.Exec(tableSQL); err != nil {
			return fmt.Errorf("failed to create table: %w, sql: %s", err, tableSQL)
		}
	}

	// Simple migrations, ignoring "duplicate column" errors.
	// MySQL does not support `ADD COLUMN IF NOT EXISTS`.
	migrations := []string{
		"ALTER TABLE accounts ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT TRUE;",
		"ALTER TABLE accounts ADD COLUMN label VARCHAR(255);",
	}

	for _, migrationSQL := range migrations {
		_, err := db.Exec(migrationSQL)
		if err != nil {
			// MySQL error number for duplicate column is 1060.
			if mysqlErr, ok := err.(*mysql.MySQLError); ok && mysqlErr.Number == 1060 {
				continue // Ignore duplicate column error
			}
			return err
		}
	}

	return nil
}

// --- Stubbed Methods ---

func (s *MySQLStore) GetAllAccounts() ([]model.Account, error) {
	return nil, fmt.Errorf("not implemented for mysql")
}
func (s *MySQLStore) AddAccount(username, hostname, label string) error {
	return fmt.Errorf("not implemented for mysql")
}
func (s *MySQLStore) DeleteAccount(id int) error {
	return fmt.Errorf("not implemented for mysql")
}
func (s *MySQLStore) UpdateAccountSerial(id, serial int) error {
	return fmt.Errorf("not implemented for mysql")
}
func (s *MySQLStore) ToggleAccountStatus(id int) error {
	return fmt.Errorf("not implemented for mysql")
}
func (s *MySQLStore) UpdateAccountLabel(id int, label string) error {
	return fmt.Errorf("not implemented for mysql")
}
func (s *MySQLStore) GetAllActiveAccounts() ([]model.Account, error) {
	return nil, fmt.Errorf("not implemented for mysql")
}
func (s *MySQLStore) AddPublicKey(algorithm, keyData, comment string) error {
	return fmt.Errorf("not implemented for mysql")
}
func (s *MySQLStore) GetAllPublicKeys() ([]model.PublicKey, error) {
	return nil, fmt.Errorf("not implemented for mysql")
}
func (s *MySQLStore) GetPublicKeyByComment(comment string) (*model.PublicKey, error) {
	return nil, fmt.Errorf("not implemented for mysql")
}
func (s *MySQLStore) AddPublicKeyAndGetModel(algorithm, keyData, comment string) (*model.PublicKey, error) {
	return nil, fmt.Errorf("not implemented for mysql")
}
func (s *MySQLStore) DeletePublicKey(id int) error {
	return fmt.Errorf("not implemented for mysql")
}
func (s *MySQLStore) GetKnownHostKey(hostname string) (string, error) {
	return "", fmt.Errorf("not implemented for mysql")
}
func (s *MySQLStore) AddKnownHostKey(hostname, key string) error {
	return fmt.Errorf("not implemented for mysql")
}
func (s *MySQLStore) CreateSystemKey(publicKey, privateKey string) (int, error) {
	return 0, fmt.Errorf("not implemented for mysql")
}
func (s *MySQLStore) RotateSystemKey(publicKey, privateKey string) (int, error) {
	return 0, fmt.Errorf("not implemented for mysql")
}
func (s *MySQLStore) GetActiveSystemKey() (*model.SystemKey, error) {
	return nil, fmt.Errorf("not implemented for mysql")
}
func (s *MySQLStore) GetSystemKeyBySerial(serial int) (*model.SystemKey, error) {
	return nil, fmt.Errorf("not implemented for mysql")
}
func (s *MySQLStore) HasSystemKeys() (bool, error) {
	return false, fmt.Errorf("not implemented for mysql")
}
func (s *MySQLStore) AssignKeyToAccount(keyID, accountID int) error {
	return fmt.Errorf("not implemented for mysql")
}
func (s *MySQLStore) UnassignKeyFromAccount(keyID, accountID int) error {
	return fmt.Errorf("not implemented for mysql")
}
func (s *MySQLStore) GetKeysForAccount(accountID int) ([]model.PublicKey, error) {
	return nil, fmt.Errorf("not implemented for mysql")
}
func (s *MySQLStore) GetAccountsForKey(keyID int) ([]model.Account, error) {
	return nil, fmt.Errorf("not implemented for mysql")
}
func (s *MySQLStore) GetAllAuditLogEntries() ([]model.AuditLogEntry, error) {
	return nil, fmt.Errorf("not implemented for mysql")
}
func (s *MySQLStore) LogAction(action string, details string) error {
	return fmt.Errorf("not implemented for mysql")
}
