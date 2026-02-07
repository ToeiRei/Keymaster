// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// Package core defines high-level facades used by UI layers (CLI/TUI).
// This file contains Phase-4 P4-4: migrated business logic from the CLI
// into deterministic core functions. Functions operate via small interfaces
// declared in interfaces.go and return results/errors instead of performing
// UI operations.
package core

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/toeirei/keymaster/config"
	"github.com/toeirei/keymaster/core/bootstrap"
	"github.com/toeirei/keymaster/i18n"

	"github.com/toeirei/keymaster/core/model"
	"github.com/toeirei/keymaster/core/sshkey"
)

// DeployResult represents the outcome of a single account deployment.
type DeployResult struct {
	// Account is the account that was the target of the deployment.
	Account model.Account
	// Error is non-nil when the deployment failed for this account.
	Error error
}

// AuditResult represents the result of auditing a single account.
type AuditResult struct {
	// Account is the account audited.
	Account model.Account
	// Error is non-nil when the audit detected an error or failed.
	Error error
}

// DecommissionSummary aggregates counts from a decommission operation.
type DecommissionSummary struct {
	// Successful is the number of accounts successfully decommissioned.
	Successful int
	// Failed is the number of accounts that failed to be decommissioned.
	Failed int
	// Skipped is the number of accounts that were intentionally skipped.
	Skipped int
}

// RestoreOptions controls restore behavior used by `Restore`.
type RestoreOptions struct {
	// Full indicates whether to perform a full restore (true) or an
	// incremental/merge restore (false).
	Full bool
}

// DBMaintenanceOptions configures database maintenance operations.
type DBMaintenanceOptions struct {
	// SkipIntegrity when true will skip expensive integrity checks.
	SkipIntegrity bool
	// Timeout bounds the maintenance operation.
	Timeout time.Duration
}

// ParallelResult reports the name and optional error returned by a
// concurrently executed worker.
type ParallelResult struct {
	Name  string
	Error error
}

// InitializeServices initializes core-level services based on provided config.
// This remains a placeholder; CLI retains setupDefaultServices for now.
func InitializeServices(ctx context.Context, cfg *config.Config) (Store, error) {
	return nil, nil
}

// DeployAccounts orchestrates deployment for either a single target identifier
// or all active accounts. Uses the provided Store and DeployerManager.
func DeployAccounts(ctx context.Context, st Store, dm DeployerManager, identifier *string, rep Reporter) ([]DeployResult, error) {
	accounts, err := st.GetAllActiveAccounts()
	if err != nil {
		return nil, fmt.Errorf("get accounts: %w", err)
	}

	var targets []model.Account
	// RunDeployCmd runs the deploy command logic against the provided Store and
	// DeployerManager. `identifier` may be nil to operate on all accounts.
	if identifier != nil && *identifier != "" {
		found := false
		norm := strings.ToLower(*identifier)
		for _, acc := range accounts {
			if strings.ToLower(fmt.Sprintf("%s@%s", acc.Username, acc.Hostname)) == norm {
				targets = append(targets, acc)
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("account not found: %s", *identifier)
		}
	} else {
		targets = accounts
	}

	results := make([]DeployResult, 0, len(targets))
	for _, acc := range targets {
		err := dm.DeployForAccount(acc, false)
		results = append(results, DeployResult{Account: acc, Error: err})
	}
	return results, nil
}

// AuditAccounts runs audit across active accounts using DeployerManager audit helpers.
func AuditAccounts(ctx context.Context, st Store, dm DeployerManager, mode string, rep Reporter) ([]AuditResult, error) {
	accounts, err := st.GetAllActiveAccounts()
	if err != nil {
		return nil, fmt.Errorf("get accounts: %w", err)
	}

	results := make([]AuditResult, 0, len(accounts))
	for _, acc := range accounts {
		var aerr error
		switch strings.ToLower(strings.TrimSpace(mode)) {
		case "serial":
			aerr = dm.AuditSerial(acc)
		case "strict", "":
			// Strict mode: fetch remote authorized_keys and compare deterministic hash
			if acc.Serial == 0 {
				aerr = fmt.Errorf("%s", i18n.T("audit.error_not_deployed"))
				break
			}
			remote, ferr := dm.FetchAuthorizedKeys(acc)
			if ferr != nil {
				aerr = fmt.Errorf("%s", i18n.T("audit.error_read_remote_file", ferr))
				break
			}
			expected, gerr := GenerateKeysContent(acc.ID)
			if gerr != nil {
				aerr = fmt.Errorf("%s", i18n.T("audit.error_generate_expected", gerr))
				break
			}
			remoteHash := HashAuthorizedKeysContent(remote)
			expectedHash := HashAuthorizedKeysContent([]byte(expected))
			if remoteHash != expectedHash {
				aerr = fmt.Errorf("%s", i18n.T("audit.error_drift_detected"))
				// Record an audit event for detected drift (host change). Do not
				// write audit entries for matches — auditing is meant for host changes,
				// not verbose debug logging.
				if aw := DefaultAuditWriter(); aw != nil {
					_ = aw.LogAction("AUDIT_HASH_MISMATCH", fmt.Sprintf("account:%d stored:%s computed:%s", acc.ID, expectedHash, remoteHash))
				}
				// Mark the account dirty so other systems know the host state changed.
				if err := st.UpdateAccountIsDirty(acc.ID, true); err != nil {
					if aw := DefaultAuditWriter(); aw != nil {
						_ = aw.LogAction("AUDIT_HASH_MARK_DIRTY_FAILED", fmt.Sprintf("account:%d err:%v", acc.ID, err))
					}
				}
			}
		default:
			return nil, fmt.Errorf("invalid audit mode: %s", mode)
		}
		results = append(results, AuditResult{Account: acc, Error: aerr})
	}
	return results, nil
}

// TrustHost fetches a host key and optionally saves it in the store.
func TrustHost(ctx context.Context, canonicalHost string, hf HostFetcher, st Store, save bool) (string, error) {
	key, err := hf.FetchHostKey(canonicalHost)
	if err != nil {
		return "", fmt.Errorf("fetch host key: %w", err)
	}
	if save {
		if err := st.AddKnownHostKey(canonicalHost, key); err != nil {
			return key, fmt.Errorf("save known host key: %w", err)
		}
	}
	return key, nil
}

// ImportAuthorizedKeys parses an authorized_keys stream and imports found keys
// via the provided KeyManager.
func ImportAuthorizedKeys(ctx context.Context, r io.Reader, km KeyManager, rep Reporter) (imported int, skipped int, err error) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		alg, keyData, comment, perr := sshkey.Parse(line)
		if perr != nil {
			skipped++
			if rep != nil {
				rep.Reportf("Skipping invalid key line\n")
			}
			continue
		}
		if comment == "" {
			skipped++
			if rep != nil {
				rep.Reportf("Skipping key with empty comment\n")
			}
			continue
		}
		if err := km.AddPublicKey(alg, keyData, comment, false, time.Time{}); err != nil {
			skipped++
			if rep != nil {
				rep.Reportf("Skipping duplicate key (comment exists): %s\n", comment)
			}
			continue
		}
		imported++
		if rep != nil {
			rep.Reportf("Imported key: %s\n", comment)
		}
	}
	if sErr := scanner.Err(); sErr != nil {
		return imported, skipped, sErr
	}
	return imported, skipped, nil
}

// Backup exports the DB into BackupData using the Store.
func Backup(ctx context.Context, st Store) (*model.BackupData, error) {
	return st.ExportDataForBackup()
}

// WriteBackup writes compressed JSON backup data to writer.
func WriteBackup(ctx context.Context, data *model.BackupData, w io.Writer) error {
	zw, err := zstd.NewWriter(w)
	if err != nil {
		return fmt.Errorf("create zstd writer: %w", err)
	}
	defer func() { _ = zw.Close() }()
	enc := json.NewEncoder(zw)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		return fmt.Errorf("encode backup: %w", err)
	}
	return nil
}

// Restore reads a zstd-compressed JSON backup and imports it via the Store.
// RecoverFromCrash performs recovery tasks after an unexpected process exit.
func Restore(ctx context.Context, r io.Reader, opts RestoreOptions, st Store) error {
	zr, err := zstd.NewReader(r)
	if err != nil {
		return fmt.Errorf("create zstd reader: %w", err)
	}
	defer zr.Close()
	var data model.BackupData
	if err := json.NewDecoder(zr).Decode(&data); err != nil {
		return fmt.Errorf("decode backup: %w", err)
	}
	if opts.Full {
		return st.ImportDataFromBackup(&data)
	}
	return st.IntegrateDataFromBackup(&data)
}

// Migrate performs a backup from source store and imports into a newly created target store.
func Migrate(ctx context.Context, factory StoreFactory, st Store, targetType, targetDsn string) error {
	data, err := st.ExportDataForBackup()
	if err != nil {
		return fmt.Errorf("export backup: %w", err)
	}
	targetStore, err := factory.NewStoreFromDSN(targetType, targetDsn)
	if err != nil {
		return fmt.Errorf("init target store: %w", err)
	}
	if err := targetStore.ImportDataFromBackup(data); err != nil {
		return fmt.Errorf("import to target: %w", err)
	}
	return nil
}

// DecommissionAccounts runs decommission using DeployerManager and returns a summary.
func DecommissionAccounts(ctx context.Context, targets []model.Account, opts interface{}, dm DeployerManager, st Store, a AuditWriter) (DecommissionSummary, error) {
	sysKey, err := st.GetActiveSystemKey()
	if err != nil {
		return DecommissionSummary{}, fmt.Errorf("get system key: %w", err)
	}
	if sysKey == nil {
		return DecommissionSummary{}, fmt.Errorf("no active system key")
	}
	if len(targets) == 1 {
		res, err := dm.DecommissionAccount(targets[0], SystemKeyToSecret(sysKey), opts)
		if err != nil {
			return DecommissionSummary{}, err
		}
		summary := DecommissionSummary{}
		if res.Skipped {
			summary.Skipped = 1
		} else if res.DatabaseDeleteError != nil {
			summary.Failed = 1
		} else {
			summary.Successful = 1
		}
		return summary, nil
	}
	results, err := dm.BulkDecommissionAccounts(targets, SystemKeyToSecret(sysKey), opts)
	if err != nil {
		return DecommissionSummary{}, err
	}
	summary := DecommissionSummary{}
	for _, r := range results {
		if r.Skipped {
			summary.Skipped++
		} else if r.DatabaseDeleteError != nil {
			summary.Failed++
		} else {
			summary.Successful++
		}
	}
	return summary, nil
}

// RunDBMaintenance delegates to DBMaintainer.
func RunDBMaintenance(ctx context.Context, maint DBMaintainer, dbType, dsn string, opts DBMaintenanceOptions) error {
	return maint.RunDBMaintenance(dbType, dsn)
}

// ExportSSHConfig builds an SSH config text for active accounts.
func ExportSSHConfig(ctx context.Context, st Store) (string, error) {
	accounts, err := st.GetAllActiveAccounts()
	if err != nil {
		return "", fmt.Errorf("get accounts: %w", err)
	}
	if len(accounts) == 0 {
		return "", nil
	}
	var b strings.Builder
	b.WriteString("# SSH config generated by Keymaster\n")
	b.WriteString(fmt.Sprintf("# date: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))
	for _, account := range accounts {
		hostAlias := account.Label
		if hostAlias == "" {
			hostAlias = fmt.Sprintf("%s-%s", account.Username, strings.ReplaceAll(account.Hostname, ".", "-"))
		}
		b.WriteString(fmt.Sprintf("# %s\n", account.String()))
		b.WriteString(fmt.Sprintf("Host %s\n", hostAlias))
		b.WriteString(fmt.Sprintf("    HostName %s\n", account.Hostname))
		b.WriteString(fmt.Sprintf("    User %s\n", account.Username))
		b.WriteString("\n")
	}
	return b.String(), nil
}

// FindAccountByIdentifier finds an account by ID, user@host, or label.
func FindAccountByIdentifier(identifier string, accounts []model.Account) (*model.Account, error) {
	var id int
	if n, err := fmt.Sscanf(identifier, "%d", &id); n == 1 && err == nil {
		for _, acc := range accounts {
			if acc.ID == id {
				return &acc, nil
			}
		}
		return nil, fmt.Errorf("no account with id %s", identifier)
	}
	if strings.Contains(identifier, "@") {
		norm := strings.ToLower(identifier)
		for _, acc := range accounts {
			if strings.ToLower(fmt.Sprintf("%s@%s", acc.Username, acc.Hostname)) == norm {
				return &acc, nil
			}
		}
	}
	for _, acc := range accounts {
		if strings.EqualFold(acc.Label, identifier) {
			return &acc, nil
		}
	}
	return nil, fmt.Errorf("no account found with identifier: %s", identifier)
}

// ParallelRun executes worker concurrently for each account and collects results.
func ParallelRun(ctx context.Context, accounts []model.Account, worker func(model.Account) error) []ParallelResult {
	results := make([]ParallelResult, 0, len(accounts))
	ch := make(chan ParallelResult, len(accounts))
	for _, acc := range accounts {
		a := acc
		go func() {
			err := worker(a)
			ch <- ParallelResult{Name: a.String(), Error: err}
		}()
	}
	for i := 0; i < len(accounts); i++ {
		results = append(results, <-ch)
	}
	close(ch)
	return results
}

// CLI-facing wrappers that call the core functions above. These are kept so
// the CLI can be rewired to call these facades in P4-5.

// RunDeployCmd runs the deploy command against the provided Store and
// DeployerManager. `identifier` may be nil to operate on all accounts.
func RunDeployCmd(ctx context.Context, st Store, dm DeployerManager, identifier *string, rep Reporter) ([]DeployResult, error) {
	return DeployAccounts(ctx, st, dm, identifier, rep)
}

// RunDeployForAccount calls DeployerManager for a single account deployment.
func RunDeployForAccount(ctx context.Context, dm DeployerManager, account model.Account, rep Reporter) error {
	return dm.DeployForAccount(account, false)
}

func RunRotateKeyCmd(ctx context.Context, kg KeyGenerator, st Store, passphrase string) (int, error) {
	pub, priv, err := kg.GenerateAndMarshalEd25519Key("keymaster-system-key", passphrase)
	if err != nil {
		return 0, fmt.Errorf("generate key: %w", err)
	}
	return st.RotateSystemKey(pub, priv)
}

func RunAuditCmd(ctx context.Context, st Store, dm DeployerManager, mode string, rep Reporter) ([]AuditResult, error) {
	return AuditAccounts(ctx, st, dm, mode, rep)
}

// RunAuditForAccount runs audit for a single account via the DeployerManager.
func RunAuditForAccount(ctx context.Context, dm DeployerManager, account model.Account, mode string, rep Reporter) error {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "serial":
		return dm.AuditSerial(account)
	case "strict", "":
		return dm.AuditStrict(account)
	default:
		return fmt.Errorf("invalid audit mode: %s", mode)
	}
}

func RunImportCmd(ctx context.Context, r io.Reader, km KeyManager, rep Reporter) (imported int, skipped int, err error) {
	return ImportAuthorizedKeys(ctx, r, km, rep)
}

// RunImportRemoteCmd fetches authorized_keys from remote via DeployerManager
// and imports via the provided KeyManager, reporting via Reporter.
func RunImportRemoteCmd(ctx context.Context, account model.Account, dm DeployerManager, km KeyManager, rep Reporter) (imported int, skipped int, warning string, err error) {
	content, ferr := dm.FetchAuthorizedKeys(account)
	if ferr != nil {
		return 0, 0, "", fmt.Errorf("fetch remote authorized_keys: %w", ferr)
	}
	imported, skipped, ierr := ImportAuthorizedKeys(ctx, strings.NewReader(string(content)), km, rep)
	return imported, skipped, "", ierr
}

func RunTrustHostCmd(ctx context.Context, canonicalHost string, dm DeployerManager, st Store, save bool) (string, error) {
	key, err := dm.GetRemoteHostKey(canonicalHost)
	if err != nil {
		return "", fmt.Errorf("fetch remote host key: %w", err)
	}
	if save {
		if err := st.AddKnownHostKey(canonicalHost, key); err != nil {
			return key, fmt.Errorf("save known host key: %w", err)
		}
	}
	return key, nil
}

func RunExportSSHConfigCmd(ctx context.Context, st Store) (string, error) {
	return ExportSSHConfig(ctx, st)
}

func RunDBMaintainCmd(ctx context.Context, maint DBMaintainer, dbType, dsn string, opts DBMaintenanceOptions) error {
	return RunDBMaintenance(ctx, maint, dbType, dsn, opts)
}

func RunBackupCmd(ctx context.Context, st Store) (*model.BackupData, error) {
	return Backup(ctx, st)
}

func RunWriteBackupCmd(ctx context.Context, data *model.BackupData, w io.Writer) error {
	return WriteBackup(ctx, data, w)
}

func RunRestoreCmd(ctx context.Context, r io.Reader, opts RestoreOptions, st Store) error {
	return Restore(ctx, r, opts, st)
}

func RunMigrateCmd(ctx context.Context, factory StoreFactory, st Store, targetType, targetDsn string) error {
	return Migrate(ctx, factory, st, targetType, targetDsn)
}

func RunDecommissionCmd(ctx context.Context, targets []model.Account, opts interface{}, dm DeployerManager, st Store, a AuditWriter) (DecommissionSummary, error) {
	return DecommissionAccounts(ctx, targets, opts, dm, st, a)
}

// Bootstrap lifecycle thin wrappers — delegate to internal/bootstrap.
// These exist so UI/CLI code can call core facades instead of importing
// internal/bootstrap directly.

// RecoverFromCrash performs bootstrap recovery after an unexpected exit.
func RecoverFromCrash() error {
	return bootstrap.RecoverFromCrash()
}

func StartSessionReaper() {
	bootstrap.StartSessionReaper()
}

func InstallSignalHandler() {
	bootstrap.InstallSignalHandler()
}

func CleanupAllActiveSessions() error {
	return bootstrap.CleanupAllActiveSessions()
}

// DB init helpers (thin wrappers around internal/db) -----------------------

// IsDBInitialized returns true when the database has been initialized.
func IsDBInitialized() bool {
	return DefaultIsDBInitialized()
}

// EnableAccount sets an account to active.
func EnableAccount(st Store, id int) error {
	allAccounts, err := st.GetAllAccounts()
	if err != nil {
		return fmt.Errorf("failed to load accounts: %w", err)
	}
	var account *model.Account
	for i, acc := range allAccounts {
		if acc.ID == id {
			account = &allAccounts[i]
			break
		}
	}
	if account == nil {
		return fmt.Errorf("account not found: %d", id)
	}
	if account.IsActive {
		return nil // Already enabled
	}
	if err := st.ToggleAccountStatus(id); err != nil {
		return fmt.Errorf("failed to enable account: %w", err)
	}
	return nil
}

// DisableAccount sets an account to inactive.
func DisableAccount(st Store, id int) error {
	allAccounts, err := st.GetAllAccounts()
	if err != nil {
		return fmt.Errorf("failed to load accounts: %w", err)
	}
	var account *model.Account
	for i, acc := range allAccounts {
		if acc.ID == id {
			account = &allAccounts[i]
			break
		}
	}
	if account == nil {
		return fmt.Errorf("account not found: %d", id)
	}
	if !account.IsActive {
		return nil // Already disabled
	}
	if err := st.ToggleAccountStatus(id); err != nil {
		return fmt.Errorf("failed to disable account: %w", err)
	}
	return nil
}

// DeleteAccount deletes an account and all its associated key assignments.
func DeleteAccount(am AccountManager, st Store, id int, force bool, confirmFunc func(account *model.Account) bool) error {
	allAccounts, err := st.GetAllAccounts()
	if err != nil {
		return fmt.Errorf("failed to load accounts: %w", err)
	}
	var account *model.Account
	for i, acc := range allAccounts {
		if acc.ID == id {
			account = &allAccounts[i]
			break
		}
	}
	if account == nil {
		return fmt.Errorf("account not found: %d", id)
	}
	if !force && confirmFunc != nil {
		if !confirmFunc(account) {
			return nil // Deletion cancelled
		}
	}
	if err := am.DeleteAccount(id); err != nil {
		return fmt.Errorf("failed to delete account: %w", err)
	}
	return nil
}

// AssignKeyToAccount assigns a key to an account.
func AssignKeyToAccount(km KeyManager, st Store, keyID, accountID int) error {
	allAccounts, err := st.GetAllAccounts()
	if err != nil {
		return fmt.Errorf("failed to load accounts: %w", err)
	}
	accountExists := false
	for _, acc := range allAccounts {
		if acc.ID == accountID {
			accountExists = true
			break
		}
	}
	if !accountExists {
		return fmt.Errorf("account not found: %d", accountID)
	}
	if err := km.AssignKeyToAccount(keyID, accountID); err != nil {
		return fmt.Errorf("failed to assign key: %w", err)
	}
	return nil
}

// UnassignKeyFromAccount removes a key assignment from an account.
func UnassignKeyFromAccount(km KeyManager, keyID, accountID int) error {
	if err := km.UnassignKeyFromAccount(keyID, accountID); err != nil {
		return fmt.Errorf("failed to unassign key: %w", err)
	}
	return nil
}

// CreateAccount creates a new account with the given parameters.
func CreateAccount(am AccountManager, username, hostname, label, tags string) (int, error) {
	if username == "" {
		return 0, fmt.Errorf("--username is required")
	}
	if hostname == "" {
		return 0, fmt.Errorf("--hostname is required")
	}
	id, err := am.AddAccount(username, hostname, label, tags)
	if err != nil {
		return 0, fmt.Errorf("failed to create account: %w", err)
	}
	return id, nil
}

// UpdateAccount updates hostname, label, or tags for an existing account.
func UpdateAccount(st Store, id int, hostname, label, tags *string) error {
	// Check if account exists
	allAccounts, err := st.GetAllAccounts()
	if err != nil {
		return fmt.Errorf("failed to load accounts: %w", err)
	}
	accountExists := false
	for _, acc := range allAccounts {
		if acc.ID == id {
			accountExists = true
			break
		}
	}
	if !accountExists {
		return fmt.Errorf("account not found: %d", id)
	}

	// Update fields if provided
	updated := false
	if hostname != nil {
		if *hostname != "" {
			if err := st.UpdateAccountHostname(id, *hostname); err != nil {
				return fmt.Errorf("failed to update hostname: %w", err)
			}
			updated = true
		}
	}
	if label != nil {
		if err := st.UpdateAccountLabel(id, *label); err != nil {
			return fmt.Errorf("failed to update label: %w", err)
		}
		updated = true
	}
	if tags != nil {
		if err := st.UpdateAccountTags(id, *tags); err != nil {
			return fmt.Errorf("failed to update tags: %w", err)
		}
		updated = true
	}
	if !updated {
		return fmt.Errorf("no fields to update. Use hostname, label, or tags")
	}
	return nil
}

// ListAccounts returns all accounts, optionally filtered by status and/or search term.
func ListAccounts(st Store, statusFilter, searchTerm string) ([]model.Account, error) {
	accounts, err := st.GetAllAccounts()
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}

	// Filter by status
	if statusFilter != "" {
		filtered := []model.Account{}
		isActive := statusFilter == "active"
		for _, acc := range accounts {
			if acc.IsActive == isActive {
				filtered = append(filtered, acc)
			}
		}
		accounts = filtered
	}

	// Filter by search term
	if searchTerm != "" {
		searchLower := strings.ToLower(searchTerm)
		filtered := []model.Account{}
		for _, acc := range accounts {
			if strings.Contains(strings.ToLower(acc.Username), searchLower) ||
				strings.Contains(strings.ToLower(acc.Hostname), searchLower) ||
				strings.Contains(strings.ToLower(acc.Label), searchLower) {
				filtered = append(filtered, acc)
			}
		}
		accounts = filtered
	}

	return accounts, nil
}

// ShowAccount returns a single account by ID or hostname.
func ShowAccount(st Store, identifier string) (*model.Account, error) {
	// Try parsing as ID first, then as hostname
	if id, parseErr := strconv.Atoi(identifier); parseErr == nil {
		allAccounts, err := st.GetAllAccounts()
		if err != nil {
			return nil, fmt.Errorf("failed to load accounts: %w", err)
		}
		for i, acc := range allAccounts {
			if acc.ID == id {
				return &allAccounts[i], nil
			}
		}
	} else {
		accounts, err := st.GetAllAccounts()
		if err != nil {
			return nil, fmt.Errorf("failed to load accounts: %w", err)
		}
		for i, acc := range accounts {
			if acc.Hostname == identifier {
				return &accounts[i], nil
			}
		}
	}
	return nil, fmt.Errorf("account not found: %s", identifier)
}
