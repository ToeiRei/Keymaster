package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
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
version-control access. A database becomes the source of truth.`,
}

func init() {
	// Here we will define our flags and configuration settings.
	// Example: rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.keymaster.yaml)")

	// Add commands
	rootCmd.AddCommand(deployCmd)
	rootCmd.AddCommand(rotateKeyCmd)
	rootCmd.AddCommand(auditCmd)
	rootCmd.AddCommand(uiCmd)
}

var deployCmd = &cobra.Command{
	Use:   "deploy [host]",
	Short: "Deploy authorized_keys to one or all hosts",
	Long:  `Renders the authorized_keys file from the database state and deploys it to the specified host(s).`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Running deployment...")
		// TODO:
		// 1. Connect to DB
		// 2. Get target hosts (all or from args)
		// 3. For each host (concurrently):
		//    a. Generate authorized_keys content
		//    b. SCP to /tmp/authorized_keys.new
		//    c. SSH and mv to final destination
		//    d. Update sequence number in DB
		fmt.Println("Deployment complete.")
	},
}

var rotateKeyCmd = &cobra.Command{
	Use:   "rotate-key [account@host]",
	Short: "Safely rotate a system key for an account",
	Long: `Rotates the Keymaster-managed system key for a given account.
This is a multi-step process to ensure no loss of access.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Rotating system key...")
		// TODO:
		// 1. Generate new key pair
		// 2. Deploy authorized_keys with BOTH old and new system keys
		// 3. Verify new key works on all relevant hosts
		// 4. Deploy authorized_keys with ONLY the new system key
		// 5. Prune old key from database
		fmt.Println("Key rotation complete.")
	},
}

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Audit hosts for configuration drift",
	Long:  `Connects to all managed hosts and checks if their deployed authorized_keys file matches the central database state.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Running audit...")
		// TODO:
		// 1. Connect to DB
		// 2. For each host (concurrently):
		//    a. SSH and read the first line of authorized_keys
		//    b. Parse the sequence number
		//    c. Compare with sequence number in DB and report drift
		fmt.Println("Audit complete.")
	},
}

var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Launch the Keymaster interactive TUI console",
	Long:  `Starts a terminal-based user interface for managing hosts, users, and keys interactively.`,
	Run: func(cmd *cobra.Command, args []string) {
		tui.Run()
	},
}
