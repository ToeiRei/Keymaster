// Package core defines high-level facades used by UI layers (CLI/TUI).
// This file contains empty stubs for the Phase-4 P4-2: Core Facade Inventory
// These functions are intentionally unimplemented and return zero values.
package core

import (
	"context"
	"io"
	"time"

	"github.com/toeirei/keymaster/internal/config"
	"github.com/toeirei/keymaster/internal/model"
)

// Interfaces used by facades are declared in interfaces.go. Keep that file
// authoritative for core-facing contracts.

// Option/result types (placeholders) ---------------------------------------
type DeployResult struct {
	Account model.Account
	Error   error
}

type AuditResult struct {
	Account model.Account
	Error   error
}

type DecommissionSummary struct {
	Successful int
	Failed     int
	Skipped    int
}

type RestoreOptions struct {
	Full bool
}

type DBMaintenanceOptions struct {
	SkipIntegrity bool
	Timeout       time.Duration
}

type ParallelResult struct {
	Name  string
	Error error
}

// Facade stubs -------------------------------------------------------------

// InitializeServices initializes core-level services based on provided config.
// TODO: implement initialization logic (config, i18n, DB, bootstrap).
func InitializeServices(ctx context.Context, cfg *config.Config) (Store, error) {
	return nil, nil
}

// DeployAccounts orchestrates deployment for a list of accounts using the provided Deployer.
// TODO: implement deployment orchestration and reporting.
func DeployAccounts(ctx context.Context, accounts []model.Account, d Deployer, rep Reporter) ([]DeployResult, error) {
	return nil, nil
}

// AuditAccounts runs audits for provided accounts using the selected mode.
// TODO: implement audit orchestration.
func AuditAccounts(ctx context.Context, accounts []model.Account, mode string, a AuditWriter, rep Reporter) ([]AuditResult, error) {
	return nil, nil
}

// NOTE: system key rotation helpers already exist in keygen_ops.go and
// expose `RotateSystemKey(store SystemKeyStore, passphrase string) (int, error)`.
// Facade-level wrappers will be added later in P4-3 when interfaces are
// stabilized.

// TrustHost fetches a remote host key and persists it via the provided store.
// TODO: implement host fetching and storage.
func TrustHost(ctx context.Context, canonicalHost string, hf HostFetcher, st Store) error {
	return nil
}

// ImportAuthorizedKeys parses authorized_keys data from reader and imports keys.
// Returns counts of imported and skipped entries.
// TODO: implement import logic and duplicate handling.
func ImportAuthorizedKeys(ctx context.Context, r io.Reader, keyManager Store, rep Reporter) (imported int, skipped int, err error) {
	return 0, 0, nil
}

// Backup exports the database contents into a BackupData structure.
// TODO: implement export logic.
func Backup(ctx context.Context, st Store) (*model.BackupData, error) {
	return nil, nil
}

// WriteBackup writes a backup data structure to the provided writer (compressed).
// TODO: implement write logic.
func WriteBackup(ctx context.Context, data *model.BackupData, w io.Writer) error {
	return nil
}

// Restore imports backup data using provided options.
// TODO: implement restore logic.
func Restore(ctx context.Context, r io.Reader, opts RestoreOptions, st Store) error {
	return nil
}

// Migrate migrates data to a target database described by type and dsn.
// TODO: implement migration orchestration.
func Migrate(ctx context.Context, targetType, targetDsn string) error {
	return nil
}

// DecommissionAccounts runs decommission for a set of accounts.
// TODO: implement decommission orchestration.
func DecommissionAccounts(ctx context.Context, targets []model.Account, opts interface{}, d Deployer, st Store, a AuditWriter) (DecommissionSummary, error) {
	return DecommissionSummary{}, nil
}

// RunDBMaintenance runs maintenance operations for the configured DB.
// TODO: implement engine-specific maintenance.
func RunDBMaintenance(ctx context.Context, opts DBMaintenanceOptions, st Store) error {
	return nil
}

// ExportSSHConfig produces an SSH client config string for the given accounts.
// TODO: implement export logic.
func ExportSSHConfig(ctx context.Context, accounts []model.Account) (string, error) {
	return "", nil
}

// FindAccountByIdentifier finds an account by ID, user@host, or label.
// TODO: move and reuse helper from CLI.
func FindAccountByIdentifier(identifier string, accounts []model.Account) (*model.Account, error) {
	return nil, nil
}

// ParallelRun executes worker concurrently for each account and returns results.
// TODO: generalize and replace CLI runParallelTasks.
func ParallelRun(ctx context.Context, accounts []model.Account, worker func(model.Account) error) []ParallelResult {
	return nil
}

// CLI-facing facades (P4-2): one stub per CLI subcommand.
// These are intentionally unimplemented and only provide signatures
// so the CLI may be rewired in later phases. Each has a TODO marker.

// RunDeployCmd is the facade for the `deploy` CLI command.
// TODO: implement orchestration and options handling.
func RunDeployCmd(ctx context.Context, accounts []model.Account) ([]DeployResult, error) {
	return nil, nil
}

// RunRotateKeyCmd is the facade for the `rotate-key` CLI command.
// TODO: implement passphrase handling and system key rotation.
func RunRotateKeyCmd(ctx context.Context, passphrase string) (int, error) {
	return 0, nil
}

// RunAuditCmd is the facade for the `audit` CLI command.
// TODO: implement audit modes and reporting.
func RunAuditCmd(ctx context.Context, accounts []model.Account, mode string) ([]AuditResult, error) {
	return nil, nil
}

// RunImportCmd is the facade for the `import` CLI command.
// TODO: parse input and import keys into core stores.
func RunImportCmd(ctx context.Context, r io.Reader) (imported int, skipped int, err error) {
	return 0, 0, nil
}

// RunTrustHostCmd is the facade for the `trust-host` CLI command.
// TODO: fetch remote host key and persist.
func RunTrustHostCmd(ctx context.Context, canonicalHost string) error {
	return nil
}

// RunExportSSHConfigCmd is the facade for the `export-ssh-client-config` command.
// TODO: generate client config for accounts.
func RunExportSSHConfigCmd(ctx context.Context, accounts []model.Account) (string, error) {
	return "", nil
}

// RunDBMaintainCmd is the facade for the `db-maintain` CLI command.
// TODO: perform database-specific maintenance tasks.
func RunDBMaintainCmd(ctx context.Context, opts DBMaintenanceOptions) error {
	return nil
}

// RunBackupCmd is the facade for the `backup` CLI command.
// TODO: export DB and related artifacts as backup data.
func RunBackupCmd(ctx context.Context) (*model.BackupData, error) {
	return nil, nil
}

// RunWriteBackupCmd writes backup data to an io.Writer (used by CLI backup piping).
// TODO: implement compression and streaming.
func RunWriteBackupCmd(ctx context.Context, data *model.BackupData, w io.Writer) error {
	return nil
}

// RunRestoreCmd is the facade for the `restore` CLI command.
// TODO: perform restore using provided options (full/partial).
func RunRestoreCmd(ctx context.Context, r io.Reader, opts RestoreOptions) error {
	return nil
}

// RunMigrateCmd is the facade for the `migrate` CLI command.
// TODO: orchestrate migration to the target DB.
func RunMigrateCmd(ctx context.Context, targetType, targetDsn string) error {
	return nil
}

// RunDecommissionCmd is the facade for the `decommission` CLI command.
// TODO: implement remote cleanup and DB deletion orchestration.
func RunDecommissionCmd(ctx context.Context, targets []model.Account, opts interface{}) (DecommissionSummary, error) {
	return DecommissionSummary{}, nil
}

// GetVersionInfo returns resolved version, commit and build date for the binary.
// TODO: wire to build metadata resolution used in CLI.
func GetVersionInfo() (versionOut, commitOut, dateOut string) {
	return "", "", ""
}

// RunDebugCmd performs diagnostic checks for `keymaster debug` and returns
// a textual diagnostic output for the user.
// TODO: implement config inspection and debug helpers.
func RunDebugCmd(ctx context.Context) (string, error) {
	return "", nil
}
