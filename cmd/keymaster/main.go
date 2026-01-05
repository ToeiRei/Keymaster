// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// main.go sets up the command-line interface (CLI) for the Keymaster
// application using the Cobra library. It defines the root command,
// subcommands (like deploy, audit, rotate-key), flags, and the main
// entry point for execution.

package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"runtime/debug"

	_ "github.com/go-sql-driver/mysql" // Blank import for migrate command
	_ "github.com/jackc/pgx/v5/stdlib" // Blank import for migrate command
	"github.com/klauspost/compress/zstd"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/toeirei/keymaster/internal/bootstrap"
	"github.com/toeirei/keymaster/internal/config"
	"github.com/toeirei/keymaster/internal/core"
	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/deploy"
	"github.com/toeirei/keymaster/internal/i18n"
	"github.com/toeirei/keymaster/internal/model"
	"github.com/toeirei/keymaster/internal/sshkey"
	"github.com/toeirei/keymaster/internal/tui"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var version = "dev"   // this will be set by the linker
var gitCommit = "dev" // set at build time with the short commit SHA
var buildDate = ""    // set at build time (RFC3339)
var cfgFile string
var auditMode string // audit mode flag: "strict" (default) or "serial"
var fullRestore bool // Flag for the restore command

var password string // Flag for rotate-key password
// TODO should be moved to project root
var appConfig config.Config

func setupDefaultServices(cmd *cobra.Command, args []string) error {
	// Load optional config file argument from cli
	optional_config_path, err := getConfigPathFromCli(cmd)
	if err != nil {
		return err
	}

	// Diagnostic: print current working directory and KEYMASTER-related env vars
	if wd, wderr := os.Getwd(); wderr == nil {
		log.Printf("startup cwd: %s", wd)
	}
	// Print any environment variables that might affect config discovery/parsing
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "KEYMASTER_") || strings.HasPrefix(e, "KEYMASTER") || strings.HasPrefix(e, "CONFIG") {
			log.Printf("env: %s", e)
		}
	}

	// Load config
	defauls := map[string]any{
		"database.type": "sqlite",
		"database.dsn":  "./keymaster.db",
		"language":      "en",
	}

	appConfig, err = config.LoadConfig[config.Config](cmd, defauls, optional_config_path)
	// A "file not found" error is expected on first run, so we handle it specifically.
	// Other errors during loading are usually fatal, but allow debugging when the
	// error is due to control characters in YAML (so `keymaster debug` can run).
	if errors.As(err, &viper.ConfigFileNotFoundError{}) {
		// This is the first run, or the config file was deleted. Create a default one.
		if writeErr := config.WriteConfigFile(&appConfig, false); writeErr != nil {
			// Log a warning but don't fail, as the app can run on defaults.
			log.Printf("Warning: could not write default config file: %v", writeErr)
		}
	} else if err != nil {
		// If it's a YAML parse error caused by control characters, log a
		// user-friendly message pointing users at the `debug` command, but
		// continue running on defaults so the app is still usable.
		if strings.Contains(err.Error(), "control characters are not allowed") {
			used := viper.ConfigFileUsed()
			if used == "" {
				log.Printf("The config appears to be invalid (parse error). Run 'keymaster debug' to inspect configuration files: %v", err)
			} else {
				log.Printf("The config you are using (%s) appears to be invalid: %v. Run 'keymaster debug' to inspect and fix it.", used, err)
			}
		} else {
			return fmt.Errorf("error loading config: %w", err)
		}
	}

	// If no config file was used (viper didn't load one), always write a default
	// config for the user so subsequent runs have a persisted file to inspect.
	if viper.ConfigFileUsed() == "" {
		if writeErr := config.WriteConfigFile(&appConfig, false); writeErr != nil {
			log.Printf("Warning: could not write default config file: %v", writeErr)
		} else {
			log.Printf("Wrote default config to user config path")
		}
	}

	// Post-process config to ensure critical values are not empty, falling back to defaults.
	// This handles cases where the user's config file has empty values for these fields.
	// We also update viper's internal state to ensure subsequent saves are correct.
	if appConfig.Database.Type == "" {
		appConfig.Database.Type = defauls["database.type"].(string)
		viper.Set("database.type", appConfig.Database.Type)
	}
	if appConfig.Database.Dsn == "" {
		appConfig.Database.Dsn = defauls["database.dsn"].(string)
		viper.Set("database.dsn", appConfig.Database.Dsn)
	}
	if appConfig.Language == "" {
		appConfig.Language = defauls["language"].(string)
		viper.Set("language", appConfig.Language)
	}

	// Initialize i18n
	i18n.Init(appConfig.Language)

	// Initialize the database if not already initialized by tests or earlier setup.
	if !db.IsInitialized() {
		if err := db.InitDB(appConfig.Database.Type, appConfig.Database.Dsn); err != nil {
			return errors.New(i18n.T("config.error_init_db", err))
		}
	}

	// Recover from any previous crashes
	if err := bootstrap.RecoverFromCrash(); err != nil {
		log.Printf("Bootstrap recovery error: %v", err)
	}

	// Start background session reaper
	bootstrap.StartSessionReaper()

	return nil
}

// main is the entry point of the application.
func main() {
	// Install signal handler for graceful shutdown of bootstrap sessions
	bootstrap.InstallSignalHandler()

	// Set up cleanup store for bootstrap operations
	defer func() {
		if err := bootstrap.CleanupAllActiveSessions(); err != nil {
			log.Printf("Error during final cleanup: %v", err)
		} else {
			log.Println("Bootstrap cleanup complete.")
		}
	}()

	rootCmd := NewRootCmd()

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func applyDefaultFlags(cmd *cobra.Command) {
	// Avoid redefining flags if they already exist (NewRootCmd may be called
	// multiple times in tests which creates a new root but uses package-level
	// subcommands). pflag will panic on duplicate flag definitions, so check
	// first.
	if cmd.Flags().Lookup("database.type") == nil {
		cmd.Flags().String("database.type", "sqlite", "Database type (e.g., sqlite, postgres)")
	}
	if cmd.Flags().Lookup("database.dsn") == nil {
		cmd.Flags().String("database.dsn", "./keymaster.db", "Database connection string (DSN)")
	}
}

func getConfigPathFromCli(cmd *cobra.Command) (*string, error) {
	// Load optional config file argument from cli
	// Only proceed if the user has explicitly set the --config flag.
	if cmd.Flags().Changed("config") {
		path, err := cmd.Flags().GetString("config")
		if err != nil {
			// This is unlikely if Changed() is true, but good practice.
			return nil, fmt.Errorf("could not read --config flag: %w", err)
		}

		// If the flag is set but the value is empty, do nothing.
		if path == "" {
			return nil, nil
		}

		// Make sure the user-provided file exists to avoid unwanted behavior.
		if _, err := os.Stat(path); err != nil {
			return nil, fmt.Errorf("config file specified via --config flag not found or is not accessible: %w", err)
		}
		return &path, nil // Return the valid path
	}
	return nil, nil
}

// NewRootCmd creates and configures a new root cobra command.
// This function is used to create the main application command as well as
// fresh instances for isolated testing.
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "keymaster",
		Short: "Keymaster is a lightweight, agentless SSH access manager.",
		Long: `Keymaster centralizes control of authorized_keys files.
Instead of scattering user keys everywhere, Keymaster plants a single
system key per account and uses it as a foothold to rewrite and
version-control access. A database becomes the source of truth.

Running without a subcommand will launch the interactive TUI.`,
		PersistentPreRunE: setupDefaultServices,
		Run: func(cmd *cobra.Command, args []string) {
			// The database is already initialized by PersistentPreRunE.
			// i18n is also initialized, so we can just run the TUI.
			tui.Run()
		},
	}

	v, c, d := resolveBuildVersion(nil)
	compositeVersion := v
	if c != "" && c != "dev" {
		compositeVersion = compositeVersion + " (" + c + ")"
	}
	if d != "" {
		compositeVersion = compositeVersion + " built: " + d
	}
	cmd.Version = compositeVersion

	// Register debug command
	cmd.AddCommand(debugCmd)

	// Define flags
	cmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file")
	cmd.PersistentFlags().String("language", "en", `TUI language ("en", "de")`)
	applyDefaultFlags(cmd)

	// Add subcommand flags
	applyDefaultFlags(deployCmd)
	applyDefaultFlags(rotateKeyCmd)
	applyDefaultFlags(auditCmd)
	if rotateKeyCmd.Flags().Lookup("password") == nil {
		rotateKeyCmd.Flags().StringVarP(&password, "password", "p", "", "Optional password to encrypt the new private key")
	}
	if auditCmd.Flags().Lookup("mode") == nil {
		auditCmd.Flags().StringVarP(&auditMode, "mode", "m", "strict", "Audit mode: 'strict' (full file comparison) or 'serial' (header serial only)")
	}

	applyDefaultFlags(importCmd)
	applyDefaultFlags(trustHostCmd)
	applyDefaultFlags(exportSSHConfigCmd)
	applyDefaultFlags(dbMaintainCmd)
	if dbMaintainCmd.Flags().Lookup("skip-integrity") == nil {
		dbMaintainCmd.Flags().Bool("skip-integrity", false, "Skip integrity_check (SQLite) during maintenance")
	}
	if dbMaintainCmd.Flags().Lookup("timeout") == nil {
		dbMaintainCmd.Flags().Int("timeout", 0, "Timeout in seconds for maintenance (0 means no timeout)")
	}
	applyDefaultFlags(restoreCmd)
	if restoreCmd.Flags().Lookup("full") == nil {
		restoreCmd.Flags().BoolVar(&fullRestore, "full", false, "Perform a full, destructive restore (wipes all existing data first)")
	}

	applyDefaultFlags(migrateCmd)
	applyDefaultFlags(decommissionCmd)
	if decommissionCmd.Flags().Lookup("skip-remote") == nil {
		decommissionCmd.Flags().Bool("skip-remote", false, "Skip remote SSH cleanup (only delete from database)")
	}
	if decommissionCmd.Flags().Lookup("keep-file") == nil {
		decommissionCmd.Flags().Bool("keep-file", false, "Remove only Keymaster content, keep other keys in authorized_keys")
	}
	if decommissionCmd.Flags().Lookup("force") == nil {
		decommissionCmd.Flags().Bool("force", false, "Continue even if remote cleanup fails")
	}
	if decommissionCmd.Flags().Lookup("dry-run") == nil {
		decommissionCmd.Flags().Bool("dry-run", false, "Show what would be done without making changes")
	}
	if decommissionCmd.Flags().Lookup("tag") == nil {
		decommissionCmd.Flags().String("tag", "", "Decommission all accounts with this tag (format: key:value)")
	}

	// Add a lightweight `version` subcommand so users and CI can run `keymaster version`.
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			// Re-resolve build info so the subcommand shows the same values
			resolvedVersion := version
			resolvedCommit := gitCommit
			resolvedDate := buildDate
			if info, ok := debug.ReadBuildInfo(); ok {
				if info.Main.Version != "" && info.Main.Version != "(devel)" {
					resolvedVersion = info.Main.Version
				}
				for _, s := range info.Settings {
					switch s.Key {
					case "vcs.revision":
						if s.Value != "" {
							resolvedCommit = s.Value
						}
					case "vcs.time":
						if s.Value != "" {
							resolvedDate = s.Value
						}
					}
				}
			}

			fmt.Printf("version: %s\n", resolvedVersion)
			fmt.Printf("commit: %s\n", resolvedCommit)
			if resolvedDate != "" {
				fmt.Printf("built: %s\n", resolvedDate)
			}
		},
	}

	// Add subcommands to the newly created root command.
	cmd.AddCommand(
		deployCmd,
		rotateKeyCmd,
		auditCmd,
		importCmd,
		trustHostCmd,
		exportSSHConfigCmd,
		dbMaintainCmd,
		backupCmd,
		restoreCmd,
		migrateCmd,
		decommissionCmd,
		versionCmd,
	)

	return cmd
}

// resolveBuildVersion computes the best-available version, commit and build
// date for the running binary. If `info` is nil, it reads build info from
// the runtime. This helper is separated to make unit testing straightforward.
func resolveBuildVersion(info *debug.BuildInfo) (versionOut, commitOut, dateOut string) {
	resolvedVersion := version
	resolvedCommit := gitCommit
	resolvedDate := buildDate

	var ok bool
	if info == nil {
		if infoLocal, found := debug.ReadBuildInfo(); found {
			info = infoLocal
			ok = true
		}
	} else {
		ok = true
	}

	if ok && info != nil {
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			resolvedVersion = info.Main.Version
		}
		// If Main doesn't contain the version (some build paths), try to
		// find our module in the dependencies and use that version.
		if (resolvedVersion == "dev" || resolvedVersion == "(devel)") && info.Deps != nil {
			for _, dep := range info.Deps {
				if dep.Path == "github.com/toeirei/keymaster" && dep.Version != "" {
					resolvedVersion = dep.Version
					break
				}
			}
		}

		for _, s := range info.Settings {
			switch s.Key {
			case "vcs.revision":
				if s.Value != "" {
					resolvedCommit = s.Value
				}
			case "vcs.time":
				if s.Value != "" {
					resolvedDate = s.Value
				}
			}
		}
	}

	// As a last resort, if no version was discovered, but a gitCommit was
	// provided via ldflags, show that to aid support.
	if resolvedVersion == "dev" && gitCommit != "dev" && gitCommit != "" {
		resolvedVersion = gitCommit
	}

	return resolvedVersion, resolvedCommit, resolvedDate
}

// deployCmd represents the 'deploy' command.
// It handles rendering the authorized_keys file from the database and deploying it
// to one or all managed accounts.
var deployCmd = &cobra.Command{
	Use:   "deploy [user@host]",
	Short: "Deploy authorized_keys to one or all hosts",
	Long: `Renders the authorized_keys file from the database state and deploys it.
If an account (user@host) is specified, deploys only to that account.
If no account is specified, deploys to all active accounts in the database.`,

	Args:    cobra.MaximumNArgs(1),
	PreRunE: setupDefaultServices,
	Run: func(cmd *cobra.Command, args []string) {
		// Build adapters for core facades
		st := &cliStoreAdapter{}
		dm := &cliDeployerManager{}

		var identifier *string
		if len(args) > 0 {
			s := args[0]
			identifier = &s
		}

		results, err := core.RunDeployCmd(cmd.Context(), st, dm, identifier, nil)
		if err != nil {
			log.Fatalf("%v", err)
		}
		// Print results similarly to previous behavior
		for _, r := range results {
			if r.Error != nil {
				fmt.Printf("%s\n", i18n.T("parallel_task.deploy_fail_message", r.Account.String(), r.Error))
			} else {
				fmt.Printf("%s\n", i18n.T("parallel_task.deploy_success_message", r.Account.String()))
			}
		}
	},
}

// rotateKeyCmd represents the 'rotate-key' command.
// It generates a new system key pair, saves it to the database as the new
// active key, and keeps the old key for transitioning hosts.
var rotateKeyCmd = &cobra.Command{
	Use:   "rotate-key",
	Short: "Rotates the active Keymaster system key",
	Long: `Generates a new ed25519 key pair, saves it to the database, and sets it as the active key.
The previous key is kept for accessing hosts that have not yet been updated.`,
	PreRunE: setupDefaultServices,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(i18n.T("rotate_key.cli_rotating"))
		passphrase := password
		if passphrase == "" {
			if term.IsTerminal(int(os.Stdin.Fd())) {
				fmt.Print(i18n.T("rotate_key.cli_password_prompt"))
				bytePassword, err := term.ReadPassword(int(os.Stdin.Fd()))
				if err != nil {
					log.Fatalf("%s", i18n.T("rotate_key.cli_error_read_password", err))
				}
				passphrase = string(bytePassword)
				fmt.Println()
			}
		}

		st := &cliStoreAdapter{}
		serial, err := core.RunRotateKeyCmd(cmd.Context(), &cliKeyGenerator{}, st, passphrase)
		if err != nil {
			log.Fatalf("%s", i18n.T("rotate_key.cli_error_save", err))
		}
		fmt.Printf("%s\n", i18n.T("rotate_key.cli_rotated_success", serial))
		fmt.Printf("%s\n", i18n.T("rotate_key.cli_deploy_reminder"))
	},
}

// auditCmd represents the 'audit' command.
// It connects to all active hosts to verify that their deployed authorized_keys
// file matches the configuration stored in the database, detecting any drift.
var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Audit hosts for configuration drift",
	Long: `Connects to all active hosts and compares the fully rendered, normalized authorized_keys content against the expected configuration from the database to detect drift.

Use --mode=serial to only verify the Keymaster header serial number on the remote host matches the account's last deployed serial (useful during staged rotations).`,
	PreRunE: setupDefaultServices,
	Run: func(cmd *cobra.Command, args []string) {
		st := &cliStoreAdapter{}
		dm := &cliDeployerManager{}
		results, err := core.RunAuditCmd(cmd.Context(), st, dm, auditMode, nil)
		if err != nil {
			log.Fatalf("%s", i18n.T("audit.cli_error_get_accounts", err))
		}
		for _, r := range results {
			if r.Error != nil {
				fmt.Printf("%s\n", i18n.T("parallel_task.audit_fail_message", r.Account.String(), r.Error))
			} else {
				fmt.Printf("%s\n", i18n.T("parallel_task.audit_success_message", r.Account.String()))
			}
		}
	},
}

// importCmd represents the 'import' command.
// It parses a standard authorized_keys file and adds the public keys
// found within it to the Keymaster database.
var importCmd = &cobra.Command{
	Use:     "import [authorized_keys_file]",
	Short:   "Import public keys from an authorized_keys file",
	Long:    `Reads a standard authorized_keys file and imports the public keys into the Keymaster database.`,
	Args:    cobra.ExactArgs(1), // Ensures we get exactly one file path
	PreRunE: setupDefaultServices,
	Run: func(cmd *cobra.Command, args []string) {
		filePath := args[0]
		fmt.Println(i18n.T("import.start", filePath))
		file, err := os.Open(filePath)
		if err != nil {
			log.Fatalf("%s", i18n.T("import.error_opening_file", err))
		}
		defer func() { _ = file.Close() }()

		km := db.DefaultKeyManager()
		scanner := bufio.NewScanner(file)
		imported, skipped := 0, 0
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			alg, keyData, comment, perr := sshkey.Parse(line)
			if perr != nil {
				fmt.Println("Skipping invalid key line")
				skipped++
				continue
			}
			if comment == "" {
				fmt.Println("Skipping key with empty comment")
				skipped++
				continue
			}
			if err := km.AddPublicKey(alg, keyData, comment, false, time.Time{}); err != nil {
				fmt.Printf("Skipping duplicate key (comment exists): %s\n", comment)
				skipped++
				continue
			}
			fmt.Printf("Imported key: %s\n", comment)
			imported++
		}
		if sErr := scanner.Err(); sErr != nil {
			log.Fatalf("%s", i18n.T("import.error_adding_key", sErr))
		}
		fmt.Printf("\nImport complete. Imported %d keys, skipped %d.\n", imported, skipped)
	},
}

// parallelTask defines a generic task to be executed in parallel across multiple
// accounts. It holds configuration for messaging, logging, and the core task
// function to be executed.
type parallelTask struct {
	name       string // e.g., "deployment", "audit"
	startMsg   string // e.g., "🚀 Starting deployment..."
	successMsg string // e.g., "✅ Successfully deployed to %s"
	failMsg    string // e.g., "💥 Failed to deploy to %s: %v"
	successLog string // e.g., "DEPLOY_SUCCESS"
	failLog    string // e.g., "DEPLOY_FAIL"
	taskFunc   func(model.Account) error
}

// trustHostCmd represents the 'trust-host' command.
// It facilitates the initial trust of a new host by fetching its public SSH key,
// displaying its fingerprint, and prompting the user to save it to the database
// as a known host.
var trustHostCmd = &cobra.Command{
	Use:   "trust-host <user@host>",
	Short: "Adds a host's public key to the list of known hosts",
	Long: `Connects to a host for the first time, retrieves its public key,
and prompts the user to save it to the database. This is a required
step before Keymaster can manage a new host.`,
	Args:    cobra.ExactArgs(1),
	PreRunE: setupDefaultServices,
	Run: func(cmd *cobra.Command, args []string) {
		target := args[0]
		var hostname string
		if strings.Contains(target, "@") {
			parts := strings.SplitN(target, "@", 2)
			hostname = parts[1]
		} else {
			hostname = target
		}
		canonicalHost := deploy.CanonicalizeHostPort(hostname)

		fmt.Printf("Attempting to retrieve host key from %s…\n", canonicalHost)
		dm := &cliDeployerManager{}
		keyStr, err := dm.GetRemoteHostKey(canonicalHost)
		if err != nil {
			log.Fatalf("%s", i18n.T("trust_host.error_get_key", err))
		}
		// Parse to compute fingerprint
		pubKey, _, _, _, perr := ssh.ParseAuthorizedKey([]byte(keyStr))
		if perr == nil {
			fmt.Printf("The authenticity of host '%s' can't be established.\n", canonicalHost)
			fmt.Printf("Key fingerprint: %s\n", ssh.FingerprintSHA256(pubKey))
			if warn := sshkey.CheckHostKeyAlgorithm(pubKey); warn != "" {
				fmt.Println(warn)
			}
		}

		// prompt user
		ans := promptForConfirmation("Are you sure you want to continue connecting (yes/no)? ")
		if ans != "yes" && ans != "y" {
			fmt.Println("Cancelled.")
			return
		}
		st := &cliStoreAdapter{}
		if err := st.AddKnownHostKey(canonicalHost, keyStr); err != nil {
			log.Fatalf("%s", i18n.T("trust_host.error_get_key", err))
		}
		fmt.Printf("Warning: Permanently added '%s' (type ) to the list of known hosts.\n", canonicalHost)
	},
}

// runParallelTasks executes a given task concurrently for a list of accounts.
// It uses a wait group to manage goroutines and a channel to collect results,
// printing status messages as tasks complete.
func runParallelTasks(accounts []model.Account, task parallelTask) {
	if len(accounts) == 0 {
		fmt.Println(i18n.T("parallel_task.no_accounts", task.name))
		return
	}

	var wg sync.WaitGroup
	results := make(chan string, len(accounts)) // This channel will now carry pre-formatted i18n strings

	// task.startMsg is already formatted by i18n.T, so just print it.
	fmt.Println(task.startMsg)

	for _, acc := range accounts {
		wg.Add(1)
		go func(account model.Account) {
			defer wg.Done()
			err := task.taskFunc(account)
			details := fmt.Sprintf("account: %s", account.String())
			if err != nil {
				results <- fmt.Sprintf(task.failMsg, account.String(), err.Error())
				_ = logAction(task.failLog, fmt.Sprintf("%s, error: %v", details, err))
			} else {
				results <- fmt.Sprintf(task.successMsg, account.String()) // Pass account string as arg
				_ = logAction(task.successLog, details)
			}
		}(acc)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	for res := range results {
		fmt.Println(res)
	}
	// Use unicode-aware titlecasing
	titleCaser := cases.Title(language.Und)
	fmt.Println("\n" + i18n.T("parallel_task.complete_message", titleCaser.String(task.name)))
}

// audit implementations moved to internal/deploy/audit.go

// exportSSHConfigCmd represents the 'export-ssh-client-config' command.
// It generates an SSH config file from all active accounts in the database.
var exportSSHConfigCmd = &cobra.Command{
	Use:   "export-ssh-client-config [output-file]",
	Short: "Export SSH config from active accounts",
	Long: `Generates an SSH config file with Host entries for all active accounts.
If no output file is specified, prints to stdout.
Each account with a label will use the label as the Host alias.`,
	Args:    cobra.MaximumNArgs(1),
	PreRunE: setupDefaultServices,
	Run: func(cmd *cobra.Command, args []string) {
		st := &cliStoreAdapter{}
		out, err := core.RunExportSSHConfigCmd(cmd.Context(), st)
		if err != nil {
			log.Fatalf("%s", i18n.T("export_ssh_config.error_get_accounts", err))
		}
		if out == "" {
			fmt.Println(i18n.T("export_ssh_config.no_accounts"))
			return
		}
		if len(args) > 0 {
			outputFile := args[0]
			if err := os.WriteFile(outputFile, []byte(out), 0644); err != nil {
				log.Fatalf("%s", i18n.T("export_ssh_config.error_write_file", err))
			}
			fmt.Printf("%s\n", i18n.T("export_ssh_config.success", outputFile))
		} else {
			fmt.Print(out)
		}
	},
}

// dbMaintainCmd runs database maintenance tasks for the configured database.
var dbMaintainCmd = &cobra.Command{
	Use:     "db-maintain",
	Short:   "Run database maintenance (VACUUM/OPTIMIZE) for the configured DB",
	Long:    `Runs engine-specific maintenance tasks (VACUUM, OPTIMIZE TABLE, PRAGMA optimize).`,
	PreRunE: setupDefaultServices,
	Run: func(cmd *cobra.Command, args []string) {
		skipIntegrity, _ := cmd.Flags().GetBool("skip-integrity")
		timeoutSec, _ := cmd.Flags().GetInt("timeout")
		dsn := appConfig.Database.Dsn
		dbType := appConfig.Database.Type
		if skipIntegrity {
			fmt.Println("Skipping integrity_check may speed up maintenance on large databases")
		}
		maint := &cliDBMaintainer{}
		if timeoutSec > 0 {
			done := make(chan error, 1)
			go func() {
				done <- core.RunDBMaintenance(cmd.Context(), maint, dbType, dsn, core.DBMaintenanceOptions{SkipIntegrity: skipIntegrity, Timeout: time.Duration(timeoutSec) * time.Second})
			}()
			select {
			case err := <-done:
				if err != nil {
					fmt.Printf("Maintenance failed: %v\n", err)
					os.Exit(1)
				}
				fmt.Println("Maintenance completed successfully")
			case <-time.After(time.Duration(timeoutSec) * time.Second):
				fmt.Println("Maintenance timed out")
				os.Exit(2)
			}
			return
		}
		if err := core.RunDBMaintenance(cmd.Context(), maint, dbType, dsn, core.DBMaintenanceOptions{SkipIntegrity: skipIntegrity}); err != nil {
			fmt.Printf("Maintenance failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Maintenance completed successfully")
	},
}

// promptForConfirmation displays a prompt and reads a line from stdin.
func promptForConfirmation(prompt string) string {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	return strings.TrimSpace(strings.ToLower(answer))
}

// runDeploymentForAccount is a simple wrapper for the CLI to match the
// signature required by runParallelTasks. It calls the centralized
// deployment logic from the deploy package.
func runDeploymentForAccount(account model.Account) error {
	return runDeploymentFunc(account)
}

// runDeploymentFunc is a package-level variable so tests can inject a mock
// implementation. By default it calls into the deploy package with CLI mode.
var runDeploymentFunc = func(account model.Account) error {
	return deploy.RunDeploymentForAccount(account, false)
}

// decommissionCmd represents the 'decommission' command.
// It removes SSH access by cleaning up authorized_keys files and deleting accounts from the database.
var decommissionCmd = &cobra.Command{
	Use:   "decommission [account-identifier]",
	Short: "Decommission one or more accounts by removing SSH access and deleting from database",
	Long: `Decommissions accounts by first removing their authorized_keys files from remote hosts,
then deleting them from the database. This ensures clean removal of SSH access.

Account can be identified by:
- Account ID (e.g., "5")
- User@host format (e.g., "deploy@server-01")
- Label (e.g., "prod-web-01")

If no account is specified, you will be prompted to select from a list.

Use --tag to decommission all accounts with specific tags (e.g., --tag env:staging).`,
	Args:    cobra.MaximumNArgs(1),
	PreRunE: setupDefaultServices,
	Run: func(cmd *cobra.Command, args []string) {
		// Parse flags
		// TODO do it better
		skipRemote, _ := cmd.Flags().GetBool("skip-remote")
		keepFile, _ := cmd.Flags().GetBool("keep-file")
		force, _ := cmd.Flags().GetBool("force")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		tagFilter, _ := cmd.Flags().GetString("tag")

		options := deploy.DecommissionOptions{
			SkipRemoteCleanup: skipRemote,
			KeepFile:          keepFile,
			Force:             force,
			DryRun:            dryRun,
		}

		// Get active system key
		systemKey, err := db.GetActiveSystemKey()
		if err != nil {
			log.Fatalf("Error getting active system key: %v", err)
		}
		if systemKey == nil {
			log.Fatal("No active system key found. Run 'keymaster rotate-key' to generate one.")
		}

		// Get all accounts
		allAccounts, err := db.GetAllAccounts()
		if err != nil {
			log.Fatalf("Error getting accounts: %v", err)
		}

		var targetAccounts []model.Account

		if tagFilter != "" {
			// Filter accounts by tag
			for _, acc := range allAccounts {
				if strings.Contains(acc.Tags, tagFilter) {
					targetAccounts = append(targetAccounts, acc)
				}
			}
			if len(targetAccounts) == 0 {
				fmt.Printf("No accounts found with tag: %s\n", tagFilter)
				return
			}
			fmt.Printf("Found %d accounts with tag '%s':\n", len(targetAccounts), tagFilter)
			for _, acc := range targetAccounts {
				fmt.Printf("  - %s\n", acc.String())
			}
		} else if len(args) > 0 {
			// Find specific account
			target := args[0]
			account, err := findAccountByIdentifier(target, allAccounts)
			if err != nil {
				log.Fatalf("Error finding account: %v", err)
			}
			targetAccounts = []model.Account{*account}
			fmt.Printf("Selected account: %s\n", account.String())
		} else {
			// No specific target - show interactive selection
			fmt.Println("Available accounts:")
			for i, acc := range allAccounts {
				status := "active"
				if !acc.IsActive {
					status = "inactive"
				}
				fmt.Printf("  %d: %s (%s)\n", i+1, acc.String(), status)
			}
			fmt.Print("Enter account number to decommission (or 'q' to quit): ")

			reader := bufio.NewReader(os.Stdin)
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)

			if input == "q" || input == "quit" {
				fmt.Println("Cancelled.")
				return
			}

			var selection int
			if _, err := fmt.Sscanf(input, "%d", &selection); err != nil || selection < 1 || selection > len(allAccounts) {
				log.Fatal("Invalid selection")
			}

			targetAccounts = []model.Account{allAccounts[selection-1]}
			fmt.Printf("Selected account: %s\n", allAccounts[selection-1].String())
		}

		// Confirmation prompt (unless dry-run)
		if !dryRun && !force {
			fmt.Printf("\nWARNING: This will decommission %d account(s) by:\n", len(targetAccounts))
			if !skipRemote {
				if keepFile {
					fmt.Println("  1. Removing Keymaster-managed content from authorized_keys files")
				} else {
					fmt.Println("  1. Removing authorized_keys files (with backup)")
				}
			}
			fmt.Println("  2. Deleting accounts from the database")
			fmt.Print("\nDo you want to continue? (yes/no): ")

			reader := bufio.NewReader(os.Stdin)
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(strings.ToLower(input))

			if input != "yes" && input != "y" {
				fmt.Println("Operation cancelled.")
				return
			}
		}

		// Process accounts
		if len(targetAccounts) == 1 {
			// Single account
			account := targetAccounts[0]
			fmt.Printf("Decommissioning account: %s\n", account.String())
			result := deploy.DecommissionAccount(account, systemKey.PrivateKey, options)
			fmt.Printf("Result: %s\n", result.String())
		} else {
			// Multiple accounts - use bulk operation
			fmt.Printf("Decommissioning %d accounts...\n", len(targetAccounts))
			results := deploy.BulkDecommissionAccounts(targetAccounts, systemKey.PrivateKey, options)

			// Summary
			successful := 0
			failed := 0
			skipped := 0

			for _, result := range results {
				if result.Skipped {
					skipped++
				} else if result.DatabaseDeleteError != nil {
					failed++
				} else {
					successful++
				}
			}

			fmt.Printf("\nSummary: %d successful, %d failed, %d skipped\n", successful, failed, skipped)

			if failed > 0 {
				fmt.Println("\nFailed operations:")
				for _, result := range results {
					if !result.Skipped && result.DatabaseDeleteError != nil {
						fmt.Printf("  - %s\n", result.String())
					}
				}
			}
		}
	},
}

// findAccountByIdentifier finds an account by ID, user@host, or label
func findAccountByIdentifier(identifier string, accounts []model.Account) (*model.Account, error) {
	// Try to parse as account ID first
	var id int
	if n, err := fmt.Sscanf(identifier, "%d", &id); n == 1 && err == nil {
		for _, acc := range accounts {
			if acc.ID == id {
				return &acc, nil
			}
		}
		return nil, fmt.Errorf("no account found with ID: %s", identifier)
	}

	// Try user@host format
	if strings.Contains(identifier, "@") {
		normalizedTarget := strings.ToLower(identifier)
		for _, acc := range accounts {
			accountIdentifier := fmt.Sprintf("%s@%s", acc.Username, acc.Hostname)
			if strings.ToLower(accountIdentifier) == normalizedTarget {
				return &acc, nil
			}
		}
		return nil, fmt.Errorf("no account found with identifier: %s", identifier)
	}

	// Try label
	for _, acc := range accounts {
		if strings.EqualFold(acc.Label, identifier) {
			return &acc, nil
		}
	}

	return nil, fmt.Errorf("no account found with identifier: %s (try ID, user@host, or label)", identifier)
}

// restoreCmd represents the 'restore' command.
// It restores the database from a compressed JSON backup file.
var restoreCmd = &cobra.Command{
	Use:   "restore <backup-file.zst>",
	Short: "Restore the database from a compressed JSON backup",
	Long: `Restores the entire Keymaster database from a Zstandard-compressed JSON backup file.
By default, this command performs a non-destructive "integration" restore, only adding
data that does not already exist.
 
To perform a full, destructive restore that WIPES all existing data before importing, use the --full flag.
WARNING: The --full flag is destructive and not reversible.
This command is intended for disaster recovery or for migrating between
database backends (e.g., from SQLite to PostgreSQL).

Example (Integrate):
  keymaster restore ./keymaster-backup-2025-10-26.json.zst

Example (Full Restore):
  keymaster restore --full ./keymaster-backup-2025-10-26.json.zst`,
	Args:    cobra.ExactArgs(1),
	PreRunE: setupDefaultServices, // This was correct, just confirming.
	Run: func(cmd *cobra.Command, args []string) {
		inputFile := args[0]
		fmt.Println(i18n.T("restore.cli_starting", inputFile))
		f, err := os.Open(inputFile)
		if err != nil {
			log.Fatalf("%s", i18n.T("restore.cli_error_read", err))
		}
		defer f.Close()
		if err := core.RunRestoreCmd(cmd.Context(), f, core.RestoreOptions{Full: fullRestore}, &cliStoreAdapter{}); err != nil {
			log.Fatalf("%s", i18n.T("restore.cli_error_import", err))
		}
		fmt.Println(i18n.T("restore.cli_success"))
	},
}

// readCompressedBackup handles reading and decoding a zstd-compressed JSON backup file.
func readCompressedBackup(filename string) (*model.BackupData, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("could not open file: %w", err)
	}
	defer func() { _ = file.Close() }()

	zstdReader, err := zstd.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("could not create zstd reader: %w", err)
	}
	defer zstdReader.Close()

	var backupData model.BackupData
	if err := json.NewDecoder(zstdReader).Decode(&backupData); err != nil {
		return nil, fmt.Errorf("could not decode json from zstd reader: %w", err)
	}

	return &backupData, nil
}

// backupCmd represents the 'backup' command.
// It dumps all data from the database into a single JSON file.
var backupCmd = &cobra.Command{ //
	Use:   "backup [output-file]", //
	Short: "Create a compressed (zstd) JSON backup of the database",
	Long: `Dumps the entire contents of the Keymaster database (accounts, keys, audit logs, etc.)
into a single, Zstandard-compressed JSON file.

If an output file is specified, '.zst' will be appended to the name if it's not already present.
If no output file is specified, a default filename 'keymaster-backup-YYYY-MM-DD.json.zst' is used.

This file can be used for disaster recovery or for migrating to a different database backend.

Examples:
  # Backup to a default file (e.g., keymaster-backup-2025-10-26.json.zst)
  keymaster backup

  # Backup to a specific file
  keymaster backup my-backup.json`, // .zst will be appended
	Args:    cobra.MaximumNArgs(1),
	PreRunE: setupDefaultServices,
	Run: func(cmd *cobra.Command, args []string) {
		var outputFile string
		if len(args) == 0 {
			outputFile = fmt.Sprintf("keymaster-backup-%s.json.zst", time.Now().Format("2006-01-02"))
		} else {
			outputFile = args[0]
			if !strings.HasSuffix(outputFile, ".zst") {
				outputFile += ".zst"
			}
		}
		fmt.Println(i18n.T("backup.cli_starting"))
		st := &cliStoreAdapter{}
		data, err := core.RunBackupCmd(cmd.Context(), st)
		if err != nil {
			log.Fatalf("%s", i18n.T("backup.cli_error_export", err))
		}
		outf, err := os.Create(outputFile)
		if err != nil {
			log.Fatalf("%s", i18n.T("backup.cli_error_write", err))
		}
		defer outf.Close()
		if err := core.RunWriteBackupCmd(cmd.Context(), data, outf); err != nil {
			log.Fatalf("%s", i18n.T("backup.cli_error_write", err))
		}
		fmt.Println(i18n.T("backup.cli_success", outputFile))
	},
}

// writeCompressedBackup handles the process of writing the backup data to a zstd-compressed file.
// It streams the JSON encoding directly to the gzip writer for memory efficiency.
func writeCompressedBackup(filename string, data *model.BackupData) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("could not create file: %w", err)
	}
	defer func() { _ = file.Close() }()

	zstdWriter, err := zstd.NewWriter(file)
	if err != nil {
		return fmt.Errorf("could not create zstd writer: %w", err)
	}
	defer func() { _ = zstdWriter.Close() }()

	encoder := json.NewEncoder(zstdWriter)
	encoder.SetIndent("", "  ") // Pretty-print the JSON inside the compressed file

	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("could not encode json to zstd writer: %w", err)
	}

	return nil
}

// TODO logic may be redundant and/or belongs into the db package
// migrateCmd represents the 'migrate' command.
var migrateCmd = &cobra.Command{
	Use:   "migrate --type <db-type> --dsn <target-dsn>",
	Short: "Migrate data from the current database to a new one",
	Long: `Performs a database migration by exporting all data from the current database
(configured in .keymaster.yaml) and importing it into a new target database.

This command automates the following steps:
1. Exports data from the source database in-memory.
2. Connects to the target database specified by --type and --dsn.
3. Applies all necessary database schema migrations to the target.
4. Performs a full, destructive restore into the target database.

Example:
  keymaster migrate --type postgres --dsn "host=localhost user=keymaster dbname=keymaster"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Viper has already been configured by PreRunE, so we can directly access the values.
		targetType := viper.GetString("database.type")
		targetDsn := viper.GetString("database.dsn")
		if targetType == "" || targetDsn == "" {
			log.Fatalf("%s", i18n.T("migrate.cli_error_flags"))
		}
		fmt.Println(i18n.T("migrate.cli_starting_backup"))
		st := &cliStoreAdapter{}
		factory := &cliStoreFactory{}
		if err := core.RunMigrateCmd(cmd.Context(), factory, st, targetType, targetDsn); err != nil {
			log.Fatalf("%s", i18n.T("migrate.cli_error_backup", err))
		}
		fmt.Println(i18n.T("migrate.cli_success"))
		fmt.Println(i18n.T("migrate.cli_next_steps"))
		return nil
	},
}

// initTargetDB is a helper function that initializes a new database connection
// for the migration target, runs migrations, and returns a Store instance.
// It is a simplified, one-off version of db.InitDB that does not affect the
// global `store` variable.
func initTargetDB(db_type, db_dsn string) (db.Store, error) {
	// Use the DB package helper to create a store from the DSN. This hides
	// direct *sql.DB handling and ensures migrations are applied.
	if db_type == "sqlite" && !strings.Contains(db_dsn, "_busy_timeout") {
		db_dsn += "?_busy_timeout=5000"
	}
	s, err := db.NewStoreFromDSN(db_type, db_dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize target store: %w", err)
	}
	return s, nil
}
