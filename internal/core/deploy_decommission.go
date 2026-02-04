// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core

import (
	"fmt"
	"strings"

	"github.com/toeirei/keymaster/internal/core/state"
	"github.com/toeirei/keymaster/internal/logging"
	"github.com/toeirei/keymaster/internal/model"
	"github.com/toeirei/keymaster/internal/security"
)

// DecommissionAccount removes SSH access for an account and deletes it from the database.
// This implementation is owned by core and uses the `NewDeployerFactory` abstraction
// so tests can inject fakes. It intentionally uses DeployAuthorizedKeys to write
// back cleaned content (and writes an empty file when deletion would previously occur),
// avoiding direct sftp manipulations from core.
func DecommissionAccount(account model.Account, systemKey security.Secret, options DecommissionOptions) DecommissionResult {
	result := DecommissionResult{
		AccountID:     account.ID,
		AccountString: account.String(),
	}

	auditAction := "DECOMMISSION_START"
	auditDetails := fmt.Sprintf("Starting decommission of account %s (ID: %d)", account.String(), account.ID)
	if options.DryRun {
		auditAction = "DECOMMISSION_DRYRUN"
		auditDetails = fmt.Sprintf("DRY RUN: Would decommission account %s (ID: %d)", account.String(), account.ID)
	}
	if w := DefaultAuditWriter(); w != nil {
		_ = w.LogAction(auditAction, auditDetails)
	}

	if options.DryRun {
		result.Skipped = true
		result.SkipReason = "dry run mode"
		return result
	}

	if !options.SkipRemoteCleanup {
		var err error
		if len(options.SelectiveKeys) > 0 {
			err = cleanupRemoteAuthorizedKeysSelective(account, systemKey, options, &result)
		} else {
			err = cleanupRemoteAuthorizedKeys(account, systemKey, options.KeepFile, &result)
		}

		if err != nil {
			result.RemoteCleanupError = err
			if !options.Force {
				result.Skipped = true
				result.SkipReason = fmt.Sprintf("remote cleanup failed and --force not specified: %v", err)
				if w := DefaultAuditWriter(); w != nil {
					_ = w.LogAction("DECOMMISSION_FAILED", fmt.Sprintf("Failed to decommission %s: %v", account.String(), err))
				}
				return result
			}
		}
	}

	mgr := DefaultAccountManager()
	if mgr == nil {
		result.DatabaseDeleteError = fmt.Errorf("no account manager configured")
		if w := DefaultAuditWriter(); w != nil {
			_ = w.LogAction("DECOMMISSION_FAILED", fmt.Sprintf("Failed to delete account %s from database: %v", account.String(), result.DatabaseDeleteError))
		}
		return result
	}
	if err := mgr.DeleteAccount(account.ID); err != nil {
		result.DatabaseDeleteError = err
		if w := DefaultAuditWriter(); w != nil {
			_ = w.LogAction("DECOMMISSION_FAILED", fmt.Sprintf("Failed to delete account %s from database: %v", account.String(), err))
		}
		return result
	}
	result.DatabaseDeleteDone = true

	details := fmt.Sprintf("Successfully decommissioned account %s (ID: %d)", account.String(), account.ID)
	if result.RemoteCleanupError != nil {
		details += fmt.Sprintf(" - Warning: remote cleanup failed: %v", result.RemoteCleanupError)
	}
	if result.BackupPath != "" {
		details += fmt.Sprintf(" - Backup created: %s", result.BackupPath)
	}
	if w := DefaultAuditWriter(); w != nil {
		_ = w.LogAction("DECOMMISSION_SUCCESS", details)
	}

	return result
}

// BulkDecommissionAccounts decommissions multiple accounts with progress reporting
func BulkDecommissionAccounts(accounts []model.Account, systemKey security.Secret, options DecommissionOptions) []DecommissionResult {
	results := make([]DecommissionResult, 0, len(accounts))

	for i, account := range accounts {
		logging.Infof("Decommissioning account %d/%d: %s", i+1, len(accounts), account.String())

		result := DecommissionAccount(account, systemKey, options)
		results = append(results, result)

		logging.Infof("  → %s", result.AccountString)
	}

	return results
}

// cleanupRemoteAuthorizedKeys connects to the remote host and removes or updates the authorized_keys content
func cleanupRemoteAuthorizedKeys(account model.Account, systemKey security.Secret, keepFile bool, result *DecommissionResult) error {
	passphrase := state.PasswordCache.Get()
	defer func() {
		for i := range passphrase {
			passphrase[i] = 0
		}
	}()

	deployer, err := NewDeployerFactory(account.Hostname, account.Username, systemKey, passphrase)
	if err != nil {
		return fmt.Errorf("failed to connect to %s@%s: %w", account.Username, account.Hostname, err)
	}
	defer deployer.Close()

	if keepFile {
		return removeKeymasterContent(deployer, result, account.ID)
	}
	// When remove file behavior was required previously, we now write an empty file
	// by deploying empty content to the host to avoid requiring sftp removal APIs.
	if err := deployer.DeployAuthorizedKeys(""); err != nil {
		return fmt.Errorf("failed to remove authorized_keys: %w", err)
	}
	result.RemoteCleanupDone = true
	return nil
}

// cleanupRemoteAuthorizedKeysSelective removes specific keys or sections using DeployAuthorizedKeys
func cleanupRemoteAuthorizedKeysSelective(account model.Account, systemKey security.Secret, options DecommissionOptions, result *DecommissionResult) error {
	passphrase := state.PasswordCache.Get()
	defer func() {
		for i := range passphrase {
			passphrase[i] = 0
		}
	}()

	deployer, err := NewDeployerFactory(account.Hostname, account.Username, systemKey, passphrase)
	if err != nil {
		return fmt.Errorf("failed to connect to %s@%s: %w", account.Username, account.Hostname, err)
	}
	defer deployer.Close()

	if len(options.SelectiveKeys) > 0 {
		return removeSelectiveKeymasterContent(deployer, result, account.ID, options.SelectiveKeys, true)
	} else if options.KeepFile {
		return removeKeymasterContent(deployer, result, account.ID)
	} else {
		if err := deployer.DeployAuthorizedKeys(""); err != nil {
			return fmt.Errorf("failed to remove authorized_keys: %w", err)
		}
		result.RemoteCleanupDone = true
		return nil
	}
}

// removeKeymasterContent removes only the Keymaster-managed section from authorized_keys
func removeKeymasterContent(deployer RemoteDeployer, result *DecommissionResult, accountID int) error {
	return removeSelectiveKeymasterContent(deployer, result, accountID, nil, true)
}

// removeSelectiveKeymasterContent removes specific keys from the Keymaster-managed section
func removeSelectiveKeymasterContent(deployer RemoteDeployer, result *DecommissionResult, accountID int, excludeKeyIDs []int, removeSystemKey bool) error {
	content, err := deployer.GetAuthorizedKeys()
	if err != nil {
		if strings.Contains(err.Error(), "no such file") {
			return nil
		}
		return fmt.Errorf("failed to read authorized_keys: %w", err)
	}

	nonKeymasterContent := extractNonKeymasterContent(string(content))

	var finalContent string
	if removeSystemKey && len(excludeKeyIDs) == 0 {
		keymasterContent, err := GenerateSelectiveKeysContent(accountID, 0, nil, true)
		if err != nil {
			return fmt.Errorf("failed to generate keys content: %w", err)
		}
		hasKeymasterContent := strings.TrimSpace(keymasterContent) != ""
		hasNonKeymasterContent := strings.TrimSpace(nonKeymasterContent) != ""

		if hasKeymasterContent {
			if hasNonKeymasterContent {
				finalContent = keymasterContent + "\n" + nonKeymasterContent
			} else {
				finalContent = keymasterContent
			}
		} else {
			finalContent = nonKeymasterContent
		}
	} else if len(excludeKeyIDs) > 0 || removeSystemKey {
		keymasterContent, err := GenerateSelectiveKeysContent(accountID, 0, excludeKeyIDs, removeSystemKey)
		if err != nil {
			return fmt.Errorf("failed to generate selective keys content: %w", err)
		}
		hasKeymasterContent := strings.TrimSpace(keymasterContent) != ""
		hasNonKeymasterContent := strings.TrimSpace(nonKeymasterContent) != ""

		if hasKeymasterContent {
			if hasNonKeymasterContent {
				finalContent = keymasterContent + "\n" + nonKeymasterContent
			} else {
				finalContent = keymasterContent
			}
		} else {
			finalContent = nonKeymasterContent
		}
	} else {
		finalContent = nonKeymasterContent
	}

	if strings.TrimSpace(finalContent) == "" {
		// Deploy an empty content to replace the file rather than direct removal
		if err := deployer.DeployAuthorizedKeys(""); err != nil {
			return fmt.Errorf("failed to remove empty authorized_keys file: %w", err)
		}
	} else {
		if err := deployer.DeployAuthorizedKeys(finalContent); err != nil {
			return fmt.Errorf("failed to update authorized_keys: %w", err)
		}
	}

	result.RemoteCleanupDone = true
	return nil
}

// extractNonKeymasterContent extracts all content that is not managed by Keymaster
func extractNonKeymasterContent(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	inKeymasterSection := false

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimmedLine, "# Keymaster Managed Keys") {
			inKeymasterSection = true
			continue
		}

		if inKeymasterSection {
			isKeymasterLine := trimmedLine == "" || strings.HasPrefix(trimmedLine, "#") || strings.HasPrefix(trimmedLine, "ssh-") || strings.HasPrefix(trimmedLine, "ecdsa-") || strings.HasPrefix(trimmedLine, "command=")
			if !isKeymasterLine {
				inKeymasterSection = false
				if trimmedLine != "" {
					result = append(result, line)
				}
			}
			continue
		}

		if !inKeymasterSection {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}
