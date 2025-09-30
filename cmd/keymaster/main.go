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
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

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

// main is the entry point of the application.
func main() {
	// Install signal handler for graceful shutdown of bootstrap sessions
	bootstrap.InstallSignalHandler()

	// Set up cleanup store for bootstrap operations
	defer func() {
		if err := bootstrap.CleanupAllActiveSessions(); err != nil {
			log.Printf("Error during final cleanup: %v", err)
		}
	}()

	if err := rootCmd.Execute(); err != nil {
		// The error is already printed by Cobra on failure.
		os.Exit(1)
	}
}

var rootCmd *cobra.Command

func init() {
	// cobra.OnInitialize can't handle errors, so we wrap initConfig.
	// The error is checked and handled inside newRootCmd's PersistentPreRunE,
	// which is a more appropriate place for error handling that needs to
	// stop command execution. For now, we just log if there's an issue
	// during the initial setup phase.
	cobra.OnInitialize(func() { _ = initConfig() })
	rootCmd = newRootCmd()

	// Set defaults in viper. These are used if not set in the config file or by flags.
	viper.SetDefault("database.type", "sqlite")
	viper.SetDefault("database.dsn", "./keymaster.db")
	viper.SetDefault("language", "en")
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
			i18n.Init(viper.GetString("language")) // Initialize i18n here
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
	cmd.AddCommand(decommissionCmd)

	// Set version
	cmd.Version = version

	// Define flags
	cmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.keymaster.yaml or ./keymaster.yaml)")
	cmd.PersistentFlags().String("db-type", "sqlite", "Database type (e.g., sqlite, postgres)")
	cmd.PersistentFlags().String("db-dsn", "./keymaster.db", "Database connection string (DSN)")
	cmd.PersistentFlags().String("lang", "en", `TUI language ("en", "de")`)

	// Bind flags to viper
	viper.BindPFlag("database.type", cmd.PersistentFlags().Lookup("db-type"))
	viper.BindPFlag("database.dsn", cmd.PersistentFlags().Lookup("db-dsn"))
	viper.BindPFlag("language", cmd.PersistentFlags().Lookup("lang"))

	// Note: Flags are configured in the init() function on the global rootCmd.
	// Cobra automatically handles making these flags available on new command
	// instances created for tests, so we don't need to re-declare them here.

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
		target := args[0]
		var hostname string
		if strings.Contains(target, "@") {
			parts := strings.SplitN(target, "@", 2)
			hostname = parts[1]
		} else {
			hostname = target // Assume the whole string is the hostname if no '@'
		}

		fmt.Println(i18n.T("trust_host.retrieving_key", hostname))
		key, err := deploy.GetRemoteHostKey(hostname)
		if err != nil {
			log.Fatalf("%s", i18n.T("trust_host.error_get_key", err))
		}

		fingerprint := ssh.FingerprintSHA256(key) // Use standard ssh package
		fmt.Printf("\n%s\n", i18n.T("trust_host.authenticity_warning_1", hostname))
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
		normalized := normalizeKnownHostKeyName(hostname)
		if err := db.AddKnownHostKey(normalized, keyStr); err != nil {
			log.Fatalf("%s", i18n.T("trust_host.error_save_key", err))
		}

		fmt.Printf("%s\n", i18n.T("trust_host.added_success", normalized, key.Type()))
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

			// Parse hostname for port if it contains one
			_, port := account.Hostname, "22"
			if idx := strings.LastIndex(account.Hostname, ":"); idx > 0 {
				// Check if it's IPv6 by looking for multiple colons
				if strings.Count(account.Hostname, ":") == 1 {
					port = account.Hostname[idx+1:]
				}
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

// normalizeKnownHostKeyName normalizes a hostname for storage in the known hosts database
// so that it matches lookups performed during SSH handshakes. It removes any port and
// strips IPv6 brackets, returning just the host portion.
func normalizeKnownHostKeyName(h string) string {
	h = strings.TrimSpace(h)
	if h == "" {
		return h
	}
	// If a port is present (e.g., "example.com:2222" or "[2001:db8::1]:2222"),
	// SplitHostPort returns the host without brackets for IPv6.
	if host, _, err := net.SplitHostPort(h); err == nil {
		return host
	}
	// If it's a bracketed IPv6 without port like "[2001:db8::1]", trim brackets.
	if strings.HasPrefix(h, "[") && strings.HasSuffix(h, "]") {
		return strings.TrimSuffix(strings.TrimPrefix(h, "["), "]")
	}
	// Otherwise, return as-is (covers plain hostnames or raw IPv6 without brackets/port).
	return h
}

// runDeploymentForAccount is a simple wrapper for the CLI to match the
// signature required by runParallelTasks. It calls the centralized
// deployment logic from the deploy package.
func runDeploymentForAccount(account model.Account) error {
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
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Parse flags
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

func init() {
	// Add flags for decommission command
	decommissionCmd.Flags().Bool("skip-remote", false, "Skip remote SSH cleanup (only delete from database)")
	decommissionCmd.Flags().Bool("keep-file", false, "Remove only Keymaster content, keep other keys in authorized_keys")
	decommissionCmd.Flags().Bool("force", false, "Continue even if remote cleanup fails")
	decommissionCmd.Flags().Bool("dry-run", false, "Show what would be done without making changes")
	decommissionCmd.Flags().String("tag", "", "Decommission all accounts with this tag (format: key:value)")
}

// findAccountByIdentifier finds an account by ID, user@host, or label
func findAccountByIdentifier(identifier string, accounts []model.Account) (*model.Account, error) {
	// Try to parse as account ID first
	if accountID, err := fmt.Sscanf(identifier, "%d", new(int)); accountID == 1 && err == nil {
		var id int
		fmt.Sscanf(identifier, "%d", &id)
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
