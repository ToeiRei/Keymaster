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

// rootCmd represents the base command when called without any subcommands.
// It initializes the database via PersistentPreRunE for all child commands
// and launches the interactive TUI when run directly.
var rootCmd = &cobra.Command{
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
			return fmt.Errorf("error initializing database: %w", err)
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		// The database is already initialized by PersistentPreRunE.
		tui.Run()
	},
}

// init is a special Go function that is called before main().
// It sets up the application's command-line interface, including flags,
// default configuration values, and subcommands.
func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.Version = version
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.keymaster.yaml or ./keymaster.yaml)")

	// Database flags
	rootCmd.PersistentFlags().String("db-type", "sqlite", "Database type (e.g., sqlite, postgres)")
	rootCmd.PersistentFlags().String("db-dsn", "./keymaster.db", "Database connection string (DSN)")
	rootCmd.PersistentFlags().String("lang", "en", `TUI language ("en", "de")`)

	// Bind flags to viper
	viper.BindPFlag("database.type", rootCmd.PersistentFlags().Lookup("db-type"))
	viper.BindPFlag("database.dsn", rootCmd.PersistentFlags().Lookup("db-dsn"))
	viper.BindPFlag("language", rootCmd.PersistentFlags().Lookup("lang"))

	// Set defaults in viper. These are used if not set in the config file or by flags.
	viper.SetDefault("database.type", "sqlite")
	viper.SetDefault("database.dsn", "./keymaster.db")
	viper.SetDefault("language", "en")

	// Add commands
	rootCmd.AddCommand(deployCmd)
	rootCmd.AddCommand(rotateKeyCmd)
	rootCmd.AddCommand(auditCmd)
	rootCmd.AddCommand(importCmd)
	rootCmd.AddCommand(trustHostCmd)
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
				log.Fatalf("Account '%s' not found or is not active.", target)
			}
		} else {
			targetAccounts = allAccounts
		}

		deployTask := parallelTask{
			name:       "deployment",
			startMsg:   "🚀 Starting deployment to %d account(s)...\n",
			successMsg: "✅ Successfully deployed to %s",
			failMsg:    "💥 Failed to deploy to %s: %v",
			successLog: "DEPLOY_SUCCESS",
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
		fmt.Println("⚙️  Rotating system key...")
		// DB is initialized in PersistentPreRunE.
		publicKeyString, privateKeyString, err := internalkey.GenerateAndMarshalEd25519Key("keymaster-system-key")
		if err != nil {
			log.Fatalf("Error generating new system key: %v", err)
		}

		serial, err := db.RotateSystemKey(publicKeyString, privateKeyString)
		if err != nil {
			log.Fatalf("Error saving rotated key to database: %v", err)
		}

		fmt.Printf("\n✅ Successfully rotated system key. The new active key is serial #%d.\n", serial)
		fmt.Println("Run 'keymaster deploy' to apply the new key to your fleet.")
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
			log.Fatalf("Error getting accounts: %v", err)
		}

		auditTask := parallelTask{
			name:       "audit",
			startMsg:   "🔬 Starting audit of %d account(s)...\n",
			successMsg: "✅ OK: %s",
			failMsg:    "🚨 Drift detected on %s: %v",
			successLog: "AUDIT_SUCCESS",
			failLog:    "AUDIT_FAIL",
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
		// DB is initialized in PersistentPreRunE.
		fmt.Printf("🔑 Importing keys from %s...\n", filePath)

		file, err := os.Open(filePath)
		if err != nil {
			log.Fatalf("Error opening file: %v", err)
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
				fmt.Printf("  - Skipping invalid key line: %v\n", err)
				skippedCount++
				continue
			}

			if comment == "" {
				fmt.Printf("  - Skipping key with empty comment: %.20s...\n", keyData)
				skippedCount++
				continue
			}

			if err := db.AddPublicKey(alg, keyData, comment, false); err != nil {
				if strings.Contains(err.Error(), "UNIQUE constraint failed") {
					fmt.Printf("  - Skipping duplicate key (comment exists): %s\n", comment)
				} else {
					fmt.Printf("  - Error adding key '%s': %v\n", comment, err)
				}
				skippedCount++
				continue
			}

			fmt.Printf("  + Imported key: %s\n", comment)
			importedCount++
		}

		fmt.Printf("\n✅ Import complete. Imported %d keys, skipped %d.\n", importedCount, skippedCount)
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
		fmt.Printf("No active accounts for %s.\n", task.name)
		return
	}

	var wg sync.WaitGroup
	results := make(chan string, len(accounts))

	fmt.Printf(task.startMsg, len(accounts))

	for _, acc := range accounts {
		wg.Add(1)
		go func(account model.Account) {
			defer wg.Done()
			err := task.taskFunc(account)
			details := fmt.Sprintf("account: %s", account.String())
			if err != nil {
				results <- fmt.Sprintf(task.failMsg, account.String(), err)
				_ = db.LogAction(task.failLog, fmt.Sprintf("%s, error: %v", details, err))
			} else {
				results <- fmt.Sprintf(task.successMsg, account.String())
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
	fmt.Printf("\n%s complete.\n", strings.Title(task.name))
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
			return fmt.Errorf("failed to get active system key for bootstrap: %w", err)
		}
		if connectKey == nil {
			return fmt.Errorf("no active system key found for bootstrap")
		}
	} else {
		connectKey, err = db.GetSystemKeyBySerial(account.Serial)
		if err != nil {
			return fmt.Errorf("failed to get system key with serial %d: %w", account.Serial, err)
		}
		if connectKey == nil {
			return fmt.Errorf("db inconsistency: no system key found for serial %d", account.Serial)
		}
	}

	content, err := deploy.GenerateKeysContent(account.ID)
	if err != nil {
		return err
	}
	activeKey, err := db.GetActiveSystemKey()
	if err != nil || activeKey == nil {
		return fmt.Errorf("could not retrieve active system key for serial update")
	}

	deployer, err := deploy.NewDeployer(account.Hostname, account.Username, connectKey.PrivateKey)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer deployer.Close()

	if err := deployer.DeployAuthorizedKeys(content); err != nil {
		return fmt.Errorf("deployment failed: %w", err)
	}

	if err := db.UpdateAccountSerial(account.ID, activeKey.Serial); err != nil {
		return fmt.Errorf("db update failed after successful deploy: %w", err)
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
		_, hostname, found := strings.Cut(target, "@")
		if !found {
			log.Fatalf("Invalid account format. Expected user@host.")
		}

		fmt.Printf("Attempting to retrieve host key from %s...\n", hostname)
		key, err := deploy.GetRemoteHostKey(hostname)
		if err != nil {
			log.Fatalf("Could not get host key: %v", err)
		}

		fingerprint := ssh.FingerprintSHA256(key) // Use standard ssh package
		fmt.Printf("\nThe authenticity of host '%s' can't be established.\n", hostname)
		fmt.Printf("%s key fingerprint is %s.\n", key.Type(), fingerprint)

		if warning := sshkey.CheckHostKeyAlgorithm(key); warning != "" {
			fmt.Printf("\n%s\n", warning)
		}

		fmt.Print("Are you sure you want to continue connecting (yes/no)? ")

		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))

		if answer != "yes" {
			fmt.Println("Host key not trusted. Aborting.")
			os.Exit(1)
		}

		keyStr := string(ssh.MarshalAuthorizedKey(key)) // Use standard ssh package
		if err := db.AddKnownHostKey(hostname, keyStr); err != nil {
			log.Fatalf("Failed to save host key to database: %v", err)
		}

		fmt.Printf("Warning: Permanently added '%s' (type %s) to the list of known hosts.\n", hostname, key.Type())
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
		return fmt.Errorf("host has not been deployed to yet (serial is 0)")
	}

	// 2. Get the system key the database *thinks* is on the host.
	connectKey, err := db.GetSystemKeyBySerial(account.Serial)
	if err != nil {
		return fmt.Errorf("could not get system key %d from db: %w", account.Serial, err)
	}
	if connectKey == nil {
		return fmt.Errorf("db inconsistency: no system key found for serial %d", account.Serial)
	}

	// 3. Attempt to connect with that key. If this fails, the key is wrong/missing.
	deployer, err := deploy.NewDeployer(account.Hostname, account.Username, connectKey.PrivateKey)
	if err != nil {
		// Give a more helpful error message than just "connection failed".
		return fmt.Errorf("connection failed using key serial %d: %w", account.Serial, err)
	}
	defer deployer.Close()

	// 4. Read the content of the remote authorized_keys file.
	remoteContentBytes, err := deployer.GetAuthorizedKeys()
	if err != nil {
		return fmt.Errorf("could not read remote authorized_keys: %w", err)
	}

	// 5. Generate the content that *should* be on the host (the desired state).
	// This is always generated using the latest active system key.
	expectedContent, err := deploy.GenerateKeysContent(account.ID)
	if err != nil {
		return fmt.Errorf("could not generate expected keys content for comparison: %w", err)
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
		return fmt.Errorf("content drift detected. Remote file does not match the expected configuration")
	}

	return nil
}
