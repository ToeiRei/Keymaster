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
	"github.com/toeirei/keymaster/core"
	"github.com/toeirei/keymaster/core/model"
	"github.com/toeirei/keymaster/uiadapters"
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

		st := uiadapters.NewStoreAdapter()
		accounts, err := core.ListAccounts(st, statusFilter, searchTerm)
		if err != nil {
			return err
		}
		if len(accounts) == 0 {
			fmt.Println("No accounts found.")
			return nil
		}
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
		st := uiadapters.NewStoreAdapter()
		account, err := core.ShowAccount(st, identifier)
		if err != nil {
			return err
		}
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
		km := core.DefaultKeyManager()
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
		am := uiadapters.NewStoreAdapter()
		id, err := core.CreateAccount(am, username, hostname, label, tags)
		if err != nil {
			return err
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
		st := uiadapters.NewStoreAdapter()
		var hostnamePtr, labelPtr, tagsPtr *string
		if cmd.Flags().Changed("hostname") {
			hostname, _ := cmd.Flags().GetString("hostname")
			hostnamePtr = &hostname
		}
		if cmd.Flags().Changed("label") {
			label, _ := cmd.Flags().GetString("label")
			labelPtr = &label
		}
		if cmd.Flags().Changed("tags") {
			tags, _ := cmd.Flags().GetString("tags")
			tagsPtr = &tags
		}
		err = core.UpdateAccount(st, id, hostnamePtr, labelPtr, tagsPtr)
		if err != nil {
			return err
		}
		if hostnamePtr != nil {
			fmt.Printf("Hostname updated to: %s\n", *hostnamePtr)
		}
		if labelPtr != nil {
			fmt.Printf("Label updated to: %s\n", *labelPtr)
		}
		if tagsPtr != nil {
			fmt.Printf("Tags updated to: %s\n", *tagsPtr)
		}
		if hostnamePtr == nil && labelPtr == nil && tagsPtr == nil {
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
		st := uiadapters.NewStoreAdapter()
		err = core.EnableAccount(st, id)
		if err != nil {
			return err
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
		st := uiadapters.NewStoreAdapter()
		err = core.DisableAccount(st, id)
		if err != nil {
			return err
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
		st := uiadapters.NewStoreAdapter()
		am := uiadapters.NewStoreAdapter()
		confirmFunc := func(account *model.Account) bool {
			if force {
				return true
			}
			fmt.Printf("Delete account: %s@%s (ID: %d)? (yes/no): ", account.Username, account.Hostname, id)
			var response string
			fmt.Scanln(&response)
			if strings.ToLower(response) != "yes" {
				fmt.Println("Deletion cancelled.")
				return false
			}
			return true
		}
		err = core.DeleteAccount(am, st, id, force, confirmFunc)
		if err != nil {
			return err
		}
		fmt.Printf("Account deleted (ID: %d)\n", id)
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
		km := core.DefaultKeyManager()
		st := uiadapters.NewStoreAdapter()
		if km == nil {
			return fmt.Errorf("no key manager available")
		}
		err = core.AssignKeyToAccount(func(k, a int) error { return km.AssignKeyToAccount(k, a) }, st, keyID, accountID)
		if err != nil {
			return err
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
		km := core.DefaultKeyManager()
		if km == nil {
			return fmt.Errorf("no key manager available")
		}
		err = core.UnassignKeyFromAccount(func(k, a int) error { return km.UnassignKeyFromAccount(k, a) }, keyID, accountID)
		if err != nil {
			return err
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
