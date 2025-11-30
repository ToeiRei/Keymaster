// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// package db provides the data access layer for Keymaster.
// It abstracts the underlying database (e.g., SQLite, PostgreSQL) behind a
// consistent interface, allowing the rest of the application to interact with
// the database in a uniform way.
package db // import "github.com/toeirei/keymaster/internal/db"

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/toeirei/keymaster/internal/model"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/mysqldialect"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	_ "modernc.org/sqlite"
)

// package-level variables
var (
	store Store
	//go:embed migrations
	embeddedMigrations embed.FS
)

// InitDB initializes the database connection based on the provided type and DSN.
// It sets the global `store` variable to the appropriate database implementation
// and runs any pending database migrations.
func InitDB(dbType, dsn string) error {
	s, err := NewStoreFromDSN(dbType, dsn)
	if err != nil {
		return fmt.Errorf("failed to initialize store: %w", err)
	}
	store = s
	return nil
}

// NewStoreFromDSN opens a sql.DB for the given DSN, runs migrations, and
// returns a Store backed by a long-lived *bun.DB. This hides *sql.DB usage
// from higher-level callers.
func NewStoreFromDSN(dbType, dsn string) (Store, error) {
	sqlDB, err := sql.Open(dbType, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := RunMigrations(sqlDB, dbType); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	switch dbType {
	case "sqlite":
		bunDB := bun.NewDB(sqlDB, sqlitedialect.New())
		return &SqliteStore{bun: bunDB}, nil
	case "postgres":
		bunDB := bun.NewDB(sqlDB, pgdialect.New())
		return &PostgresStore{bun: bunDB}, nil
	case "mysql":
		bunDB := bun.NewDB(sqlDB, mysqldialect.New())
		return &MySQLStore{bun: bunDB}, nil
	default:
		return nil, fmt.Errorf("unsupported database type for store creation: '%s'", dbType)
	}
}

// (old NewStore removed) Use NewStoreFromDSN to create stores from a DSN.

// RunMigrations applies the necessary database migrations for a given database connection.
func RunMigrations(db *sql.DB, dbType string) error {
	migrationsPath := fmt.Sprintf("migrations/%s", dbType)

	entries, err := fs.ReadDir(embeddedMigrations, migrationsPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// No migrations embedded for this DB type.
			return nil
		}
		return fmt.Errorf("failed to read embedded migrations (%s): %w", migrationsPath, err)
	}

	// Collect .up.sql files and sort them
	var ups []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".up.sql") {
			ups = append(ups, name)
		}
	}
	sort.Strings(ups)

	// Ensure schema_migrations table exists and is compatible with current schema
	if err := ensureSchemaMigrationsTable(db, dbType); err != nil {
		return fmt.Errorf("failed to ensure schema_migrations table: %w", err)
	}

	for _, fname := range ups {
		version := strings.TrimSuffix(fname, ".up.sql")

		// Check if already applied.
		var exists int
		query := "SELECT 1 FROM schema_migrations WHERE version = ?"
		if dbType == "postgres" {
			query = "SELECT 1 FROM schema_migrations WHERE version = $1"
		}
		err := db.QueryRow(query, version).Scan(&exists)
		if err == nil {
			// applied, skip
			continue
		}
		if err == sql.ErrNoRows {
			// not applied, continue to apply
		} else {
			return fmt.Errorf("failed to check migration version %s: %w", version, err)
		}

		// Read migration file contents
		path := path.Join(migrationsPath, fname)
		data, err := embeddedMigrations.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", path, err)
		}

		// Apply within a transaction
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction for migration %s: %w", version, err)
		}
		if _, err := tx.Exec(string(data)); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to execute migration %s: %w", version, err)
		}

		// Insert migration record; use DB-specific placeholder
		insertQuery := "INSERT INTO schema_migrations(version, applied_at) VALUES(?, ?)"
		if dbType == "postgres" {
			insertQuery = "INSERT INTO schema_migrations(version, applied_at) VALUES($1, $2)"
		}
		if _, err := tx.Exec(insertQuery, version, time.Now()); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %s: %w", version, err)
		}
		if err := tx.Commit(); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to commit migration %s: %w", version, err)
		}
	}
	return nil
}

// ensureSchemaMigrationsTable creates schema_migrations if missing and adds
// the `applied_at` column when the table exists but is missing that column.
func ensureSchemaMigrationsTable(db *sql.DB, dbType string) error {
	// create the table with the desired schema if it does not exist
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY, applied_at TIMESTAMP)`); err != nil {
		return err
	}

	// Check whether applied_at column exists; if not, add it.
	hasAppliedAt := false
	switch dbType {
	case "sqlite":
		rows, err := db.Query("PRAGMA table_info(schema_migrations)")
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			// cid, name, type, notnull, dflt_value, pk
			var cid int
			var name string
			var typ string
			var notnull int
			var dflt sql.NullString
			var pk int
			if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
				return err
			}
			if name == "applied_at" {
				hasAppliedAt = true
				break
			}
		}
	case "postgres", "mysql":
		// Use information_schema to detect column presence
		var query string
		if dbType == "postgres" {
			query = `SELECT column_name FROM information_schema.columns WHERE table_name='schema_migrations'`
		} else {
			query = `SELECT column_name FROM information_schema.columns WHERE table_name='schema_migrations' AND table_schema=DATABASE()`
		}
		rows, err := db.Query(query)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				return err
			}
			if name == "applied_at" {
				hasAppliedAt = true
				break
			}
		}
	default:
		// Unknown DB type; assume table is fine
		hasAppliedAt = true
	}

	if !hasAppliedAt {
		// Add the column. Different engines understand TIMESTAMP; MySQL accepts it too.
		alter := "ALTER TABLE schema_migrations ADD COLUMN applied_at TIMESTAMP"
		if dbType == "sqlite" {
			// SQLite supports ALTER TABLE ADD COLUMN
			// Use same statement
		}
		if _, err := db.Exec(alter); err != nil {
			return fmt.Errorf("failed to add applied_at column to schema_migrations: %w", err)
		}
	}
	return nil
}

// GetAllAccounts retrieves all accounts from the database.
func GetAllAccounts() ([]model.Account, error) {
	return store.GetAllAccounts()
}

// AddAccount adds a new account to the database.
func AddAccount(username, hostname, label, tags string) (int, error) {
	return store.AddAccount(username, hostname, label, tags)
}

// DeleteAccount removes an account from the database by its ID.
func DeleteAccount(id int) error {
	return store.DeleteAccount(id)
}

// UpdateAccountSerial sets the system key serial for a given account ID.
// This is typically called after a successful deployment.
func UpdateAccountSerial(id, serial int) error {
	return store.UpdateAccountSerial(id, serial)
}

// ToggleAccountStatus flips the active status of an account.
func ToggleAccountStatus(id int) error {
	return store.ToggleAccountStatus(id)
}

// UpdateAccountLabel updates the label for a given account.
func UpdateAccountLabel(id int, label string) error {
	return store.UpdateAccountLabel(id, label)
}

// UpdateAccountHostname updates the hostname for a given account.
func UpdateAccountHostname(id int, hostname string) error {
	return store.UpdateAccountHostname(id, hostname)
}

// UpdateAccountTags updates the tags for a given account.
func UpdateAccountTags(id int, tags string) error {
	return store.UpdateAccountTags(id, tags)
}

// GetAllActiveAccounts retrieves all active accounts from the database.
func GetAllActiveAccounts() ([]model.Account, error) {
	return store.GetAllActiveAccounts()
}

// AddPublicKey adds a new public key to the database.
func AddPublicKey(algorithm, keyData, comment string, isGlobal bool) error {
	return store.AddPublicKey(algorithm, keyData, comment, isGlobal)
}

// GetAllPublicKeys retrieves all public keys from the database.
func GetAllPublicKeys() ([]model.PublicKey, error) {
	return store.GetAllPublicKeys()
}

// GetPublicKeyByComment retrieves a single public key by its unique comment.
func GetPublicKeyByComment(comment string) (*model.PublicKey, error) {
	return store.GetPublicKeyByComment(comment)
}

// AddPublicKeyAndGetModel adds a public key to the database if it doesn't already
// exist (based on the comment) and returns the full key model. If a key with
// the same comment already exists, it returns (nil, nil) to indicate a
// duplicate without an error.
func AddPublicKeyAndGetModel(algorithm, keyData, comment string, isGlobal bool) (*model.PublicKey, error) { //
	return store.AddPublicKeyAndGetModel(algorithm, keyData, comment, isGlobal)
}

// TogglePublicKeyGlobal flips the 'is_global' status of a public key.
func TogglePublicKeyGlobal(id int) error {
	return store.TogglePublicKeyGlobal(id)
}

// GetGlobalPublicKeys retrieves all keys marked as global.
func GetGlobalPublicKeys() ([]model.PublicKey, error) {
	return store.GetGlobalPublicKeys()
}

// GetKnownHostKey retrieves the trusted public key for a given hostname.
func GetKnownHostKey(hostname string) (string, error) {
	return store.GetKnownHostKey(hostname)
}

// AddKnownHostKey adds a new trusted host key to the database.
func AddKnownHostKey(hostname, key string) error {
	return store.AddKnownHostKey(hostname, key)
}

// CreateSystemKey adds a new system key to the database. It determines the correct serial automatically.
func CreateSystemKey(publicKey, privateKey string) (int, error) {
	return store.CreateSystemKey(publicKey, privateKey)
}

// RotateSystemKey deactivates all current system keys and adds a new one as active.
// This should be performed within a transaction to ensure atomicity.
func RotateSystemKey(publicKey, privateKey string) (int, error) {
	return store.RotateSystemKey(publicKey, privateKey)
}

// GetActiveSystemKey retrieves the currently active system key for deployments.
func GetActiveSystemKey() (*model.SystemKey, error) {
	return store.GetActiveSystemKey()
}

// GetSystemKeyBySerial retrieves a system key by its serial number.
func GetSystemKeyBySerial(serial int) (*model.SystemKey, error) {
	return store.GetSystemKeyBySerial(serial)
}

// HasSystemKeys checks if any system keys exist in the database.
func HasSystemKeys() (bool, error) {
	return store.HasSystemKeys()
}

// DeletePublicKey removes a public key and all its associations.
// The ON DELETE CASCADE constraint handles the associations in account_keys.
func DeletePublicKey(id int) error {
	return store.DeletePublicKey(id)
}

// AssignKeyToAccount creates an association between a key and an account.
func AssignKeyToAccount(keyID, accountID int) error {
	return store.AssignKeyToAccount(keyID, accountID)
}

// UnassignKeyFromAccount removes an association between a key and an account.
func UnassignKeyFromAccount(keyID, accountID int) error {
	return store.UnassignKeyFromAccount(keyID, accountID)
}

// GetKeysForAccount retrieves all public keys assigned to a specific account.
func GetKeysForAccount(accountID int) ([]model.PublicKey, error) {
	return store.GetKeysForAccount(accountID)
}

// GetAccountsForKey retrieves all accounts that have a specific public key assigned.
func GetAccountsForKey(keyID int) ([]model.Account, error) {
	return store.GetAccountsForKey(keyID)
}

// GetAllAuditLogEntries retrieves all entries from the audit log, most recent first.
func GetAllAuditLogEntries() ([]model.AuditLogEntry, error) {
	return store.GetAllAuditLogEntries()
}

// LogAction records an audit trail event.
func LogAction(action string, details string) error {
	return store.LogAction(action, details)
}

// SaveBootstrapSession saves a bootstrap session to the database.
func SaveBootstrapSession(id, username, hostname, label, tags, tempPublicKey string, expiresAt time.Time, status string) error {
	return store.SaveBootstrapSession(id, username, hostname, label, tags, tempPublicKey, expiresAt, status)
}

// GetBootstrapSession retrieves a bootstrap session by ID.
func GetBootstrapSession(id string) (*model.BootstrapSession, error) {
	return store.GetBootstrapSession(id)
}

// DeleteBootstrapSession removes a bootstrap session from the database.
func DeleteBootstrapSession(id string) error {
	return store.DeleteBootstrapSession(id)
}

// UpdateBootstrapSessionStatus updates the status of a bootstrap session.
func UpdateBootstrapSessionStatus(id string, status string) error {
	return store.UpdateBootstrapSessionStatus(id, status)
}

// GetExpiredBootstrapSessions returns all expired bootstrap sessions.
func GetExpiredBootstrapSessions() ([]*model.BootstrapSession, error) {
	return store.GetExpiredBootstrapSessions()
}

// GetOrphanedBootstrapSessions returns all orphaned bootstrap sessions.
func GetOrphanedBootstrapSessions() ([]*model.BootstrapSession, error) {
	return store.GetOrphanedBootstrapSessions()
}

// ExportDataForBackup retrieves all data from the database for a backup.
func ExportDataForBackup() (*model.BackupData, error) {
	return store.ExportDataForBackup()
}

// ImportDataFromBackup restores the database from a backup data structure.
func ImportDataFromBackup(backup *model.BackupData) error {
	return store.ImportDataFromBackup(backup)
}

// IntegrateDataFromBackup restores the database from a backup data structure in a non-destructive way.
func IntegrateDataFromBackup(backup *model.BackupData) error {
	return store.IntegrateDataFromBackup(backup)
}
