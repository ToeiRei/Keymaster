// Package core contains small, deterministic interface definitions used by
// facade functions. Keep these interfaces minimal — they describe side‑effect
// boundaries that UIs and higher-level services will implement.
package core

import (
	"context"
	"io"

	"github.com/toeirei/keymaster/internal/model"
)

// Store defines minimal data-store operations used by CLI facades.
// Implementations will typically delegate to the DB layer.
type Store interface {
	GetAccounts() ([]model.Account, error)
	GetAccount(id int) (*model.Account, error)
	AddAccount(username, hostname, label, tags string) (int, error)
	DeleteAccount(accountID int) error
	AssignKeyToAccount(keyID, accountID int) error

	// Backup helpers
	GetBackupData(ctx context.Context) (*model.BackupData, error)
	ApplyBackup(ctx context.Context, data *model.BackupData) error
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
