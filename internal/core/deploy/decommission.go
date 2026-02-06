// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// Package deploy provides functionality for decommissioning accounts by
// removing their authorized_keys files before deleting from the database.
package deploy // import "github.com/toeirei/keymaster/internal/core/deploy"

import (
	"fmt"
	"os"
	"strings"

	"github.com/toeirei/keymaster/internal/core/db"
	"github.com/toeirei/keymaster/internal/core/logging"
	"github.com/toeirei/keymaster/internal/core/model"
	"github.com/toeirei/keymaster/internal/core/state"
	"github.com/toeirei/keymaster/internal/security"
)

// DecommissionOptions configures how accounts are decommissioned
type DecommissionOptions struct {
	// SkipRemoteCleanup bypasses SSH connection and authorized_keys removal
	SkipRemoteCleanup bool
	// KeepFile removes only Keymaster-managed content, leaves other keys intact
	KeepFile bool
	// Force continues decommission even if remote cleanup fails
	Force bool
	// DryRun shows what would be done without making changes
	DryRun bool
	// SelectiveKeys specifies which keys to remove (by ID). If empty, removes all Keymaster-managed keys
	SelectiveKeys []int
	// RemoveSystemKeyOnly removes only the system key, preserving all user keys
	RemoveSystemKeyOnly bool
}

// DecommissionResult contains the outcome of a decommission operation
type DecommissionResult struct {
	AccountID           int
	AccountString       string
	RemoteCleanupDone   bool
	RemoteCleanupError  error
	DatabaseDeleteDone  bool
	DatabaseDeleteError error
	BackupPath          string
	Skipped             bool
	SkipReason          string
}

// String returns a human-readable summary of the decommission result
func (r DecommissionResult) String() string {
	if r.Skipped {
		return fmt.Sprintf("SKIPPED %s: %s", r.AccountString, r.SkipReason)
	}

	status := "SUCCESS"
	details := []string{}

	if r.RemoteCleanupError != nil {
		status = "PARTIAL"
		details = append(details, fmt.Sprintf("remote cleanup failed: %v", r.RemoteCleanupError))
	} else if r.RemoteCleanupDone {
		details = append(details, "authorized_keys removed")
	}

	if r.DatabaseDeleteError != nil {
		status = "FAILED"
		details = append(details, fmt.Sprintf("database delete failed: %v", r.DatabaseDeleteError))
	} else if r.DatabaseDeleteDone {
		details = append(details, "removed from database")
	}

	if r.BackupPath != "" {
		details = append(details, fmt.Sprintf("backup: %s", r.BackupPath))
	}

	return fmt.Sprintf("%s %s: %s", status, r.AccountString, strings.Join(details, ", "))
}

// DecommissionAccount removes SSH access for an account and deletes it from the database.
// It first attempts to clean up the remote authorized_keys file, then removes the account
// from the database. The operation can be configured with DecommissionOptions.
func DecommissionAccount(account model.Account, systemKey security.Secret, options DecommissionOptions) DecommissionResult {
	result := DecommissionResult{
		AccountID:     account.ID,
		AccountString: account.String(),
	}

	// Log the decommission attempt
	auditAction := "DECOMMISSION_START"
	auditDetails := fmt.Sprintf("Starting decommission of account %s (ID: %d)", account.String(), account.ID)
	if options.DryRun {
		auditAction = "DECOMMISSION_DRYRUN"
		auditDetails = fmt.Sprintf("DRY RUN: Would decommission account %s (ID: %d)", account.String(), account.ID)
	}
	if err := logAction(auditAction, auditDetails); err != nil {
		// Log the error but continue - audit logging shouldn't block decommission
		logging.Warnf("Failed to log audit entry: %v", err)
	}

	if options.DryRun {
		result.Skipped = true
		result.SkipReason = "dry run mode"
		return result
	}

	// Step 1: Remote cleanup (unless skipped)
	if !options.SkipRemoteCleanup {
		var err error
		if len(options.SelectiveKeys) > 0 || options.RemoveSystemKeyOnly {
			// Use selective cleanup
			err = cleanupRemoteAuthorizedKeysSelective(account, systemKey, options, &result)
		} else {
			// Use traditional cleanup
			err = cleanupRemoteAuthorizedKeys(account, systemKey, options.KeepFile, &result)
		}

		if err != nil {
			result.RemoteCleanupError = err
			if !options.Force {
				result.Skipped = true
				result.SkipReason = fmt.Sprintf("remote cleanup failed and --force not specified: %v", err)

				// Log the failure
				_ = logAction("DECOMMISSION_FAILED",
					fmt.Sprintf("Failed to decommission %s: %v", account.String(), err))
				return result
			}
			// With --force, we continue despite remote cleanup failure
		}
	}

	// Step 2: Database cleanup
	mgr := db.DefaultAccountManager()
	if mgr == nil {
		result.DatabaseDeleteError = fmt.Errorf("no account manager configured")
		_ = logAction("DECOMMISSION_FAILED",
			fmt.Sprintf("Failed to delete account %s from database: %v", account.String(), result.DatabaseDeleteError))
		return result
	}
	if err := mgr.DeleteAccount(account.ID); err != nil {
		result.DatabaseDeleteError = err
		_ = logAction("DECOMMISSION_FAILED",
			fmt.Sprintf("Failed to delete account %s from database: %v", account.String(), err))
		return result
	}
	result.DatabaseDeleteDone = true

	// Log successful decommission
	details := fmt.Sprintf("Successfully decommissioned account %s (ID: %d)", account.String(), account.ID)
	if result.RemoteCleanupError != nil {
		details += fmt.Sprintf(" - Warning: remote cleanup failed: %v", result.RemoteCleanupError)
	}
	if result.BackupPath != "" {
		details += fmt.Sprintf(" - Backup created: %s", result.BackupPath)
	}
	_ = logAction("DECOMMISSION_SUCCESS", details)

	return result
}

// cleanupRemoteAuthorizedKeys connects to the remote host and removes the authorized_keys file
func cleanupRemoteAuthorizedKeys(account model.Account, systemKey security.Secret, keepFile bool, result *DecommissionResult) error {
	// Get passphrase from cache and ensure it's wiped after use.
	passphrase := state.PasswordCache.Get()
	defer func() {
		for i := range passphrase {
			passphrase[i] = 0
		}
	}()

	// Create deployer connection
	deployer, err := NewDeployerFunc(account.Hostname, account.Username, systemKey, passphrase)
	if err != nil {
		return fmt.Errorf("failed to connect to %s@%s: %w", account.Username, account.Hostname, err)
	}
	defer deployer.Close()

	if keepFile {
		// Remove only Keymaster-managed content, preserve other keys
		return removeKeymasterContent(deployer, result, account.ID)
	} else {
		// Remove the entire authorized_keys file
		return removeAuthorizedKeysFile(deployer, result)
	}
}

// cleanupRemoteAuthorizedKeysSelective connects to the remote host and removes specific keys
func cleanupRemoteAuthorizedKeysSelective(account model.Account, systemKey security.Secret, options DecommissionOptions, result *DecommissionResult) error {
	// Get passphrase from cache and ensure it's wiped after use.
	passphrase := state.PasswordCache.Get()
	defer func() {
		for i := range passphrase {
			passphrase[i] = 0
		}
	}()

	// Create deployer connection
	deployer, err := NewDeployerFunc(account.Hostname, account.Username, systemKey, passphrase)
	if err != nil {
		return fmt.Errorf("failed to connect to %s@%s: %w", account.Username, account.Hostname, err)
	}
	defer deployer.Close()

	if options.RemoveSystemKeyOnly {
		// Remove only the system key, keep all user keys
		return removeSelectiveKeymasterContent(deployer, result, account.ID, nil, true)
	} else if len(options.SelectiveKeys) > 0 {
		// Remove specific keys (system key is always removed in decommission)
		return removeSelectiveKeymasterContent(deployer, result, account.ID, options.SelectiveKeys, true)
	} else if options.KeepFile {
		// Remove all Keymaster-managed content, preserve other keys
		return removeKeymasterContent(deployer, result, account.ID)
	} else {
		// Remove the entire authorized_keys file
		return removeAuthorizedKeysFile(deployer, result)
	}
}

// removeAuthorizedKeysFile completely removes the authorized_keys file
func removeAuthorizedKeysFile(deployer *Deployer, result *DecommissionResult) error {
	authorizedKeysPath := ".ssh/authorized_keys"

	// Check if file exists
	if _, err := deployer.sftp.Stat(authorizedKeysPath); err != nil {
		// File doesn't exist, nothing to remove. Accept common forms of "not found"
		if os.IsNotExist(err) || strings.Contains(err.Error(), "file does not exist") || strings.Contains(err.Error(), "no such file") {
			return nil
		}
		return fmt.Errorf("failed to check authorized_keys file: %w", err)
	}

	// Remove the file
	if err := deployer.sftp.Remove(authorizedKeysPath); err != nil {
		return fmt.Errorf("failed to remove authorized_keys: %w", err)
	}
	result.RemoteCleanupDone = true

	return nil
}

// removeKeymasterContent removes only the Keymaster-managed section from authorized_keys
func removeKeymasterContent(deployer *Deployer, result *DecommissionResult, accountID int) error {
	return removeSelectiveKeymasterContent(deployer, result, accountID, nil, true)
}

// removeSelectiveKeymasterContent removes specific keys from the Keymaster-managed section
func removeSelectiveKeymasterContent(deployer *Deployer, result *DecommissionResult, accountID int, excludeKeyIDs []int, removeSystemKey bool) error {
	authorizedKeysPath := ".ssh/authorized_keys"

	// Read current content
	content, err := deployer.GetAuthorizedKeys()
	if err != nil {
		// File doesn't exist, nothing to remove
		if strings.Contains(err.Error(), "no such file") {
			return nil
		}
		return fmt.Errorf("failed to read authorized_keys: %w", err)
	}

	// Parse content and extract non-Keymaster content
	nonKeymasterContent := extractNonKeymasterContent(string(content))

	var finalContent string
	if removeSystemKey && len(excludeKeyIDs) == 0 {
		// Remove system key entirely, keep all user keys
		// This is used when we only want to revoke Keymaster's access
		keymasterContent, err := GenerateSelectiveKeysContent(accountID, 0, nil, true)
		if err != nil {
			return fmt.Errorf("failed to generate keys content: %w", err)
		}
		// Check if content is empty, but don't trim (to preserve trailing newlines)
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
		// Regenerate Keymaster section without excluded keys and/or without system key
		logging.Debugf("DEBUG decommission: removeSystemKey=%v, excludeKeyIDs=%v, accountID=%d", removeSystemKey, excludeKeyIDs, accountID)
		keymasterContent, err := GenerateSelectiveKeysContent(accountID, 0, excludeKeyIDs, removeSystemKey)
		if err != nil {
			return fmt.Errorf("failed to generate selective keys content: %w", err)
		}
		logging.Debugf("DEBUG decommission: Generated keymaster content length=%d", len(keymasterContent))
		logging.Debugf("DEBUG decommission: Keymaster content (truncated): %s", keymasterContent)
		logging.Debugf("DEBUG decommission: Non-Keymaster content (truncated): %s", nonKeymasterContent)
		// Check if content is empty, but don't trim (to preserve trailing newlines)
		hasKeymasterContent := strings.TrimSpace(keymasterContent) != ""
		hasNonKeymasterContent := strings.TrimSpace(nonKeymasterContent) != ""

		if hasKeymasterContent {
			if hasNonKeymasterContent {
				finalContent = keymasterContent + "\n" + nonKeymasterContent
			} else {
				finalContent = keymasterContent
			}
		} else {
			// No Keymaster keys remain, only non-Keymaster content
			finalContent = nonKeymasterContent
		}
		logging.Debugf("DEBUG decommission: Final content (truncated): %s", finalContent)
	} else {
		// Remove entire Keymaster-managed section
		finalContent = nonKeymasterContent
	}

	if strings.TrimSpace(finalContent) == "" {
		// No content remains, remove the file entirely
		if err := deployer.sftp.Remove(authorizedKeysPath); err != nil {
			return fmt.Errorf("failed to remove empty authorized_keys file: %w", err)
		}
	} else {
		// Write cleaned content back
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

		// Check for Keymaster header (start of managed section)
		if strings.HasPrefix(trimmedLine, "# Keymaster Managed Keys") {
			inKeymasterSection = true
			continue
		}

		if inKeymasterSection {
			// Still in Keymaster section - skip all lines until we find a clear end marker
			// The Keymaster section ends when we encounter two consecutive empty lines
			// OR when we find a line that doesn't start with ssh-, command=, or #

			// Check if this is a Keymaster-managed line
			isKeymasterLine := trimmedLine == "" ||
				strings.HasPrefix(trimmedLine, "#") ||
				strings.HasPrefix(trimmedLine, "ssh-") ||
				strings.HasPrefix(trimmedLine, "ecdsa-") ||
				strings.HasPrefix(trimmedLine, "command=")

			if !isKeymasterLine {
				// Found a line that's clearly not part of Keymaster section
				inKeymasterSection = false
				if trimmedLine != "" {
					result = append(result, line)
				}
			}
			// Skip all Keymaster lines
			continue
		}

		// Include non-Keymaster lines (preserve original line with spacing)
		if !inKeymasterSection {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

// (removed unused helper: removeKeymasterManagedSection)

// BulkDecommissionAccounts decommissions multiple accounts with progress reporting
func BulkDecommissionAccounts(accounts []model.Account, systemKey security.Secret, options DecommissionOptions) []DecommissionResult {
	results := make([]DecommissionResult, 0, len(accounts))

	for i, account := range accounts {
		logging.Infof("Decommissioning account %d/%d: %s", i+1, len(accounts), account.String())

		result := DecommissionAccount(account, systemKey, options)
		results = append(results, result)

		// Print immediate result
		logging.Infof("  → %s", result.String())
	}

	return results
}
