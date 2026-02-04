// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/toeirei/keymaster/internal/core"
	"github.com/toeirei/keymaster/internal/core/db"
	"github.com/toeirei/keymaster/internal/model"
)

var showKeysCmd = &cobra.Command{
	Use:   "show-keys [user@hostname]",
	Short: "Show what authorized_keys content would be deployed to an account",
	Long:  `Displays the authorized_keys content that would be deployed to the specified account without actually deploying it. Useful for debugging key deployment issues.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		identifier := args[0]

		// Parse username@hostname using same logic as deploy command
		parts := splitUsernameHostname(identifier)
		if len(parts) != 2 {
			fmt.Fprintf(os.Stderr, "Invalid format '%s'. Use: user@hostname\n", identifier)
			os.Exit(1)
		}
		username := parts[0]
		hostname := parts[1]

		// Find the account in database
		accounts, err := db.GetAllAccounts()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting accounts: %v\n", err)
			os.Exit(1)
		}

		var dbAccount *model.Account
		for _, acc := range accounts {
			if acc.Hostname == hostname && acc.Username == username {
				dbAccount = &acc
				break
			}
		}

		if dbAccount == nil {
			fmt.Fprintf(os.Stderr, "Account '%s' not found in database\n", identifier)
			os.Exit(1)
		}

		// Generate the authorized_keys content
		content, err := core.GenerateKeysContent(dbAccount.ID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating keys content: %v\n", err)
			os.Exit(1)
		}

		fmt.Println(content)
	},
}

// Helper function to split user@hostname
func splitUsernameHostname(identifier string) []string {
	parts := make([]string, 0, 2)
	idx := -1
	for i := len(identifier) - 1; i >= 0; i-- {
		if identifier[i] == '@' {
			idx = i
			break
		}
	}
	if idx == -1 {
		return parts
	}
	parts = append(parts, identifier[:idx], identifier[idx+1:])
	return parts
}
