// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// package deploy provides functionality for connecting to remote hosts via SSH
// and managing their authorized_keys files. This file contains the logic for
// generating the content of an authorized_keys file from database records.
package deploy // import "github.com/toeirei/keymaster/internal/core/deploy"

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/toeirei/keymaster/internal/core/db"
	"github.com/toeirei/keymaster/internal/keys"
	"github.com/toeirei/keymaster/internal/model"
)

// SystemKeyRestrictions defines the SSH options applied to the Keymaster system key.
// These restrictions limit the key to only allow SFTP access for file management,
// enhancing security by preventing shell access, port forwarding, etc.
const SystemKeyRestrictions = `command="internal-sftp",no-port-forwarding,no-x11-forwarding,no-agent-forwarding,no-pty`

// GenerateKeysContent constructs the authorized_keys file content for a given account.
// It defaults to using the currently active system key.
func GenerateKeysContent(accountID int) (string, error) {
	activeKey, err := db.GetActiveSystemKey()
	if err != nil {
		return "", fmt.Errorf("could not retrieve active system key: %w", err)
	}
	if activeKey == nil {
		return "", fmt.Errorf("no active system key found. please generate one first")
	}
	return GenerateKeysContentForSerial(accountID, activeKey.Serial)
}

// GenerateKeysContentForSerial constructs the authorized_keys file content for a given account using a specific system key serial.
func GenerateKeysContentForSerial(accountID int, serial int) (string, error) {
	// Fetch system key
	systemKey, err := db.GetSystemKeyBySerial(serial)
	if err != nil {
		return "", fmt.Errorf("could not retrieve active system key: %w", err)
	}
	if systemKey == nil {
		return "", fmt.Errorf("no active system key found. please generate one first")
	}

	km := db.DefaultKeyManager()
	if km == nil {
		return "", fmt.Errorf("no key manager available")
	}
	globalKeys, err := km.GetGlobalPublicKeys()
	if err != nil {
		return "", fmt.Errorf("could not retrieve global public keys: %w", err)
	}
	accountKeys, err := km.GetKeysForAccount(accountID)
	if err != nil {
		return "", fmt.Errorf("could not retrieve keys for account ID %d: %w", accountID, err)
	}

	// Delegate pure formatting/sorting/dedup to keys helper
	return keys.BuildAuthorizedKeysContent(systemKey, globalKeys, accountKeys)
}

// GenerateSelectiveKeysContent constructs authorized_keys content excluding specific keys.
// The system key is always included unless removeSystemKey is true.
func GenerateSelectiveKeysContent(accountID int, serial int, excludeKeyIDs []int, removeSystemKey bool) (string, error) {
	var content strings.Builder

	// 1. Get the system key (always included unless removeSystemKey is true)
	if !removeSystemKey {
		// If serial==0, treat it as "use active system key"
		if serial == 0 {
			activeKey, err := db.GetActiveSystemKey()
			if err != nil {
				return "", fmt.Errorf("could not retrieve active system key: %w", err)
			}
			if activeKey == nil {
				return "", fmt.Errorf("no active system key found. please generate one first")
			}
			serial = activeKey.Serial
		}

		systemKey, err := db.GetSystemKeyBySerial(serial)
		if err != nil {
			return "", fmt.Errorf("could not retrieve system key: %w", err)
		}
		if systemKey == nil {
			return "", fmt.Errorf("no system key found for serial %d", serial)
		}

		// Add the Keymaster header and the restricted system key.
		content.WriteString(fmt.Sprintf("# Keymaster Managed Keys (Serial: %d)\n", systemKey.Serial))
		restrictedSystemKey := fmt.Sprintf("%s %s", SystemKeyRestrictions, systemKey.PublicKey)
		content.WriteString(restrictedSystemKey)
	}

	// 2. Get all global public keys.
	// Reuse KeyManager for both global and account keys.
	km := db.DefaultKeyManager()
	if km == nil {
		return "", fmt.Errorf("no key manager available")
	}
	globalKeys, err := km.GetGlobalPublicKeys()
	if err != nil {
		return "", fmt.Errorf("could not retrieve global public keys: %w", err)
	}

	// 3. Get keys specifically assigned to this account.
	accountKeys, err := km.GetKeysForAccount(accountID)
	if err != nil {
		return "", fmt.Errorf("could not retrieve keys for account ID %d: %w", accountID, err)
	}

	// 4. Combine and de-duplicate keys, excluding specified keys.
	type keyInfo struct {
		id      int
		line    string
		comment string
	}
	allUserKeysMap := make(map[int]keyInfo)
	excludeSet := make(map[int]bool)

	// Create exclude set for fast lookup
	for _, keyID := range excludeKeyIDs {
		excludeSet[keyID] = true
	}

	formatKey := func(key model.PublicKey) string {
		if key.Comment != "" {
			return fmt.Sprintf("%s %s %s", key.Algorithm, key.KeyData, key.Comment)
		}
		return fmt.Sprintf("%s %s", key.Algorithm, key.KeyData)
	}

	// Filter expired keys first
	filterExpired := func(keys []model.PublicKey) []model.PublicKey {
		var out []model.PublicKey
		now := time.Now().UTC()
		for _, k := range keys {
			if k.ExpiresAt.IsZero() || k.ExpiresAt.After(now) {
				out = append(out, k)
			}
		}
		return out
	}
	globalKeys = filterExpired(globalKeys)
	accountKeys = filterExpired(accountKeys)

	// Add global keys (excluding those in excludeSet)
	for _, key := range globalKeys {
		if !excludeSet[key.ID] {
			allUserKeysMap[key.ID] = keyInfo{id: key.ID, line: formatKey(key), comment: key.Comment}
		}
	}

	// Add account keys (excluding those in excludeSet)
	for _, key := range accountKeys {
		if !excludeSet[key.ID] {
			allUserKeysMap[key.ID] = keyInfo{id: key.ID, line: formatKey(key), comment: key.Comment}
		}
	}

	// Convert map to slice for sorting by comment to ensure stable output
	var sortedKeys []keyInfo
	for _, ki := range allUserKeysMap {
		sortedKeys = append(sortedKeys, ki)
	}
	sort.Slice(sortedKeys, func(i, j int) bool {
		return sortedKeys[i].comment < sortedKeys[j].comment
	})

	// Add user keys to the content
	for _, ki := range sortedKeys {
		content.WriteString("\n")
		content.WriteString(ki.line)
	}

	// Ensure file ends with newline
	if len(sortedKeys) > 0 {
		content.WriteString("\n")
	}

	return content.String(), nil
}
