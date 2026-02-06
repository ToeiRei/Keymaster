// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/toeirei/keymaster/internal/core/db"
	"github.com/toeirei/keymaster/internal/core/model"
)

// keyCmd is the root command for public key management operations.
var keyCmd = &cobra.Command{
	Use:   "key",
	Short: "Manage SSH public keys (list, add, delete, set-expiry)",
	Long: `The 'key' command group provides full public key management capabilities:
  - List all public keys with status and metadata
  - View detailed key information
  - Add new public keys
  - Delete public keys
  - Set or clear key expiration dates
  - Enable/disable global deployment status`,
}

// keyListCmd lists all public keys with optional filtering.
var keyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all public keys",
	Long: `Display all public keys in table format with their algorithms, comments, and status.
You can filter by global status or search by comment/algorithm.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		globalFilter, _ := cmd.Flags().GetString("global")
		searchTerm, _ := cmd.Flags().GetString("search")

		km := db.DefaultKeyManager()
		if km == nil {
			return fmt.Errorf("no key manager available")
		}

		keys, err := km.GetAllPublicKeys()
		if err != nil {
			return fmt.Errorf("failed to list keys: %w", err)
		}

		// Filter by global status
		if globalFilter != "" {
			filtered := []model.PublicKey{}
			isGlobal := globalFilter == "yes" || globalFilter == "true"
			for _, key := range keys {
				if key.IsGlobal == isGlobal {
					filtered = append(filtered, key)
				}
			}
			keys = filtered
		}

		// Filter by search term
		if searchTerm != "" {
			searchLower := strings.ToLower(searchTerm)
			filtered := []model.PublicKey{}
			for _, key := range keys {
				if strings.Contains(strings.ToLower(key.Comment), searchLower) ||
					strings.Contains(strings.ToLower(key.Algorithm), searchLower) {
					filtered = append(filtered, key)
				}
			}
			keys = filtered
		}

		if len(keys) == 0 {
			fmt.Println("No keys found.")
			return nil
		}

		// Display as table
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tALGORITHM\tCOMMENT\tGLOBAL\tEXPIRES")
		for _, key := range keys {
			globalStatus := "no"
			if key.IsGlobal {
				globalStatus = "yes"
			}
			expires := "never"
			if !key.ExpiresAt.IsZero() {
				expires = key.ExpiresAt.Format("2006-01-02")
			}
			fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n",
				key.ID, key.Algorithm, key.Comment, globalStatus, expires)
		}
		w.Flush()

		return nil
	},
}

// keyShowCmd displays detailed information about a specific key.
var keyShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show detailed key information",
	Long:  `Display full details of a public key including accounts it's assigned to.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid key ID: %w", err)
		}

		km := db.DefaultKeyManager()
		if km == nil {
			return fmt.Errorf("no key manager available")
		}

		// Get all keys and find the one we want
		keys, err := km.GetAllPublicKeys()
		if err != nil {
			return fmt.Errorf("failed to load keys: %w", err)
		}

		var key *model.PublicKey
		for i, k := range keys {
			if k.ID == id {
				key = &keys[i]
				break
			}
		}

		if key == nil {
			return fmt.Errorf("key not found: %d", id)
		}

		// Display key details
		globalStatus := "no"
		if key.IsGlobal {
			globalStatus = "yes"
		}
		expires := "never"
		if !key.ExpiresAt.IsZero() {
			expires = key.ExpiresAt.Format("2006-01-02 15:04:05")
		}

		fmt.Printf("ID:         %d\n", key.ID)
		fmt.Printf("Algorithm:  %s\n", key.Algorithm)
		fmt.Printf("Comment:    %s\n", key.Comment)
		fmt.Printf("Global:     %s\n", globalStatus)
		fmt.Printf("Expires:    %s\n", expires)
		fmt.Printf("Key Data:   %s... (truncated)\n", truncateString(key.KeyData, 50))

		// Get assigned accounts
		accounts, accountErr := km.GetAccountsForKey(key.ID)
		if accountErr == nil && len(accounts) > 0 {
			fmt.Println("\nAssigned Accounts:")
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ACCOUNT_ID\tUSERNAME\tHOSTNAME\tLABEL")
			for _, acc := range accounts {
				fmt.Fprintf(w, "%d\t%s\t%s\t%s\n",
					acc.ID, acc.Username, acc.Hostname, acc.Label)
			}
			w.Flush()
		}

		return nil
	},
}

// keyAddCmd adds a new public key.
var keyAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new public key",
	Long:  `Add a new SSH public key with algorithm, key data, and comment.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		algorithm, _ := cmd.Flags().GetString("algorithm")
		keyData, _ := cmd.Flags().GetString("key-data")
		comment, _ := cmd.Flags().GetString("comment")
		isGlobal, _ := cmd.Flags().GetBool("global")
		expiresStr, _ := cmd.Flags().GetString("expires")

		if algorithm == "" {
			return fmt.Errorf("--algorithm is required")
		}
		if keyData == "" {
			return fmt.Errorf("--key-data is required")
		}
		if comment == "" {
			return fmt.Errorf("--comment is required")
		}

		var expiresAt time.Time
		if expiresStr != "" {
			parsed, err := time.Parse("2006-01-02", expiresStr)
			if err != nil {
				return fmt.Errorf("invalid expiry date format (use YYYY-MM-DD): %w", err)
			}
			expiresAt = parsed
		}

		km := db.DefaultKeyManager()
		if km == nil {
			return fmt.Errorf("no key manager available")
		}

		addedKey, err := km.AddPublicKeyAndGetModel(algorithm, keyData, comment, isGlobal, expiresAt)
		if err != nil {
			return fmt.Errorf("failed to add key: %w", err)
		}

		fmt.Printf("Key added successfully with ID: %d\n", addedKey.ID)
		return nil
	},
}

// keyDeleteCmd deletes a public key.
var keyDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a public key",
	Long:  `Delete a public key and remove it from all account assignments.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid key ID: %w", err)
		}

		force, _ := cmd.Flags().GetBool("force")

		km := db.DefaultKeyManager()
		if km == nil {
			return fmt.Errorf("no key manager available")
		}

		// Get key info for confirmation
		keys, err := km.GetAllPublicKeys()
		if err != nil {
			return fmt.Errorf("failed to load keys: %w", err)
		}

		var key *model.PublicKey
		for i, k := range keys {
			if k.ID == id {
				key = &keys[i]
				break
			}
		}

		if key == nil {
			return fmt.Errorf("key not found: %d", id)
		}

		// Confirm deletion unless --force is used
		if !force {
			fmt.Printf("Delete key: %s (ID: %d)? (yes/no): ", key.Comment, id)
			var response string
			fmt.Scanln(&response)
			if strings.ToLower(response) != "yes" {
				fmt.Println("Deletion cancelled.")
				return nil
			}
		}

		if err := km.DeletePublicKey(id); err != nil {
			return fmt.Errorf("failed to delete key: %w", err)
		}

		fmt.Printf("Key deleted: %s\n", key.Comment)
		return nil
	},
}

// keySetExpiryCmd sets or clears the expiration date for a key.
var keySetExpiryCmd = &cobra.Command{
	Use:   "set-expiry <id> <date>",
	Short: "Set or clear key expiration date",
	Long: `Set the expiration date for a key (format: YYYY-MM-DD) or use 'never' to clear expiration.
Keys past their expiration date will not be deployed.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid key ID: %w", err)
		}

		dateStr := args[1]
		var expiresAt time.Time

		if strings.ToLower(dateStr) == "never" {
			// Zero time means no expiration
			expiresAt = time.Time{}
		} else {
			parsed, err := time.Parse("2006-01-02", dateStr)
			if err != nil {
				return fmt.Errorf("invalid date format (use YYYY-MM-DD or 'never'): %w", err)
			}
			expiresAt = parsed
		}

		km := db.DefaultKeyManager()
		if km == nil {
			return fmt.Errorf("no key manager available")
		}

		if err := km.SetPublicKeyExpiry(id, expiresAt); err != nil {
			return fmt.Errorf("failed to set expiry: %w", err)
		}

		if expiresAt.IsZero() {
			fmt.Printf("Key %d expiration cleared (never expires)\n", id)
		} else {
			fmt.Printf("Key %d expiration set to: %s\n", id, expiresAt.Format("2006-01-02"))
		}
		return nil
	},
}

// keyEnableGlobalCmd enables global deployment for a key.
var keyEnableGlobalCmd = &cobra.Command{
	Use:   "enable-global <id>",
	Short: "Enable global deployment for a key",
	Long:  `Mark a key as global, so it will be deployed to all active accounts automatically.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid key ID: %w", err)
		}

		km := db.DefaultKeyManager()
		if km == nil {
			return fmt.Errorf("no key manager available")
		}

		// Check current status
		keys, err := km.GetAllPublicKeys()
		if err != nil {
			return fmt.Errorf("failed to load keys: %w", err)
		}

		var key *model.PublicKey
		for i, k := range keys {
			if k.ID == id {
				key = &keys[i]
				break
			}
		}

		if key == nil {
			return fmt.Errorf("key not found: %d", id)
		}

		// If already global, report but don't error
		if key.IsGlobal {
			fmt.Printf("Key %d is already global\n", id)
			return nil
		}

		// Toggle to enable global
		if err := km.TogglePublicKeyGlobal(id); err != nil {
			return fmt.Errorf("failed to enable global: %w", err)
		}

		fmt.Printf("Key %d enabled for global deployment\n", id)
		return nil
	},
}

// keyDisableGlobalCmd disables global deployment for a key.
var keyDisableGlobalCmd = &cobra.Command{
	Use:   "disable-global <id>",
	Short: "Disable global deployment for a key",
	Long:  `Remove global status from a key, so it will only be deployed to explicitly assigned accounts.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid key ID: %w", err)
		}

		km := db.DefaultKeyManager()
		if km == nil {
			return fmt.Errorf("no key manager available")
		}

		// Check current status
		keys, err := km.GetAllPublicKeys()
		if err != nil {
			return fmt.Errorf("failed to load keys: %w", err)
		}

		var key *model.PublicKey
		for i, k := range keys {
			if k.ID == id {
				key = &keys[i]
				break
			}
		}

		if key == nil {
			return fmt.Errorf("key not found: %d", id)
		}

		// If already non-global, report but don't error
		if !key.IsGlobal {
			fmt.Printf("Key %d is already non-global\n", id)
			return nil
		}

		// Toggle to disable global
		if err := km.TogglePublicKeyGlobal(id); err != nil {
			return fmt.Errorf("failed to disable global: %w", err)
		}

		fmt.Printf("Key %d disabled from global deployment\n", id)
		return nil
	},
}

// registerKeyCommands registers all key-related subcommands.
func registerKeyCommands() {
	// Register subcommands with the main key command
	keyCmd.AddCommand(keyListCmd)
	keyCmd.AddCommand(keyShowCmd)
	keyCmd.AddCommand(keyAddCmd)
	keyCmd.AddCommand(keyDeleteCmd)
	keyCmd.AddCommand(keySetExpiryCmd)
	keyCmd.AddCommand(keyEnableGlobalCmd)
	keyCmd.AddCommand(keyDisableGlobalCmd)

	// Setup flags for add (only if not already defined)
	if keyAddCmd.Flags().Lookup("algorithm") == nil {
		keyAddCmd.Flags().StringP("algorithm", "a", "", "Key algorithm (e.g., ssh-ed25519, ssh-rsa) (required)")
		keyAddCmd.Flags().StringP("key-data", "k", "", "Base64-encoded key data (required)")
		keyAddCmd.Flags().StringP("comment", "c", "", "Key comment/identifier (required)")
		keyAddCmd.Flags().BoolP("global", "g", false, "Deploy to all accounts")
		keyAddCmd.Flags().String("expires", "", "Expiration date (YYYY-MM-DD)")
		keyAddCmd.MarkFlagRequired("algorithm")
		keyAddCmd.MarkFlagRequired("key-data")
		keyAddCmd.MarkFlagRequired("comment")
	}

	// Setup flags for delete (only if not already defined)
	if keyDeleteCmd.Flags().Lookup("force") == nil {
		keyDeleteCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
	}

	// Setup flags for list (only if not already defined)
	if keyListCmd.Flags().Lookup("global") == nil {
		keyListCmd.Flags().String("global", "", "Filter by global status (yes or no)")
		keyListCmd.Flags().String("search", "", "Search by comment or algorithm")
	}
}

// truncateString truncates a string to maxLen characters, adding "..." if truncated.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
