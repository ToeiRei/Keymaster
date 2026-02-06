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

	"github.com/spf13/cobra"
	"github.com/toeirei/keymaster/core/db"
	"github.com/toeirei/keymaster/core/model"
)

// accountCmd is the root command for account management operations.
var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Manage SSH accounts (list, create, update, delete, assign keys)",
	Long: `The 'account' command group provides full account management capabilities:
  - List all accounts with status and assignment info
  - View detailed account information
  - Create new accounts
  - Update account properties (hostname, label, tags)
  - Enable/disable accounts (active/inactive status)
  - Delete accounts
  - Assign and unassign SSH keys to/from accounts`,
}

// accountListCmd lists all accounts with optional filtering.
var accountListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all accounts",
	Long: `Display all accounts in table format with their hostnames, labels, tags, and status.
You can filter by status (active, inactive) or search by hostname/username.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		statusFilter, _ := cmd.Flags().GetString("status")
		searchTerm, _ := cmd.Flags().GetString("search")

		accounts, err := db.GetAllAccounts()
		if err != nil {
			return fmt.Errorf("failed to list accounts: %w", err)
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

		if len(accounts) == 0 {
			fmt.Println("No accounts found.")
			return nil
		}

		// Display as table
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tUSERNAME\tHOSTNAME\tLABEL\tTAGS\tSTATUS")
		for _, acc := range accounts {
			status := "active"
			if !acc.IsActive {
				status = "inactive"
			}
			fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\n",
				acc.ID, acc.Username, acc.Hostname, acc.Label, acc.Tags, status)
		}
		w.Flush()

		return nil
	},
}

// accountShowCmd displays detailed information about a specific account.
var accountShowCmd = &cobra.Command{
	Use:   "show <id or hostname>",
	Short: "Show detailed account information",
	Long:  `Display full details of an account including assigned SSH keys.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		identifier := args[0]

		var account *model.Account

		// Try parsing as ID first, then as hostname
		if id, parseErr := strconv.Atoi(identifier); parseErr == nil {
			// Look up account by ID
			allAccounts, err := db.GetAllAccounts()
			if err != nil {
				return fmt.Errorf("failed to load accounts: %w", err)
			}
			for i, acc := range allAccounts {
				if acc.ID == id {
					account = &allAccounts[i]
					break
				}
			}
		} else {
			// Try to find by hostname
			accounts, err := db.GetAllAccounts()
			if err != nil {
				return fmt.Errorf("failed to load accounts: %w", err)
			}
			for i, acc := range accounts {
				if acc.Hostname == identifier {
					account = &accounts[i]
					break
				}
			}
		}

		if account == nil {
			return fmt.Errorf("account not found: %s", identifier)
		}

		// Display account details
		status := "active"
		if !account.IsActive {
			status = "inactive"
		}

		fmt.Printf("ID:        %d\n", account.ID)
		fmt.Printf("Username:  %s\n", account.Username)
		fmt.Printf("Hostname:  %s\n", account.Hostname)
		fmt.Printf("Label:     %s\n", account.Label)
		fmt.Printf("Tags:      %s\n", account.Tags)
		fmt.Printf("Status:    %s\n", status)
		fmt.Printf("Serial:    %d\n", account.Serial)

		// Get assigned keys
		km := db.DefaultKeyManager()
		if km != nil {
			keys, keyErr := km.GetKeysForAccount(account.ID)
			if keyErr == nil && len(keys) > 0 {
				fmt.Println("\nAssigned Keys:")
				w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
				fmt.Fprintln(w, "KEY_ID\tALGORITHM\tCOMMENT\tIS_GLOBAL")
				for _, key := range keys {
					isGlobal := "no"
					if key.IsGlobal {
						isGlobal = "yes"
					}
					fmt.Fprintf(w, "%d\t%s\t%s\t%s\n",
						key.ID, key.Algorithm, key.Comment, isGlobal)
				}
				w.Flush()
			}
		}

		return nil
	},
}

// accountCreateCmd creates a new account.
var accountCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new account",
	Long:  `Create a new SSH account with username, hostname, and optional label and tags.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		username, _ := cmd.Flags().GetString("username")
		hostname, _ := cmd.Flags().GetString("hostname")
		label, _ := cmd.Flags().GetString("label")
		tags, _ := cmd.Flags().GetString("tags")

		if username == "" {
			return fmt.Errorf("--username is required")
		}
		if hostname == "" {
			return fmt.Errorf("--hostname is required")
		}

		am := db.DefaultAccountManager()
		if am == nil {
			return fmt.Errorf("no account manager available")
		}

		id, err := am.AddAccount(username, hostname, label, tags)
		if err != nil {
			return fmt.Errorf("failed to create account: %w", err)
		}

		fmt.Printf("Account created successfully with ID: %d\n", id)
		return nil
	},
}

// accountUpdateCmd updates account properties.
var accountUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update account properties",
	Long:  `Update hostname, label, or tags for an existing account.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid account ID: %w", err)
		}

		// Check if account exists
		allAccounts, err := db.GetAllAccounts()
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
		if cmd.Flags().Changed("hostname") {
			hostname, _ := cmd.Flags().GetString("hostname")
			if hostname != "" {
				if err := db.UpdateAccountHostname(id, hostname); err != nil {
					return fmt.Errorf("failed to update hostname: %w", err)
				}
				fmt.Printf("Hostname updated to: %s\n", hostname)
			}
		}

		if cmd.Flags().Changed("label") {
			label, _ := cmd.Flags().GetString("label")
			if err := db.UpdateAccountLabel(id, label); err != nil {
				return fmt.Errorf("failed to update label: %w", err)
			}
			fmt.Printf("Label updated to: %s\n", label)
		}

		if cmd.Flags().Changed("tags") {
			tags, _ := cmd.Flags().GetString("tags")
			if err := db.UpdateAccountTags(id, tags); err != nil {
				return fmt.Errorf("failed to update tags: %w", err)
			}
			fmt.Printf("Tags updated to: %s\n", tags)
		}

		if !cmd.Flags().Changed("hostname") && !cmd.Flags().Changed("label") && !cmd.Flags().Changed("tags") {
			fmt.Println("No fields to update. Use --hostname, --label, or --tags flags.")
		}

		return nil
	},
}

// accountEnableCmd enables an account (sets it to active).
var accountEnableCmd = &cobra.Command{
	Use:   "enable <id>",
	Short: "Enable an account (set to active)",
	Long:  `Enable an account by setting its status to active.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid account ID: %w", err)
		}

		// Check if account exists and get current status
		allAccounts, err := db.GetAllAccounts()
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

		// If already active, report but don't error
		if account.IsActive {
			fmt.Printf("Account %d is already enabled\n", id)
			return nil
		}

		// Enable the account
		if err := db.ToggleAccountStatus(id); err != nil {
			return fmt.Errorf("failed to enable account: %w", err)
		}

		fmt.Printf("Account %d enabled\n", id)
		return nil
	},
}

// accountDisableCmd disables an account (sets it to inactive).
var accountDisableCmd = &cobra.Command{
	Use:   "disable <id>",
	Short: "Disable an account (set to inactive)",
	Long:  `Disable an account by setting its status to inactive.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid account ID: %w", err)
		}

		// Check if account exists and get current status
		allAccounts, err := db.GetAllAccounts()
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

		// If already inactive, report but don't error
		if !account.IsActive {
			fmt.Printf("Account %d is already disabled\n", id)
			return nil
		}

		// Disable the account
		if err := db.ToggleAccountStatus(id); err != nil {
			return fmt.Errorf("failed to disable account: %w", err)
		}

		fmt.Printf("Account %d disabled\n", id)
		return nil
	},
}

// accountDeleteCmd deletes an account.
var accountDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete an account",
	Long:  `Delete an account and all its associated key assignments.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid account ID: %w", err)
		}

		force, _ := cmd.Flags().GetBool("force")

		// Get account info for confirmation
		allAccounts, err := db.GetAllAccounts()
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

		// Confirm deletion unless --force is used
		if !force {
			fmt.Printf("Delete account: %s@%s (ID: %d)? (yes/no): ", account.Username, account.Hostname, id)
			var response string
			fmt.Scanln(&response)
			if strings.ToLower(response) != "yes" {
				fmt.Println("Deletion cancelled.")
				return nil
			}
		}

		am := db.DefaultAccountManager()
		if am == nil {
			return fmt.Errorf("no account manager available")
		}

		if err := am.DeleteAccount(id); err != nil {
			return fmt.Errorf("failed to delete account: %w", err)
		}

		fmt.Printf("Account deleted: %s@%s\n", account.Username, account.Hostname)
		return nil
	},
}

// accountAssignKeyCmd assigns a key to an account.
var accountAssignKeyCmd = &cobra.Command{
	Use:   "assign-key <account-id> <key-id>",
	Short: "Assign a public key to an account",
	Long: `Assign a public key (by ID) to an account. The key will be deployed
to this account's authorized_keys file on next deploy.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		accountID, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid account ID: %w", err)
		}

		keyID, err := strconv.Atoi(args[1])
		if err != nil {
			return fmt.Errorf("invalid key ID: %w", err)
		}

		km := db.DefaultKeyManager()
		if km == nil {
			return fmt.Errorf("no key manager available")
		}

		// Get account and key for display - verify account exists
		allAccounts, _ := db.GetAllAccounts()
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

		fmt.Printf("Key %d assigned to account %d\n", keyID, accountID)
		return nil
	},
}

// accountUnassignKeyCmd unassigns a key from an account.
var accountUnassignKeyCmd = &cobra.Command{
	Use:   "unassign-key <account-id> <key-id>",
	Short: "Unassign a public key from an account",
	Long: `Remove the assignment of a public key from an account.
The key will no longer be deployed to this account's authorized_keys.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		accountID, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid account ID: %w", err)
		}

		keyID, err := strconv.Atoi(args[1])
		if err != nil {
			return fmt.Errorf("invalid key ID: %w", err)
		}

		km := db.DefaultKeyManager()
		if km == nil {
			return fmt.Errorf("no key manager available")
		}

		if err := km.UnassignKeyFromAccount(keyID, accountID); err != nil {
			return fmt.Errorf("failed to unassign key: %w", err)
		}

		fmt.Printf("Key %d unassigned from account %d\n", keyID, accountID)
		return nil
	},
}

// registerAccountCommands registers all account-related subcommands.
func registerAccountCommands() {
	// Register subcommands with the main account command
	accountCmd.AddCommand(accountListCmd)
	accountCmd.AddCommand(accountShowCmd)
	accountCmd.AddCommand(accountCreateCmd)
	accountCmd.AddCommand(accountUpdateCmd)
	accountCmd.AddCommand(accountEnableCmd)
	accountCmd.AddCommand(accountDisableCmd)
	accountCmd.AddCommand(accountDeleteCmd)
	accountCmd.AddCommand(accountAssignKeyCmd)
	accountCmd.AddCommand(accountUnassignKeyCmd)

	// Setup flags for create (only if not already defined)
	if accountCreateCmd.Flags().Lookup("username") == nil {
		accountCreateCmd.Flags().StringP("username", "u", "", "Username (required)")
		accountCreateCmd.Flags().String("hostname", "", "Hostname (required)")
		accountCreateCmd.Flags().StringP("label", "l", "", "Optional label")
		accountCreateCmd.Flags().String("tags", "", "Optional tags (comma-separated)")
	}

	// Setup flags for update (only if not already defined)
	if accountUpdateCmd.Flags().Lookup("hostname") == nil {
		accountUpdateCmd.Flags().String("hostname", "", "Update hostname")
		accountUpdateCmd.Flags().String("label", "", "Update label")
		accountUpdateCmd.Flags().String("tags", "", "Update tags")
	}

	// Setup flags for delete (only if not already defined)
	if accountDeleteCmd.Flags().Lookup("force") == nil {
		accountDeleteCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
	}

	// Setup flags for list (only if not already defined)
	if accountListCmd.Flags().Lookup("status") == nil {
		accountListCmd.Flags().String("status", "", "Filter by status (active or inactive)")
		accountListCmd.Flags().String("search", "", "Search by username, hostname, or label")
	}
}
