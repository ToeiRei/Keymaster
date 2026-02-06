package cli

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	log "github.com/charmbracelet/log"

	"github.com/spf13/cobra"
	"github.com/toeirei/keymaster/internal/core"
	"github.com/toeirei/keymaster/internal/core/db"
	"github.com/toeirei/keymaster/internal/core/model"
	"github.com/toeirei/keymaster/internal/core/security"
	"github.com/toeirei/keymaster/internal/i18n"
	"github.com/toeirei/keymaster/internal/uiadapters"
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

		// Fetch host key (best-effort)
		var hostKey string
		dm := &cliDeployerManager{}
		if hk, herr := dm.GetRemoteHostKey(host); herr == nil {
			hostKey = hk
		}

		// Build transfer package payload (excluding crc)
		payload := map[string]string{
			"magic":                "keymaster-transfer-v1",
			"user":                 user,
			"host":                 host,
			"host_key":             hostKey,
			"transfer_private_key": base64.StdEncoding.EncodeToString([]byte(priv)),
		}
		compact, cerr := json.Marshal(payload)
		if cerr != nil {
			log.Fatalf("failed to marshal transfer payload: %v", cerr)
		}
		// Compute CRC (sha256 hex) over compact payload
		sum := sha256.Sum256(compact)
		crc := hex.EncodeToString(sum[:])

		pkg := map[string]string{
			"magic":                payload["magic"],
			"user":                 payload["user"],
			"host":                 payload["host"],
			"host_key":             payload["host_key"],
			"transfer_private_key": payload["transfer_private_key"],
			"crc":                  crc,
		}
		data, jerr := json.MarshalIndent(pkg, "", "  ")
		if jerr != nil {
			log.Fatalf("failed to marshal transfer package: %v", jerr)
		}

		// Attempt to deactivate the account if it exists and is active
		st := uiadapters.NewStoreAdapter()
		accts, aerr := st.GetAllAccounts()
		if aerr == nil {
			if acc, ferr := core.FindAccountByIdentifier(fmt.Sprintf("%s@%s", user, host), accts); ferr == nil {
				if acc.IsActive {
					if derr := st.SetAccountActiveState(cmd.Context(), acc.ID, false); derr != nil {
						if verbose {
							log.Warnf("warning: failed to deactivate account %s: %v", acc.String(), derr)
						}
					} else {
						log.Infof("Deactivated account on this instance: %s", acc.String())
					}
				}
			}
		}

		if outFile != "" {
			// Ensure directory exists
			if err := os.MkdirAll(filepath.Dir(outFile), 0o700); err != nil {
				log.Fatalf("%v", err)
			}
			if err := os.WriteFile(outFile, data, 0o600); err != nil {
				log.Fatalf("%v", err)
			}
			log.Infof("Wrote transfer package to %s (session: %s)", outFile, sid)
		} else {
			log.Infof("# transfer session: %s", sid)
			log.Info(string(data))
		}
	},
}

func init() {
	transferCreateCmd.Flags().StringP("out", "o", "", "Write transfer package JSON to file (0600)")
	transferCreateCmd.Flags().String("label", "", "Optional label for the transferred account")
	transferCreateCmd.Flags().String("tags", "", "Optional tags for the transferred account")
	transferCmd.AddCommand(transferCreateCmd)
	// transfer accept: accept a transfer by reading a transfer package (JSON)
	var transferAcceptCmd = &cobra.Command{
		Use:     "accept <package.json>",
		Short:   "Accept a transfer by importing a transfer package and bootstrapping the account",
		Args:    cobra.ExactArgs(1),
		PreRunE: setupDefaultServices,
		Run: func(cmd *cobra.Command, args []string) {
			pkgPath := args[0]
			var in []byte
			var err error
			if pkgPath == "-" {
				in, err = io.ReadAll(os.Stdin)
			} else {
				in, err = os.ReadFile(pkgPath)
			}
			if err != nil {
				log.Fatalf("failed to read transfer package: %v", err)
			}
			var pkg map[string]string
			if err := json.Unmarshal(in, &pkg); err != nil {
				log.Fatalf("invalid transfer package: %v", err)
			}

			// Optional overrides
			labelFlag, _ := cmd.Flags().GetString("label")
			tagsFlag, _ := cmd.Flags().GetString("tags")

			// Validate magic
			if pkg["magic"] != "keymaster-transfer-v1" {
				log.Fatalf("unsupported transfer package: magic=%s", pkg["magic"])
			}
			// Recompute CRC and verify
			payload := map[string]string{
				"magic":                pkg["magic"],
				"user":                 pkg["user"],
				"host":                 pkg["host"],
				"host_key":             pkg["host_key"],
				"transfer_private_key": pkg["transfer_private_key"],
			}
			compact, cerr := json.Marshal(payload)
			if cerr != nil {
				log.Fatalf("internal crc error: %v", cerr)
			}
			sum := sha256.Sum256(compact)
			expect := hex.EncodeToString(sum[:])
			if pkg["crc"] != expect {
				log.Fatalf("transfer package CRC mismatch")
			}

			// Decode the base64 private key
			privBytes, derr := base64.StdEncoding.DecodeString(pkg["transfer_private_key"])
			if derr != nil {
				log.Fatalf("invalid base64 private key: %v", derr)
			}

			params := core.BootstrapParams{
				Username:       pkg["user"],
				Hostname:       pkg["host"],
				Label:          labelFlag,
				Tags:           tagsFlag,
				TempPrivateKey: security.FromBytes(privBytes),
				HostKey:        pkg["host_key"],
				SessionID:      pkg["session_id"],
			}

			// Prepare deps using CLI adapters (reuse existing helpers)
			deps := core.BootstrapDeps{
				AddAccount:    func(u, h, l, t string) (int, error) { return uiadapters.NewStoreAdapter().AddAccount(u, h, l, t) },
				DeleteAccount: func(id int) error { return uiadapters.NewStoreAdapter().DeleteAccount(id) },
				AssignKey:     func(kid, aid int) error { return uiadapters.NewStoreAdapter().AssignKeyToAccount(kid, aid) },
				GenerateKeysContent: func(accountID int) (string, error) {
					return uiadapters.NewStoreAdapter().GenerateAuthorizedKeysContent(cmd.Context(), accountID)
				},
				NewBootstrapDeployer: func(hostname, username string, privateKey interface{}, expectedHostKey string) (core.BootstrapDeployer, error) {
					// Normalize to security.Secret for core
					switch v := privateKey.(type) {
					case security.Secret:
						return core.NewBootstrapDeployer(hostname, username, v, expectedHostKey)
					case string:
						return core.NewBootstrapDeployer(hostname, username, security.FromString(v), expectedHostKey)
					case []byte:
						return core.NewBootstrapDeployer(hostname, username, security.FromBytes(v), expectedHostKey)
					default:
						return core.NewBootstrapDeployer(hostname, username, nil, expectedHostKey)
					}
				},
				GetActiveSystemKey: func() (*model.SystemKey, error) { return uiadapters.NewStoreAdapter().GetActiveSystemKey() },
				LogAudit: func(e core.BootstrapAuditEvent) error {
					if w := db.DefaultAuditWriter(); w != nil {
						return w.LogAction(e.Action, e.Details)
					}
					return nil
				},
			}

			res, err := core.PerformBootstrapDeployment(cmd.Context(), params, deps)
			if err != nil {
				log.Fatalf("accept transfer failed: %v", err)
			}
			log.Infof("Accepted transfer: account id=%d %s@%s deployed=%v", res.Account.ID, res.Account.Username, res.Account.Hostname, res.RemoteDeployed)
		},
	}
	transferAcceptCmd.Flags().String("label", "", "Optional label for the new account")
	transferAcceptCmd.Flags().String("tags", "", "Optional tags for the new account")
	transferCmd.AddCommand(transferAcceptCmd)
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
