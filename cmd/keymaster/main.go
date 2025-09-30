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
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql" // Blank import for migrate command
	_ "github.com/jackc/pgx/v5/stdlib" // Blank import for migrate command
	"github.com/klauspost/compress/zstd"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/toeirei/keymaster/internal/bootstrap"
	internalkey "github.com/toeirei/keymaster/internal/crypto/ssh"
	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/deploy"
	"github.com/toeirei/keymaster/internal/i18n"
	"github.com/toeirei/keymaster/internal/model"
	"github.com/toeirei/keymaster/internal/sshkey"
	"github.com/toeirei/keymaster/internal/tui"
	"golang.org/x/crypto/ssh"
)

var version = "dev" // this will be set by the linker
var cfgFile string
var auditMode string // audit mode flag: "strict" (default) or "serial"
var fullRestore bool // Flag for the restore command

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

	if err := rootCmd.Execute(); err != nil {
		// The error is already printed by Cobra on failure.
		os.Exit(1)
	}
}

var rootCmd *cobra.Command

func init() {
	rootCmd = newRootCmd()
}

// newRootCmd creates and configures a new root cobra command.
// This function is used to create the main application command as well as
// fresh instances for isolated testing.
func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "keymaster",
		Short: "Keymaster is a lightweight, agentless SSH access manager.",
		Long: `Keymaster centralizes control of authorized_keys files.
Instead of scattering user keys everywhere, Keymaster plants a single
system key per account and uses it as a foothold to rewrite and
version-control access. A database becomes the source of truth.

Running without a subcommand will launch the interactive TUI.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Initialize the database for all commands.
			// Viper has already read the config by this point.
			dbType := viper.GetString("database.type")
			dsn := viper.GetString("database.dsn")
			if err := db.InitDB(dbType, dsn); err != nil {
				return errors.New(i18n.T("config.error_init_db", err))
			}

			// Recover from any previous crashes
			if err := bootstrap.RecoverFromCrash(); err != nil {
				log.Printf("Bootstrap recovery error: %v", err)
			}

			// Start background session reaper
			bootstrap.StartSessionReaper()

			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			// The database is already initialized by PersistentPreRunE.
			i18n.Init(viper.GetString("language")) // Initialize i18n for the TUI
			tui.Run()
		},
	}

	// Add subcommands to the newly created root command.
	cmd.AddCommand(deployCmd)
	cmd.AddCommand(rotateKeyCmd)
	cmd.AddCommand(auditCmd)
	cmd.AddCommand(importCmd)
	cmd.AddCommand(trustHostCmd)
	cmd.AddCommand(exportSSHConfigCmd)
	cmd.AddCommand(backupCmd)
	cmd.AddCommand(restoreCmd)
	cmd.AddCommand(migrateCmd)

	// Set version
	cmd.Version = version

	// Initialize config on every command execution.
	cobra.OnInitialize(func() { _ = initConfig() })

	// Set defaults in viper. These are used if not set in the config file or by flags.
	viper.SetDefault("database.type", "sqlite")
	viper.SetDefault("database.dsn", "./keymaster.db")
	// Define flags
	cmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.keymaster.yaml or ./keymaster.yaml)")
	cmd.PersistentFlags().String("db-type", "sqlite", "Database type (e.g., sqlite, postgres)")
	cmd.PersistentFlags().String("db-dsn", "./keymaster.db", "Database connection string (DSN)")
	cmd.PersistentFlags().String("lang", "en", `TUI language ("en", "de")`)

	// Bind flags to viper
	viper.BindPFlag("database.type", cmd.PersistentFlags().Lookup("db-type"))
	viper.BindPFlag("database.dsn", cmd.PersistentFlags().Lookup("db-dsn"))
	viper.BindPFlag("language", cmd.PersistentFlags().Lookup("lang"))

	return cmd
}

// initConfig reads in a configuration file and environment variables.
// It uses Viper to search for a config file (e.g., .keymaster.yaml) in the home
// and current directories. If a config file is not found, it attempts to create
// a default one. It also binds environment variables prefixed with "KEYMASTER".
func initConfig() error {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory and current directory with name ".keymaster" (without extension).
		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName(".keymaster")
	}

	viper.SetEnvPrefix("KEYMASTER")
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// We only do this if no config file was found and none was specified via flag.
			// We'll attempt to write a default config to the current directory if cfgFile is empty.
			if cfgFile == "" {
				const defaultConfigPath = ".keymaster.yaml"

				defaultContent := `# Keymaster configuration file.
# This file is automatically generated with default values.
# You can modify these settings to configure Keymaster.

database:
  # The type of database to use. Supported values: "sqlite", "postgres", "mysql".
  # Note: PostgreSQL and MySQL support is experimental.
  type: sqlite

  # The Data Source Name (DSN) for the database connection.
  # For SQLite, this is the path to the database file.
  dsn: ./keymaster.db

# The default language for the TUI. Supported: "en", "de".
language: en

# Example for future PostgreSQL configuration:
# database:
#   type: postgres
#   dsn: "host=localhost user=keymaster password=secret dbname=keymaster port=5432 sslmode=disable"

# Example for future MySQL configuration:
# database:
#   type: mysql
#   dsn: "keymaster:password@tcp(127.0.0.1:3306)/keymaster?parseTime=true"
`
				// If writing fails (e.g., due to permissions), we don't treat it as a
				// fatal error. The app will simply run with the default values set in memory.
				if err := os.WriteFile(defaultConfigPath, []byte(defaultContent), 0644); err == nil {
					// Return a specific error/message that can be handled by the caller.
					// We also re-read the config we just wrote to ensure viper is in a clean state.
					_ = viper.ReadInConfig()
					// The message is useful for the CLI user, but for tests, returning nil is cleaner.
					return nil
				}
			}
		} else {
			// The config file was found but was malformed or unreadable.
			// We return the error but don't exit, allowing the app to proceed with defaults.
			return fmt.Errorf("error reading config file: %w", err)
		}
	}
	return nil
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
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// DB is initialized in PersistentPreRunE.
		i18n.Init(viper.GetString("language")) // Initialize i18n for CLI output
		allAccounts, err := db.GetAllActiveAccounts()
		if err != nil {
			log.Fatalf("Error getting accounts: %v", err)
		}

		var targetAccounts []model.Account
		if len(args) > 0 {
			target := args[0]
			found := false
			// Normalize the target account string for comparison.
			normalizedTarget := strings.ToLower(target)
			for _, acc := range allAccounts {
				// Compare against a similarly normalized account string from the database.
				accountIdentifier := fmt.Sprintf("%s@%s", acc.Username, acc.Hostname)
				if strings.ToLower(accountIdentifier) == normalizedTarget {
					targetAccounts = append(targetAccounts, acc)
					found = true
					break
				}
			}
			if !found {
				log.Fatalf("%s", i18n.T("deploy.cli_account_not_found", target))
			}
		} else {
			targetAccounts = allAccounts
		}

		deployTask := parallelTask{
			name:       "deployment",
			startMsg:   i18n.T("parallel_task.start_message", "deployment", len(targetAccounts)),
			successMsg: i18n.T("parallel_task.deploy_success_message"),
			failMsg:    i18n.T("parallel_task.deploy_fail_message"),
			successLog: "CLI_DEPLOY_SUCCESS",
			failLog:    "DEPLOY_FAIL",
			taskFunc:   runDeploymentForAccount,
		}

		runParallelTasks(targetAccounts, deployTask)
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
	Run: func(cmd *cobra.Command, args []string) {
		i18n.Init(viper.GetString("language")) // Initialize i18n for CLI output
		fmt.Println(i18n.T("rotate_key.cli_rotating"))
		// DB is initialized in PersistentPreRunE.
		publicKeyString, privateKeyString, err := internalkey.GenerateAndMarshalEd25519Key("keymaster-system-key")
		if err != nil {
			log.Fatalf("%s", i18n.T("rotate_key.cli_error_generate", err))
		}

		serial, err := db.RotateSystemKey(publicKeyString, privateKeyString)
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
	Run: func(cmd *cobra.Command, args []string) {
		// DB is initialized in PersistentPreRunE.
		i18n.Init(viper.GetString("language")) // Initialize i18n for CLI output
		accounts, err := db.GetAllActiveAccounts()
		if err != nil {
			log.Fatalf("%s", i18n.T("audit.cli_error_get_accounts", err))
		}

		// Select audit function based on mode
		var auditFunc func(model.Account) error
		switch strings.ToLower(strings.TrimSpace(auditMode)) {
		case "serial":
			auditFunc = deploy.AuditAccountSerial
		case "strict", "":
			auditFunc = deploy.AuditAccountStrict
		default:
			log.Fatalf("invalid audit mode: %s (use 'strict' or 'serial')", auditMode)
		}

		auditTask := parallelTask{
			name:       "audit",
			startMsg:   i18n.T("parallel_task.start_message", "audit", len(accounts)),
			successMsg: i18n.T("parallel_task.audit_success_message"),
			failMsg:    i18n.T("parallel_task.audit_fail_message"),
			successLog: "CLI_AUDIT_SUCCESS",
			failLog:    "CLI_AUDIT_FAIL",
			taskFunc:   auditFunc,
		}

		runParallelTasks(accounts, auditTask)
	},
}

func init() {
	// Attach flags after auditCmd is defined
	auditCmd.Flags().StringVarP(&auditMode, "mode", "m", "strict", "Audit mode: 'strict' (full file comparison) or 'serial' (header serial only)")
	restoreCmd.Flags().BoolVar(&fullRestore, "full", false, "Perform a full, destructive restore (wipes all existing data first)")
	migrateCmd.Flags().String("type", "", "The target database type (sqlite, postgres, mysql)")
	migrateCmd.Flags().String("dsn", "", "The DSN for the target database")
}

// importCmd represents the 'import' command.
// It parses a standard authorized_keys file and adds the public keys
// found within it to the Keymaster database.
var importCmd = &cobra.Command{
	Use:   "import [authorized_keys_file]",
	Short: "Import public keys from an authorized_keys file",
	Long:  `Reads a standard authorized_keys file and imports the public keys into the Keymaster database.`,
	Args:  cobra.ExactArgs(1), // Ensures we get exactly one file path
	Run: func(cmd *cobra.Command, args []string) {
		filePath := args[0]
		i18n.Init(viper.GetString("language")) // Initialize i18n for CLI output
		// DB and i18n are initialized in PersistentPreRunE.
		fmt.Println(i18n.T("import.start", filePath))

		file, err := os.Open(filePath)
		if err != nil {
			log.Fatalf("%s", i18n.T("import.error_opening_file", err))
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		importedCount := 0
		skippedCount := 0

		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())

			// Skip empty lines or comments
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			alg, keyData, comment, err := sshkey.Parse(line)
			if err != nil {
				fmt.Printf("%s\n", i18n.T("import.skip_invalid_line", err))
				skippedCount++
				continue
			}

			if comment == "" {
				fmt.Printf("%s\n", i18n.T("import.skip_empty_comment", string(keyData)))
				skippedCount++
				continue
			}

			if err := db.AddPublicKey(alg, keyData, comment, false); err != nil {
				// Check if the error is a unique constraint violation. This makes the CLI
				// import behave consistently with the TUI remote import.
				// The exact error string can vary between DB drivers.
				if err == db.ErrDuplicate || strings.Contains(strings.ToLower(err.Error()), "unique constraint") {
					fmt.Printf("%s\n", i18n.T("import.skip_duplicate", comment))
				} else {
					fmt.Printf("%s\n", i18n.T("import.error_adding_key", comment, err.Error()))
				}
				skippedCount++
				continue
			}

			fmt.Printf("%s\n", i18n.T("import.imported_key", comment))
			importedCount++
		}

		fmt.Printf("\n%s\n", i18n.T("import.summary", importedCount, skippedCount))
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
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// DB is initialized in PersistentPreRunE.
		i18n.Init(viper.GetString("language")) // Initialize i18n for CLI output
		target := args[0]
		var hostname string
		if strings.Contains(target, "@") {
			parts := strings.SplitN(target, "@", 2)
			hostname = parts[1]
		} else {
			hostname = target // Assume the whole string is the hostname if no '@'
		}
		// Always use host:port form with default :22
		canonicalHost := deploy.CanonicalizeHostPort(hostname)

		fmt.Println(i18n.T("trust_host.retrieving_key", canonicalHost))
		key, err := deploy.GetRemoteHostKey(canonicalHost)
		if err != nil {
			log.Fatalf("%s", i18n.T("trust_host.error_get_key", err))
		}

		fingerprint := ssh.FingerprintSHA256(key) // Use standard ssh package
		fmt.Printf("\n%s\n", i18n.T("trust_host.authenticity_warning_1", canonicalHost))
		fmt.Printf("%s\n", i18n.T("trust_host.authenticity_warning_2", key.Type(), fingerprint))

		if warning := sshkey.CheckHostKeyAlgorithm(key); warning != "" {
			fmt.Printf("\n%s\n", warning)
		}

		answer := promptForConfirmation(i18n.T("trust_host.confirm_prompt"))

		if answer != "yes" {
			fmt.Println(i18n.T("trust_host.not_trusted_abort"))
			os.Exit(1)
		}

		keyStr := string(ssh.MarshalAuthorizedKey(key)) // Use standard ssh package
		if err := db.AddKnownHostKey(canonicalHost, keyStr); err != nil {
			log.Fatalf("%s", i18n.T("trust_host.error_save_key", err))
		}

		fmt.Printf("%s\n", i18n.T("trust_host.added_success", canonicalHost, key.Type()))
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
				_ = db.LogAction(task.failLog, fmt.Sprintf("%s, error: %v", details, err))
			} else {
				results <- fmt.Sprintf(task.successMsg, account.String()) // Pass account string as arg
				_ = db.LogAction(task.successLog, details)
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
	fmt.Println("\n" + i18n.T("parallel_task.complete_message", strings.Title(task.name)))
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
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// DB is initialized in PersistentPreRunE.
		i18n.Init(viper.GetString("language")) // Initialize i18n for CLI output
		accounts, err := db.GetAllActiveAccounts()
		if err != nil {
			log.Fatalf("%s", i18n.T("export_ssh_config.error_get_accounts", err))
		}

		if len(accounts) == 0 {
			fmt.Println(i18n.T("export_ssh_config.no_accounts"))
			return
		}

		var output strings.Builder
		output.WriteString("# " + i18n.T("export_ssh_config.header") + "\n")
		output.WriteString(fmt.Sprintf("# %s: %s\n\n", i18n.T("export_ssh_config.date"), time.Now().Format("2006-01-02 15:04:05")))

		for _, account := range accounts {
			// Use label as host alias if available, otherwise use username@hostname
			hostAlias := account.Label
			if hostAlias == "" {
				hostAlias = fmt.Sprintf("%s-%s", account.Username, strings.ReplaceAll(account.Hostname, ".", "-"))
			}

			output.WriteString(fmt.Sprintf("# %s\n", account.String()))
			output.WriteString(fmt.Sprintf("Host %s\n", hostAlias))
			output.WriteString(fmt.Sprintf("    HostName %s\n", account.Hostname))
			output.WriteString(fmt.Sprintf("    User %s\n", account.Username))

			// Parse hostname for port (supports IPv4/IPv6 and names)
			_, port, _ := deploy.ParseHostPort(account.Hostname)
			if port == "" {
				port = "22"
			}
			if port != "22" {
				output.WriteString(fmt.Sprintf("    Port %s\n", port))
			}

			output.WriteString("\n")
		}

		// Output to file or stdout
		if len(args) > 0 {
			outputFile := args[0]
			if err := os.WriteFile(outputFile, []byte(output.String()), 0644); err != nil {
				log.Fatalf("%s", i18n.T("export_ssh_config.error_write_file", err))
			}
			fmt.Printf("%s\n", i18n.T("export_ssh_config.success", outputFile))
		} else {
			fmt.Print(output.String())
		}
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
	return deploy.RunDeploymentForAccount(account, false)
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
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		inputFile := args[0]
		i18n.Init(viper.GetString("language"))

		fmt.Println(i18n.T("restore.cli_starting", inputFile))

		backupData, err := readCompressedBackup(inputFile)
		if err != nil {
			log.Fatalf("%s", i18n.T("restore.cli_error_read", err))
		}

		if fullRestore {
			// Destructive, full restore path. No confirmation for CLI operations.
			err = db.ImportDataFromBackup(backupData)
		} else {
			// Default, non-destructive integration path
			err = db.IntegrateDataFromBackup(backupData)
		}

		if err != nil {
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
	defer file.Close()

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
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		i18n.Init(viper.GetString("language"))

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

		backupData, err := db.ExportDataForBackup()
		if err != nil {
			log.Fatalf("%s", i18n.T("backup.cli_error_export", err))
		}

		if err := writeCompressedBackup(outputFile, backupData); err != nil {
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
	defer file.Close()

	zstdWriter, err := zstd.NewWriter(file)
	if err != nil {
		return fmt.Errorf("could not create zstd writer: %w", err)
	}
	defer zstdWriter.Close()

	encoder := json.NewEncoder(zstdWriter)
	encoder.SetIndent("", "  ") // Pretty-print the JSON inside the compressed file

	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("could not encode json to zstd writer: %w", err)
	}

	return nil
}

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
	Run: func(cmd *cobra.Command, args []string) {
		i18n.Init(viper.GetString("language"))

		targetType, _ := cmd.Flags().GetString("type")
		targetDSN, _ := cmd.Flags().GetString("dsn")

		if targetType == "" || targetDSN == "" {
			log.Fatalf("%s", i18n.T("migrate.cli_error_flags"))
		}

		// --- 1. Backup from source DB ---
		fmt.Println(i18n.T("migrate.cli_starting_backup"))
		backupData, err := db.ExportDataForBackup()
		if err != nil {
			log.Fatalf("%s", i18n.T("migrate.cli_error_backup", err))
		}
		fmt.Println(i18n.T("migrate.cli_backup_success"))

		// --- 2. Connect to target DB and run migrations ---
		fmt.Println(i18n.T("migrate.cli_connecting_target", targetType))
		targetStore, err := initTargetDB(targetType, targetDSN)
		if err != nil {
			log.Fatalf("%s", i18n.T("migrate.cli_error_connect", err))
		}
		fmt.Println(i18n.T("migrate.cli_connect_success"))

		// --- 3. Restore to target DB ---
		fmt.Println(i18n.T("migrate.cli_starting_restore"))
		// We call the method directly on our temporary store instance.
		if err := targetStore.ImportDataFromBackup(backupData); err != nil {
			log.Fatalf("%s", i18n.T("migrate.cli_error_restore", err))
		}

		fmt.Println(i18n.T("migrate.cli_success"))
		fmt.Println(i18n.T("migrate.cli_next_steps"))
	},
}

// initTargetDB is a helper function that initializes a new database connection
// for the migration target, runs migrations, and returns a Store instance.
// It is a simplified, one-off version of db.InitDB that does not affect the
// global `store` variable.
func initTargetDB(dbType, dsn string) (db.Store, error) {
	// This logic is intentionally duplicated from db.InitDB to create an
	// isolated instance for the migration target without modifying the global state.
	var targetDB *sql.DB
	var err error

	switch dbType {
	case "sqlite":
		if !strings.Contains(dsn, "_busy_timeout") {
			dsn += "?_busy_timeout=5000"
		}
		targetDB, err = sql.Open("sqlite", dsn)
		if err == nil {
			if _, err = targetDB.Exec("PRAGMA foreign_keys = ON;"); err != nil {
				return nil, err
			}
			if _, err = targetDB.Exec("PRAGMA journal_mode=WAL;"); err != nil {
				return nil, err
			}
		}
	case "postgres":
		targetDB, err = sql.Open("pgx", dsn)
	case "mysql":
		targetDB, err = sql.Open("mysql", dsn)
	default:
		return nil, fmt.Errorf("unsupported database type: '%s'", dbType)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to open target database: %w", err)
	}

	// Run migrations on the target DB.
	// We can reuse the migration logic from the db package.
	if err := db.RunMigrations(targetDB, dbType); err != nil {
		return nil, fmt.Errorf("failed to apply migrations to target database: %w", err)
	}

	// Return a new store instance for the target.
	return db.NewStore(dbType, targetDB)
}
