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
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/toeirei/keymaster/internal/config"
	"github.com/toeirei/keymaster/internal/model"
	"github.com/toeirei/keymaster/internal/sshkey"
)

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
			aerr = dm.AuditStrict(acc)
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
			continue
		}
		if comment == "" {
			skipped++
			continue
		}
		if err := km.AddPublicKey(alg, keyData, comment, false, time.Time{}); err != nil {
			skipped++
			continue
		}
		imported++
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
	defer zw.Close()
	enc := json.NewEncoder(zw)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		return fmt.Errorf("encode backup: %w", err)
	}
	return nil
}

// Restore reads a zstd-compressed JSON backup and imports it via the Store.
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
		res, err := dm.DecommissionAccount(targets[0], sysKey.PrivateKey, opts)
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
	results, err := dm.BulkDecommissionAccounts(targets, sysKey.PrivateKey, opts)
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
func RunDeployCmd(ctx context.Context, st Store, dm DeployerManager, identifier *string, rep Reporter) ([]DeployResult, error) {
	return DeployAccounts(ctx, st, dm, identifier, rep)
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

func RunImportCmd(ctx context.Context, r io.Reader, km KeyManager, rep Reporter) (imported int, skipped int, err error) {
	return ImportAuthorizedKeys(ctx, r, km, rep)
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
