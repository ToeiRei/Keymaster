// Package core contains small, deterministic interface definitions used by
// facade functions. Keep these interfaces minimal — they describe side‑effect
// boundaries that UIs and higher-level services will implement.
package core

import (
	"context"
	"io"
	"time"

	"github.com/toeirei/keymaster/internal/model"
)

// Store defines minimal data-store operations used by CLI facades.
// Implementations will typically delegate to the DB layer.
type Store interface {
	GetAccounts() ([]model.Account, error)
	GetAllActiveAccounts() ([]model.Account, error)
	GetAllAccounts() ([]model.Account, error)
	GetAccount(id int) (*model.Account, error)
	AddAccount(username, hostname, label, tags string) (int, error)
	DeleteAccount(accountID int) error
	AssignKeyToAccount(keyID, accountID int) error

	// System key helpers
	CreateSystemKey(publicKey, privateKey string) (int, error)
	RotateSystemKey(publicKey, privateKey string) (int, error)
	GetActiveSystemKey() (*model.SystemKey, error)

	// Host keys
	AddKnownHostKey(hostname, key string) error

	// Backup helpers
	ExportDataForBackup() (*model.BackupData, error)
	ImportDataFromBackup(*model.BackupData) error
	IntegrateDataFromBackup(*model.BackupData) error
}

// Deployer defines the minimal remote deployment operations.
type Deployer interface {
	DeployAuthorizedKeys(hostname, username, content string) error
	Close() error
}

// AuditWriter is the minimal contract for emitting audit events.
type AuditWriter interface {
	LogAction(action, details string) error
}

// HostFetcher fetches host key material from a remote host.
type HostFetcher interface {
	FetchHostKey(host string) (string, error)
}

// KeyGenerator generates and marshals key material.
type KeyGenerator interface {
	GenerateAndMarshalEd25519Key(comment, passphrase string) (publicKey string, privateKey string, err error)
}

// KeyManager provides higher-level key operations (importing public keys).
type KeyManager interface {
	AddPublicKey(alg string, keyData string, comment string, managed bool, expiresAt time.Time) error
}

// DeployerManager aggregates deploy-related operations used by facades.
type DeployerManager interface {
	DeployForAccount(account model.Account, keepFile bool) error
	AuditSerial(account model.Account) error
	AuditStrict(account model.Account) error
	DecommissionAccount(account model.Account, systemPrivateKey string, options interface{}) (DecommissionResult, error)
	BulkDecommissionAccounts(accounts []model.Account, systemPrivateKey string, options interface{}) ([]DecommissionResult, error)
	CanonicalizeHostPort(host string) string
	ParseHostPort(host string) (string, string, error)
	GetRemoteHostKey(host string) (string, error)
}

// DecommissionResult mirrors the outcome reported by deploy package for each account.
type DecommissionResult struct {
	Account             model.Account
	Skipped             bool
	DatabaseDeleteError error
}

// DBMaintainer runs engine-specific maintenance operations.
type DBMaintainer interface {
	RunDBMaintenance(dbType, dsn string) error
}

// StoreFactory can initialize a new Store from DSN (used by migrate).
type StoreFactory interface {
	NewStoreFromDSN(dbType, dsn string) (Store, error)
}

// Reporter is used by facades to emit progress or human-readable messages.
// Implementations may write to stdout, logs, or test buffers.
type Reporter interface {
	Reportf(format string, args ...any)
}

// BackupStore provides streaming helpers for backup/restore operations.
type BackupStore interface {
	WriteBackup(ctx context.Context, w io.Writer, data *model.BackupData) error
	ReadBackup(ctx context.Context, r io.Reader) (*model.BackupData, error)
}
