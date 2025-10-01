// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// package deploy provides drift detection and remediation functionality.
package deploy // import "github.com/toeirei/keymaster/internal/deploy"

import (
	"fmt"
	"strings"

	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/i18n"
	"github.com/toeirei/keymaster/internal/model"
	"github.com/toeirei/keymaster/internal/sshkey"
)

// AnalyzeDrift detects and analyzes configuration drift for a given account.
// It compares the actual authorized_keys content on the remote host with the
// expected content generated from the database state.
func AnalyzeDrift(account model.Account) (*model.DriftAnalysis, error) {
	analysis := &model.DriftAnalysis{
		HasDrift:       false,
		Classification: model.DriftInfo,
	}

	// If account has never been deployed, it's not technically drift
	if account.Serial == 0 {
		return analysis, nil
	}

	// Get the system key the account should be using
	connectKey, err := db.GetSystemKeyBySerial(account.Serial)
	if err != nil {
		return nil, fmt.Errorf("failed to get system key for serial %d: %w", account.Serial, err)
	}
	if connectKey == nil {
		return nil, fmt.Errorf("no system key found with serial %d", account.Serial)
	}

	// Connect to the host
	deployer, err := NewDeployer(account.Hostname, account.Username, connectKey.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to host: %w", err)
	}
	defer deployer.Close()

	// Read actual content from remote host
	remoteContentBytes, err := deployer.GetAuthorizedKeys()
	if err != nil {
		return nil, fmt.Errorf("failed to read remote authorized_keys: %w", err)
	}

	// Generate expected content
	expectedContent, err := GenerateKeysContent(account.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate expected content: %w", err)
	}

	// Store raw contents for detailed analysis
	analysis.RawActualContent = string(remoteContentBytes)
	analysis.RawExpectedContent = expectedContent

	// Normalize both for comparison
	normalize := func(s string) string {
		s = strings.ReplaceAll(s, "\r\n", "\n")
		s = strings.TrimSpace(s)
		return s
	}

	normalizedActual := normalize(analysis.RawActualContent)
	normalizedExpected := normalize(analysis.RawExpectedContent)

	// If content matches exactly, no drift
	if normalizedActual == normalizedExpected {
		return analysis, nil
	}

	// Drift detected - perform detailed analysis
	analysis.HasDrift = true

	// Parse the serial from the remote file
	lines := strings.Split(normalizedActual, "\n")
	if len(lines) > 0 {
		firstLine := strings.TrimSpace(lines[0])
		if strings.HasPrefix(firstLine, "#") {
			parsedSerial, err := sshkey.ParseSerial(firstLine)
			if err == nil {
				analysis.ActualSerial = parsedSerial
			}
		}
	}

	// Check for missing Keymaster header
	if !strings.Contains(normalizedActual, "# Keymaster Managed Keys") {
		analysis.MissingKeymasterHeader = true
		analysis.Classification = model.DriftCritical
	}

	// Get the active system key serial
	activeKey, err := db.GetActiveSystemKey()
	if err == nil && activeKey != nil {
		analysis.ExpectedSerial = activeKey.Serial
		if analysis.ActualSerial != analysis.ExpectedSerial {
			analysis.SerialMismatch = true
			analysis.Classification = model.DriftCritical
		}
	}

	// Parse keys from both contents to find missing/extra keys
	expectedKeys := parseKeysFromContent(normalizedExpected)
	actualKeys := parseKeysFromContent(normalizedActual)

	// Find missing keys (in expected but not in actual)
	for _, expectedKey := range expectedKeys {
		found := false
		for _, actualKey := range actualKeys {
			if keysMatch(expectedKey, actualKey) {
				found = true
				break
			}
		}
		if !found && !isSystemKey(expectedKey) {
			// Try to find the key in our database for better reporting
			if pk := findPublicKeyByLine(expectedKey); pk != nil {
				analysis.MissingKeys = append(analysis.MissingKeys, *pk)
			}
		}
	}

	// Find extra keys (in actual but not in expected)
	for _, actualKey := range actualKeys {
		found := false
		for _, expectedKey := range expectedKeys {
			if keysMatch(actualKey, expectedKey) {
				found = true
				break
			}
		}
		if !found && !isSystemKey(actualKey) {
			analysis.ExtraKeys = append(analysis.ExtraKeys, actualKey)
		}
	}

	// Classify drift severity
	if analysis.Classification != model.DriftCritical {
		if len(analysis.MissingKeys) > 0 {
			analysis.Classification = model.DriftWarning
		} else if len(analysis.ExtraKeys) > 0 {
			analysis.Classification = model.DriftInfo
		}
	}

	return analysis, nil
}

// RemediateAccount attempts to fix detected drift by deploying the correct configuration.
func RemediateAccount(account model.Account, dryRun bool) (*model.RemediationResult, error) {
	result := &model.RemediationResult{
		Success: false,
	}

	// First, analyze the drift
	analysis, err := AnalyzeDrift(account)
	if err != nil {
		result.Error = err
		return result, err
	}

	if !analysis.HasDrift {
		result.Success = true
		result.Details = i18n.T("remediation.no_drift")
		return result, nil
	}

	result.FixedDriftType = analysis.Classification
	result.PreRemediationSerial = analysis.ActualSerial

	if dryRun {
		result.Details = fmt.Sprintf("Dry-run: Would remediate %s drift (missing keys: %d, extra keys: %d)",
			analysis.Classification, len(analysis.MissingKeys), len(analysis.ExtraKeys))
		result.Success = true
		return result, nil
	}

	// Perform actual remediation by running a deployment
	err = RunDeploymentForAccount(account, false)
	if err != nil {
		result.Error = err
		result.Details = fmt.Sprintf("Failed to deploy: %v", err)
		return result, err
	}

	// Get the active system key to report post-remediation serial
	activeKey, err := db.GetActiveSystemKey()
	if err == nil && activeKey != nil {
		result.PostRemediationSerial = activeKey.Serial
	}

	result.Success = true
	result.DeployedKeys = len(analysis.MissingKeys)
	result.Details = fmt.Sprintf("Successfully remediated %s drift", analysis.Classification)

	return result, nil
}

// parseKeysFromContent extracts individual key lines from authorized_keys content.
func parseKeysFromContent(content string) []string {
	var keys []string
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip empty lines and comments (except the system key which starts with restrictions)
		if line == "" || (strings.HasPrefix(line, "#") && !strings.Contains(line, "command=")) {
			continue
		}
		keys = append(keys, line)
	}
	return keys
}

// keysMatch compares two SSH key lines for equality (ignoring whitespace variations).
func keysMatch(key1, key2 string) bool {
	// Normalize whitespace
	normalize := func(s string) string {
		fields := strings.Fields(s)
		return strings.Join(fields, " ")
	}
	return normalize(key1) == normalize(key2)
}

// isSystemKey checks if a key line is the Keymaster system key.
func isSystemKey(keyLine string) bool {
	return strings.Contains(keyLine, "keymaster-system-key") ||
		strings.Contains(keyLine, "command=\"internal-sftp\"")
}

// findPublicKeyByLine attempts to find a PublicKey model from the database
// that matches the given key line.
func findPublicKeyByLine(keyLine string) *model.PublicKey {
	// Extract the comment (last field) from the key line
	fields := strings.Fields(keyLine)
	if len(fields) < 2 {
		return nil
	}

	// Try to get the key by comment if it exists
	if len(fields) >= 3 {
		comment := fields[len(fields)-1]
		pk, err := db.GetPublicKeyByComment(comment)
		if err == nil && pk != nil {
			return pk
		}
	}

	return nil
}
