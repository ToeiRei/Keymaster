// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

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

	// UpdateAccountIsDirty sets or clears the is_dirty flag for an account.
	UpdateAccountIsDirty(id int, dirty bool) error

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
	// FetchAuthorizedKeys should return the raw authorized_keys content from the remote host for the given account.
	FetchAuthorizedKeys(account model.Account) ([]byte, error)
	// ImportRemoteKeys fetches authorized_keys from the remote host and parses
	// them into public key models. It returns imported keys, skipped count,
	// an optional warning, and an error.
	ImportRemoteKeys(account model.Account) ([]model.PublicKey, int, string, error)
	// IsPassphraseRequired reports whether the given error indicates a missing
	// passphrase for a protected private key during SSH auth attempts.
	IsPassphraseRequired(err error) bool
}

// DecommissionResult mirrors the outcome reported by deploy package for each account.
type DecommissionResult struct {
	// Account contains the account object for which this result applies.
	Account model.Account

	// Account metadata
	AccountID     int
	AccountString string

	// Remote cleanup fields
	RemoteCleanupDone  bool
	RemoteCleanupError error

	// Database cleanup fields
	DatabaseDeleteDone  bool
	DatabaseDeleteError error

	// Backup path created during decommission (if any)
	BackupPath string

	// Skip/flags
	Skipped    bool
	SkipReason string
}

// DBMaintainer runs engine-specific maintenance operations.
type DBMaintainer interface {
	RunDBMaintenance(dbType, dsn string) error
}

// DecommissionOptions configures how a decommission should behave. This is a
// UI-facing, core-level representation so UIs can construct options without
// importing the lower-level deploy package. Adapters will convert this into
// the deploy package's DecommissionOptions when delegating.
type DecommissionOptions struct {
	SkipRemoteCleanup bool
	KeepFile          bool
	Force             bool
	DryRun            bool
	SelectiveKeys     []int
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
