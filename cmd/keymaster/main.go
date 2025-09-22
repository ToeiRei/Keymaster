package main

import (
	"bufio"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	"github.com/toeirei/keymaster/internal/crypto/ssh"
	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/deploy"
	"github.com/toeirei/keymaster/internal/model"
	"github.com/toeirei/keymaster/internal/sshkey"
	"github.com/toeirei/keymaster/internal/tui"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "keymaster",
	Short: "Keymaster is a lightweight, agentless SSH access manager.",
	Long: `Keymaster centralizes control of authorized_keys files.
Instead of scattering user keys everywhere, Keymaster plants a single
system key per account and uses it as a foothold to rewrite and
version-control access. A database becomes the source of truth.

Running without a subcommand will launch the interactive TUI.`,
	Run: func(cmd *cobra.Command, args []string) {
		tui.Run()
	},
}

func init() {
	// Here we will define our flags and configuration settings.
	// Example: rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.keymaster.yaml)")

	// Add commands
	rootCmd.AddCommand(deployCmd)
	rootCmd.AddCommand(rotateKeyCmd)
	rootCmd.AddCommand(auditCmd)
	rootCmd.AddCommand(importCmd)
}

var deployCmd = &cobra.Command{
	Use:   "deploy [user@host]",
	Short: "Deploy authorized_keys to one or all hosts",
	Long: `Renders the authorized_keys file from the database state and deploys it.
If an account (user@host) is specified, deploys only to that account.
If no account is specified, deploys to all active accounts in the database.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if _, err := db.InitDB("./keymaster.db"); err != nil {
			log.Fatalf("Error initializing database: %v", err)
		}

		allAccounts, err := db.GetAllActiveAccounts()
		if err != nil {
			log.Fatalf("Error getting accounts: %v", err)
		}

		var targetAccounts []model.Account
		if len(args) > 0 {
			target := args[0]
			found := false
			for _, acc := range allAccounts {
				if acc.String() == target {
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

		if len(targetAccounts) == 0 {
			fmt.Println("No active accounts to deploy to.")
			return
		}

		var wg sync.WaitGroup
		results := make(chan string, len(targetAccounts))

		fmt.Printf("🚀 Starting deployment to %d account(s)...\n", len(targetAccounts))

		for _, acc := range targetAccounts {
			wg.Add(1)
			go func(account model.Account) {
				defer wg.Done()
				err := runDeploymentForAccount(account)
				if err != nil {
					results <- fmt.Sprintf("💥 Failed to deploy to %s: %v", account.String(), err)
				} else {
					results <- fmt.Sprintf("✅ Successfully deployed to %s", account.String())
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
		fmt.Println("\nDeployment complete.")
	},
}

var rotateKeyCmd = &cobra.Command{
	Use:   "rotate-key",
	Short: "Rotates the active Keymaster system key",
	Long: `Generates a new ed25519 key pair, saves it to the database, and sets it as the active key.
The previous key is kept for accessing hosts that have not yet been updated.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("⚙️  Rotating system key...")
		if _, err := db.InitDB("./keymaster.db"); err != nil {
			log.Fatalf("Error initializing database: %v", err)
		}

		pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			log.Fatalf("Error generating key pair: %v", err)
		}

		sshPubKey, err := ssh.NewPublicKey(pubKey)
		if err != nil {
			log.Fatalf("Error creating SSH public key: %v", err)
		}
		pubKeyBytes := ssh.MarshalAuthorizedKey(sshPubKey)
		publicKeyString := fmt.Sprintf("%s keymaster-system-key", strings.TrimSpace(string(pubKeyBytes)))

		pemBlock, err := ssh.MarshalEd25519PrivateKey(privKey, "")
		if err != nil {
			log.Fatalf("Error marshaling private key: %v", err)
		}
		privateKeyString := string(pem.EncodeToMemory(pemBlock))

		serial, err := db.RotateSystemKey(publicKeyString, privateKeyString)
		if err != nil {
			log.Fatalf("Error saving rotated key to database: %v", err)
		}

		fmt.Printf("\n✅ Successfully rotated system key. The new active key is serial #%d.\n", serial)
		fmt.Println("Run 'keymaster deploy' to apply the new key to your fleet.")
	},
}

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Audit hosts for configuration drift",
	Long:  `Connects to all active hosts and checks if their deployed authorized_keys file has the expected Keymaster serial number.`,
	Run: func(cmd *cobra.Command, args []string) {
		if _, err := db.InitDB("./keymaster.db"); err != nil {
			log.Fatalf("Error initializing database: %v", err)
		}

		accounts, err := db.GetAllActiveAccounts()
		if err != nil {
			log.Fatalf("Error getting accounts: %v", err)
		}

		if len(accounts) == 0 {
			fmt.Println("No active accounts to audit.")
			return
		}

		var wg sync.WaitGroup
		results := make(chan string, len(accounts))

		fmt.Printf("🔬 Starting audit of %d account(s)...\n", len(accounts))

		for _, acc := range accounts {
			wg.Add(1)
			go func(account model.Account) {
				defer wg.Done()
				err := runAuditForAccount(account)
				if err != nil {
					results <- fmt.Sprintf("🚨 Drift detected on %s: %v", account.String(), err)
				} else {
					results <- fmt.Sprintf("✅ OK: %s", account.String())
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
		fmt.Println("\nAudit complete.")
	},
}

var importCmd = &cobra.Command{
	Use:   "import [authorized_keys_file]",
	Short: "Import public keys from an authorized_keys file",
	Long:  `Reads a standard authorized_keys file and imports the public keys into the Keymaster database.`,
	Args:  cobra.ExactArgs(1), // Ensures we get exactly one file path
	Run: func(cmd *cobra.Command, args []string) {
		filePath := args[0]
		fmt.Printf("🔑 Importing keys from %s...\n", filePath)

		// Initialize the database.
		if _, err := db.InitDB("./keymaster.db"); err != nil {
			log.Fatalf("Error initializing database: %v", err)
		}

		file, err := os.Open(filePath)
		if err != nil {
			fmt.Printf("Error opening file: %v\n", err)
			os.Exit(1)
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
				skippedCount++
				continue
			}

			if comment == "" {
				fmt.Printf("  - Skipping key with empty comment: %.20s...\n", keyData)
				skippedCount++
				continue
			}

			if err := db.AddPublicKey(alg, keyData, comment); err != nil {
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

func runAuditForAccount(account model.Account) error {
	connectKey, err := db.GetSystemKeyBySerial(account.Serial)
	if err != nil {
		return fmt.Errorf("could not get system key %d from db: %w", account.Serial, err)
	}
	if connectKey == nil {
		// If serial is 0, it's a new host, which is technically not out of sync.
		if account.Serial == 0 {
			return fmt.Errorf("host has not been deployed to yet (serial is 0)")
		}
		return fmt.Errorf("db inconsistency: no system key found for serial %d", account.Serial)
	}

	deployer, err := deploy.NewDeployer(account.Hostname, account.Username, connectKey.PrivateKey)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer deployer.Close()

	remoteContent, err := deployer.GetAuthorizedKeys()
	if err != nil {
		return fmt.Errorf("could not read remote authorized_keys: %w", err)
	}

	firstLine := strings.Split(string(remoteContent), "\n")[0]
	remoteSerial, err := sshkey.ParseSerial(firstLine)
	if err != nil {
		return fmt.Errorf("could not parse serial from remote file: %w", err)
	}

	if remoteSerial != account.Serial {
		return fmt.Errorf("remote serial (%d) does not match database serial (%d)", remoteSerial, account.Serial)
	}

	return nil
}
