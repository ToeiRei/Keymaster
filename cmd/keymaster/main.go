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
	"os"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

// main is the entry point of the application.
func main() {
	if err := rootCmd.Execute(); err != nil {
		// The error is already printed by Cobra on failure.
		os.Exit(1)
	}
}

var rootCmd *cobra.Command

func init() {
	cobra.OnInitialize(initConfig)
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
				return fmt.Errorf(i18n.T("config.error_init_db", err))
			}
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
func initConfig() {
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
		// If the config file is not found, we can create one with default values
		// to make configuration discoverable for the user.
		if _, ok := err.(viper.ConfigFileNotFoundError); ok && cfgFile == "" {
			// We only do this if no config file was found and none was specified via flag.
			// We'll attempt to write a default config to the current directory.
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
				fmt.Println("No config file found. Created a default '.keymaster.yaml' in the current directory.")
			}
		}
	}
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
				log.Fatalf(i18n.T("deploy.cli_account_not_found"), target)
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
			log.Fatalf(i18n.T("rotate_key.cli_error_generate", err))
		}

		serial, err := db.RotateSystemKey(publicKeyString, privateKeyString)
		if err != nil {
			log.Fatalf(i18n.T("rotate_key.cli_error_save", err))
		}

		fmt.Println(i18n.T("rotate_key.cli_rotated_success", serial))
		fmt.Println(i18n.T("rotate_key.cli_deploy_reminder"))
	},
}

// auditCmd represents the 'audit' command.
// It connects to all active hosts to verify that their deployed authorized_keys
// file matches the configuration stored in the database, detecting any drift.
var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Audit hosts for configuration drift",
	Long:  `Connects to all active hosts and checks if their deployed authorized_keys file has the expected Keymaster serial number.`,
	Run: func(cmd *cobra.Command, args []string) {
		// DB is initialized in PersistentPreRunE.
		accounts, err := db.GetAllActiveAccounts()
		if err != nil {
			log.Fatalf(i18n.T("audit.cli_error_get_accounts", err))
		}

		auditTask := parallelTask{
			name:       "audit",
			startMsg:   i18n.T("parallel_task.start_message", "audit", len(accounts)),
			successMsg: i18n.T("parallel_task.audit_success_message"),
			failMsg:    i18n.T("parallel_task.audit_fail_message"),
			successLog: "CLI_AUDIT_SUCCESS",
			failLog:    "CLI_AUDIT_FAIL",
			taskFunc:   runAuditForAccount,
		}

		runParallelTasks(accounts, auditTask)
	},
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
			log.Fatalf(i18n.T("import.error_opening_file", err))
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
				fmt.Println(i18n.T("import.skip_invalid_line", err))
				skippedCount++
				continue
			}

			if comment == "" {
				fmt.Println(i18n.T("import.skip_empty_comment", string(keyData)))
				skippedCount++
				continue
			}

			if err := db.AddPublicKey(alg, keyData, comment, false); err != nil {
				if err == db.ErrDuplicate {
					fmt.Println(i18n.T("import.skip_duplicate", comment))
				} else {
					fmt.Println(i18n.T("import.error_adding_key", comment, err.Error()))
				}
				skippedCount++
				continue
			}

			fmt.Println(i18n.T("import.imported_key", comment))
			importedCount++
		}

		fmt.Println("\n" + i18n.T("import.summary", importedCount, skippedCount))
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

// runDeploymentForAccount handles the deployment logic for a single account.
// It determines the correct system key to use for connection (either the active
// key for bootstrapping or the account's last known key), generates the
// authorized_keys content, deploys it, and updates the account's key serial
// in the database upon success.
func runDeploymentForAccount(account model.Account) error {
	var connectKey *model.SystemKey
	var err error
	if account.Serial == 0 {
		connectKey, err = db.GetActiveSystemKey()
		if err != nil {
			return fmt.Errorf(i18n.T("deploy.error_get_bootstrap_key", err))
		}
		if connectKey == nil {
			return errors.New(i18n.T("deploy.error_no_bootstrap_key"))
		}
	} else {
		connectKey, err = db.GetSystemKeyBySerial(account.Serial)
		if err != nil {
			return fmt.Errorf(i18n.T("deploy.error_get_serial_key", account.Serial, err))
		}
		if connectKey == nil {
			return fmt.Errorf(i18n.T("deploy.error_no_serial_key", account.Serial))
		}
	}

	content, err := deploy.GenerateKeysContent(account.ID)
	if err != nil {
		return err // This error is already i18n-ready from the generator
	}
	activeKey, err := db.GetActiveSystemKey()
	if err != nil || activeKey == nil {
		return errors.New(i18n.T("deploy.error_get_active_key_for_serial"))
	}

	deployer, err := deploy.NewDeployer(account.Hostname, account.Username, connectKey.PrivateKey)
	if err != nil {
		return fmt.Errorf(i18n.T("deploy.error_connection_failed", err))
	}
	defer deployer.Close()

	if err := deployer.DeployAuthorizedKeys(content); err != nil {
		return fmt.Errorf(i18n.T("deploy.error_deployment_failed", err))
	}

	if err := db.UpdateAccountSerial(account.ID, activeKey.Serial); err != nil {
		return fmt.Errorf(i18n.T("deploy.error_db_update_failed", err))
	}

	return nil
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
			log.Fatalf(i18n.T("trust_host.error_get_key", err))
		}

		fingerprint := ssh.FingerprintSHA256(key) // Use standard ssh package
		fmt.Printf("\n"+i18n.T("trust_host.authenticity_warning_1")+"\n", hostname)
		fmt.Printf(i18n.T("trust_host.authenticity_warning_2")+"\n", key.Type(), fingerprint)

		if warning := sshkey.CheckHostKeyAlgorithm(key); warning != "" {
			fmt.Printf("\n%s\n", warning)
		}

		answer := promptForConfirmation(i18n.T("trust_host.confirm_prompt"))

		if answer != "yes" {
			fmt.Println(i18n.T("trust_host.not_trusted_abort"))
			os.Exit(1)
		}

		keyStr := string(ssh.MarshalAuthorizedKey(key)) // Use standard ssh package
		if err := db.AddKnownHostKey(hostname, keyStr); err != nil {
			log.Fatalf(i18n.T("trust_host.error_save_key", err))
		}

		fmt.Println(i18n.T("trust_host.added_success", hostname, key.Type()))
	},
}

// runAuditForAccount performs a configuration audit on a single account.
// It connects to the host using the system key it's expected to have,
// reads the remote authorized_keys file, and compares it against the
// configuration that *should* be there according to the database. It returns
// an error if a drift is detected.
func runAuditForAccount(account model.Account) error {
	// 1. An account with serial 0 has never been deployed. This is a known state, not a drift.
	if account.Serial == 0 {
		return errors.New(i18n.T("audit.error_not_deployed"))
	}

	// 2. Get the system key the database *thinks* is on the host.
	connectKey, err := db.GetSystemKeyBySerial(account.Serial)
	if err != nil {
		return fmt.Errorf(i18n.T("audit.error_get_serial_key", account.Serial, err))
	}
	if connectKey == nil {
		return fmt.Errorf(i18n.T("audit.error_no_serial_key", account.Serial))
	}

	// 3. Attempt to connect with that key. If this fails, the key is wrong/missing.
	deployer, err := deploy.NewDeployer(account.Hostname, account.Username, connectKey.PrivateKey)
	if err != nil {
		// Give a more helpful error message than just "connection failed".
		return fmt.Errorf(i18n.T("audit.error_connection_failed", account.Serial, err))
	}
	defer deployer.Close()

	// 4. Read the content of the remote authorized_keys file.
	remoteContentBytes, err := deployer.GetAuthorizedKeys()
	if err != nil {
		return fmt.Errorf(i18n.T("audit.error_read_remote_file", err))
	}

	// 5. Generate the content that *should* be on the host (the desired state).
	// This is always generated using the latest active system key.
	expectedContent, err := deploy.GenerateKeysContent(account.ID)
	if err != nil {
		return fmt.Errorf(i18n.T("audit.error_generate_expected", err))
	}

	// 6. Normalize both remote and expected content for a reliable, canonical comparison.
	// This handles CRLF vs LF line endings and surrounding whitespace.
	normalize := func(s string) string {
		s = strings.ReplaceAll(s, "\r\n", "\n") // Normalize line endings
		s = strings.TrimSpace(s)                // Remove leading/trailing whitespace
		return s
	}

	normalizedRemote := normalize(string(remoteContentBytes))
	normalizedExpected := normalize(expectedContent)

	// 7. Compare the actual state with the desired state.
	if normalizedRemote != normalizedExpected {
		return errors.New(i18n.T("audit.error_drift_detected"))
	}

	return nil
}

// promptForConfirmation displays a prompt and reads a line from stdin.
func promptForConfirmation(prompt string) string {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	return strings.TrimSpace(strings.ToLower(answer))
}
