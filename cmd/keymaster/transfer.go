package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/toeirei/keymaster/internal/core"
	"github.com/toeirei/keymaster/internal/i18n"
)

// transferCmd is the root `transfer` command.
var transferCmd = &cobra.Command{
	Use:   "transfer",
	Short: "Transfer an account to another Keymaster instance",
	Long:  `Helpers for secure transfer/handover of an account between Keymaster instances.`,
}

// transferCreateCmd generates a transfer bootstrap keypair and persists the public
// side as a bootstrap session. The private key PEM is written to --out or stdout.
var transferCreateCmd = &cobra.Command{
	Use:     "create <user@host>",
	Short:   "Create a transfer bootstrap for an account",
	Args:    cobra.ExactArgs(1),
	PreRunE: setupDefaultServices,
	Run: func(cmd *cobra.Command, args []string) {
		target := args[0]
		var user, host string
		if parts := splitUserHost(target); parts != nil {
			user = parts[0]
			host = parts[1]
		} else {
			log.Fatalf("%s", i18n.T("transfer.error_invalid_target", target))
		}

		outFile, _ := cmd.Flags().GetString("out")
		label, _ := cmd.Flags().GetString("label")
		tags, _ := cmd.Flags().GetString("tags")

		sid, priv, err := core.CreateTransferBootstrap(user, host, label, tags)
		if err != nil {
			log.Fatalf("%v", err)
		}

		if outFile != "" {
			// Ensure directory exists
			if err := os.MkdirAll(filepath.Dir(outFile), 0o700); err != nil {
				log.Fatalf("%v", err)
			}
			if err := os.WriteFile(outFile, []byte(priv), 0o600); err != nil {
				log.Fatalf("%v", err)
			}
			fmt.Printf("Wrote transfer private key to %s (session: %s)\n", outFile, sid)
		} else {
			fmt.Printf("# transfer session: %s\n", sid)
			fmt.Print(priv)
		}
	},
}

func init() {
	transferCreateCmd.Flags().StringP("out", "o", "", "Write private key PEM to file (0600)")
	transferCreateCmd.Flags().String("label", "", "Optional label for the transferred account")
	transferCreateCmd.Flags().String("tags", "", "Optional tags for the transferred account")
	transferCmd.AddCommand(transferCreateCmd)
}

// splitUserHost splits a user@host identifier into components.
func splitUserHost(s string) []string {
	if s == "" {
		return nil
	}
	for i := 0; i < len(s); i++ {
		if s[i] == '@' {
			return []string{s[:i], s[i+1:]}
		}
	}
	return nil
}
