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

	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"context" // Added context import for maintenance timeouts

	"github.com/toeirei/keymaster/internal/model"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/mysqldialect"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	_ "modernc.org/sqlite"

	// SQL drivers required for integration tests and runtime.
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// package-level variables
var (
	store Store
	//go:embed migrations
	embeddedMigrations embed.FS
	// sqlOpenFunc allows tests to override database opening behavior.
	sqlOpenFunc = sql.Open
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

// IsInitialized reports whether the package-level store has been set.
func IsInitialized() bool {
	return store != nil
}

// RunDBMaintenance performs engine-specific maintenance tasks for the given
// database DSN. It is safe to call for SQLite/Postgres/MySQL. For SQLite this
// will run PRAGMA optimize, VACUUM and WAL checkpoint. For Postgres it runs
// VACUUM ANALYZE. For MySQL it runs OPTIMIZE TABLE for all tables.
func RunDBMaintenance(dbType, dsn string) error {
	driverName := dbType
	if dbType == "postgres" {
		driverName = "pgx"
	}
	sqlDB, err := sqlOpenFunc(driverName, dsn)
	if err != nil {
		return fmt.Errorf("failed to open database for maintenance: %w", err)
	}
	defer func() { _ = sqlDB.Close() }()

	// Small timeout for maintenance operations to avoid blocking CI.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	switch dbType {
	case "sqlite":
		// Run PRAGMA optimize, VACUUM, and checkpoint WAL (if present).
		// PRAGMA optimize may not be supported or useful in some environments
		// (e.g., in-memory filesystems); treat optimize errors as non-fatal.
		if _, err := sqlDB.ExecContext(ctx, "PRAGMA optimize;"); err != nil {
			dbLogf("db: sqlite optimize failed (ignored): %v", err)
		}
		if _, err := sqlDB.ExecContext(ctx, "VACUUM;"); err != nil {
			return fmt.Errorf("sqlite vacuum failed: %w", err)
		}
		// WAL checkpoint; ignore errors if not supported.
		_, _ = sqlDB.ExecContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE);")
		// Optionally run integrity_check and return error if database is corrupt.
		var res string
		if row := sqlDB.QueryRowContext(ctx, "PRAGMA integrity_check;"); row != nil {
			_ = row.Scan(&res)
			if res != "ok" {
				return fmt.Errorf("sqlite integrity_check failed: %s", res)
			}
		}
	case "postgres":
		// VACUUM ANALYZE
		if _, err := sqlDB.ExecContext(ctx, "VACUUM ANALYZE;"); err != nil {
			return fmt.Errorf("postgres vacuum failed: %w", err)
		}
	case "mysql":
		// Get list of tables and run OPTIMIZE TABLE
		rows, err := sqlDB.QueryContext(ctx, "SHOW TABLES")
		if err != nil {
			return fmt.Errorf("mysql show tables failed: %w", err)
		}
		defer func() { _ = rows.Close() }()
		var table string
		var lastErr error
		for rows.Next() {
			if err := rows.Scan(&table); err != nil {
				return fmt.Errorf("mysql read table name failed: %w", err)
			}
			if _, err := sqlDB.ExecContext(ctx, fmt.Sprintf("OPTIMIZE TABLE %s", table)); err != nil {
				// Non-fatal per-table: remember last error and continue
				dbLogf("db: mysql optimize table %s failed: %v", table, err)
				lastErr = err
			}
		}
		if lastErr != nil {
			return fmt.Errorf("mysql optimize encountered errors: %w", lastErr)
		}
	default:
		return fmt.Errorf("unsupported db type for maintenance: %s", dbType)
	}
	return nil
}

// NewStoreFromDSN opens a sql.DB for the given DSN, runs migrations, and
// returns a Store backed by a long-lived *bun.DB. This hides *sql.DB usage
// from higher-level callers.
func NewStoreFromDSN(dbType, dsn string) (Store, error) {
	driverName := dbType
	// The pgx stdlib registers driver name "pgx"; map "postgres" to that driver.
	if dbType == "postgres" {
		driverName = "pgx"
	}
	start := time.Now()
	sqlDB, err := sqlOpenFunc(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure DB connection pool with sensible defaults. Values can be
	// overridden via environment variables for CI or production tuning.
	// Defaults chosen to be conservative for small deployments.
	const (
		defaultMaxOpenConns    = 25
		defaultMaxIdleConns    = 25
		defaultConnMaxLifetime = 5 * time.Minute
	)

	maxOpen := defaultMaxOpenConns
	if v := os.Getenv("KEYMASTER_DB_MAX_OPEN_CONNS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			maxOpen = n
		}
	}
	maxIdle := defaultMaxIdleConns
	if v := os.Getenv("KEYMASTER_DB_MAX_IDLE_CONNS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			maxIdle = n
		}
	}

	// For in-memory SQLite databases (":memory:" or file::memory:), force a single
	// open connection to avoid the SQLite per-connection in-memory database
	// semantics which can make schema changes invisible across different
	// connections. Tests commonly use ":memory:" and rely on a single DB.
	if dbType == "sqlite" && dsn == ":memory:" {
		maxOpen = 1
		maxIdle = 1
	}
	connMax := defaultConnMaxLifetime
	if v := os.Getenv("KEYMASTER_DB_CONN_MAX_LIFETIME_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			connMax = time.Duration(n) * time.Second
		}
	}

	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetMaxIdleConns(maxIdle)
	sqlDB.SetConnMaxLifetime(connMax)
	// ConnMaxIdleTime: control idle connection lifetime (seconds) via env var if set.
	connIdle := 60 // seconds
	if v := os.Getenv("KEYMASTER_DB_CONN_MAX_IDLE_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			connIdle = n
		}
	}
	sqlDB.SetConnMaxIdleTime(time.Duration(connIdle) * time.Second)

	openDur := time.Since(start)
	dbLogf("db: opened %s driver in %s (conn max open=%d, idle=%ds, maxLifetime=%s)", driverName, openDur, maxOpen, connIdle, connMax)

	migStart := time.Now()
	if err := RunMigrations(sqlDB, dbType); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}
	dbLogf("db: migrations for %s completed in %s", dbType, time.Since(migStart))
	// Create a Bun DB wrapper for the sql.DB based on dialect
	bunDB := createBunDB(sqlDB, dbType)
	switch dbType {
	case "sqlite":
		return &SqliteStore{bun: bunDB}, nil
	case "postgres":
		return &PostgresStore{bun: bunDB}, nil
	case "mysql":
		return &MySQLStore{bun: bunDB}, nil
	default:
		return nil, fmt.Errorf("unsupported database type for store creation: '%s'", dbType)
	}
}

// createBunDB constructs a *bun.DB for the provided *sql.DB and dbType.
// Centralizing construction makes it easier to apply consistent options
// and to test Bun initialization in one place.
func createBunDB(sqlDB *sql.DB, dbType string) *bun.DB {
	switch dbType {
	case "sqlite":
		return bun.NewDB(sqlDB, sqlitedialect.New())
	case "postgres":
		return bun.NewDB(sqlDB, pgdialect.New())
	case "mysql":
		return bun.NewDB(sqlDB, mysqldialect.New())
	default:
		// Fallback to SQLite dialect as a safe default; callers should validate dbType earlier.
		return bun.NewDB(sqlDB, sqlitedialect.New())
	}
}

// (old NewStore removed) Use NewStoreFromDSN to create stores from a DSN.

// RunMigrations applies the necessary database migrations for a given database connection.
func RunMigrations(db *sql.DB, dbType string) error {
	start := time.Now()
	dbLogf("db: starting migrations for %s", dbType)
	migrationsPath := fmt.Sprintf("migrations/%s", dbType)

	entries, err := fs.ReadDir(embeddedMigrations, migrationsPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// No migrations embedded for this DB type.
			dbLogf("db: applied migrations for %s in %s", dbType, time.Since(start))
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
			_ = tx.Rollback()
			return fmt.Errorf("failed to execute migration %s: %w", version, err)
		}

		// Insert migration record; use DB-specific placeholder
		insertQuery := "INSERT INTO schema_migrations(version, applied_at) VALUES(?, ?)"
		if dbType == "postgres" {
			insertQuery = "INSERT INTO schema_migrations(version, applied_at) VALUES($1, $2)"
		}
		if _, err := tx.Exec(insertQuery, version, time.Now()); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("failed to record migration %s: %w", version, err)
		}
		if err := tx.Commit(); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("failed to commit migration %s: %w", version, err)
		}
	}

	return nil
}

// ensureSchemaMigrationsTable creates schema_migrations if missing and adds
// the `applied_at` column when the table exists but is missing that column.
func ensureSchemaMigrationsTable(db *sql.DB, dbType string) error {
	// create the table with the desired schema if it does not exist
	// MySQL does not permit TEXT/BLOB columns to be indexed without a length,
	// so use a VARCHAR with a safe length there. Other engines can use TEXT.
	if dbType == "mysql" {
		if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (version VARCHAR(191) PRIMARY KEY, applied_at TIMESTAMP)`); err != nil {
			return err
		}
	} else {
		if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY, applied_at TIMESTAMP)`); err != nil {
			return err
		}
	}

	// Check whether applied_at column exists; if not, add it.
	hasAppliedAt := false
	switch dbType {
	case "sqlite":
		rows, err := db.Query("PRAGMA table_info(schema_migrations)")
		if err != nil {
			return err
		}
		defer func() { _ = rows.Close() }()
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
		defer func() { _ = rows.Close() }()
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
		// For SQLite the same statement works; no special handling needed
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
// NOTE: Account management now goes through the AccountManager interface.
// The old package-level helpers `AddAccount` and `DeleteAccount` were removed
// to force consumers to use `DefaultAccountManager()` or inject a manager.

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

// UpdateAccountIsDirty sets or clears the is_dirty flag for the account.
func UpdateAccountIsDirty(id int, dirty bool) error {
	return store.UpdateAccountIsDirty(id, dirty)
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

// Key-related operations are handled via the KeyManager interface (use
// DefaultKeyManager() or inject a KeyManager). The old package-level
// helper wrappers were removed to encourage explicit dependency injection.

// GetAllAuditLogEntries retrieves all entries from the audit log, most recent first.
func GetAllAuditLogEntries() ([]model.AuditLogEntry, error) {
	return store.GetAllAuditLogEntries()
}

// LogAction records an audit trail event.
func LogAction(action string, details string) error {
	// Prefer an injected AuditWriter when available (useful for tests).
	if w := DefaultAuditWriter(); w != nil {
		return w.LogAction(action, details)
	}
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
