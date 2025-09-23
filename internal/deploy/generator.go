// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// package deploy provides functionality for connecting to remote hosts via SSH
// and managing their authorized_keys files. This file contains the logic for
// generating the content of an authorized_keys file from database records.
package deploy // import "github.com/toeirei/keymaster/internal/deploy"

import (
	"fmt"
	"sort"
	"strings"

	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/model"
)

// SystemKeyRestrictions defines the SSH options applied to the Keymaster system key.
// These restrictions limit the key to only allow SFTP access for file management,
// enhancing security by preventing shell access, port forwarding, etc.
const SystemKeyRestrictions = "command=\"internal-sftp\",no-port-forwarding,no-x11-forwarding,no-agent-forwarding,no-pty"

// GenerateKeysContent constructs the authorized_keys file content for a given account.
// It combines the active system key, global user keys, and account-specific keys.
func GenerateKeysContent(accountID int) (string, error) {
	var content strings.Builder

	// 1. Get the active system key. This is always the first key.
	activeKey, err := db.GetActiveSystemKey()
	if err != nil {
		return "", fmt.Errorf("could not retrieve active system key: %w", err)
	}
	if activeKey == nil {
		return "", fmt.Errorf("no active system key found. please generate one first")
	}

	// Add the Keymaster header and the restricted system key.
	content.WriteString(fmt.Sprintf("# Keymaster Managed Keys (Serial: %d)\n", activeKey.Serial))

	// Prepend restrictions to the system key.
	restrictedSystemKey := fmt.Sprintf("%s %s", SystemKeyRestrictions, activeKey.PublicKey)
	content.WriteString(restrictedSystemKey)

	// 2. Get all global public keys.
	globalKeys, err := db.GetGlobalPublicKeys()
	if err != nil {
		return "", fmt.Errorf("could not retrieve global public keys: %w", err)
	}

	// 3. Get keys specifically assigned to this account.
	accountKeys, err := db.GetKeysForAccount(accountID)
	if err != nil {
		return "", fmt.Errorf("could not retrieve keys for account ID %d: %w", accountID, err)
	}

	// 4. Combine and de-duplicate keys.
	// Use a map to de-duplicate by key ID, and a struct to hold key parts for sorting.
	type keyInfo struct {
		id      int
		line    string
		comment string
	}
	allUserKeysMap := make(map[int]keyInfo)

	formatKey := func(key model.PublicKey) string {
		if key.Comment != "" {
			return fmt.Sprintf("%s %s %s", key.Algorithm, key.KeyData, key.Comment)
		}
		return fmt.Sprintf("%s %s", key.Algorithm, key.KeyData)
	}

	for _, key := range globalKeys {
		allUserKeysMap[key.ID] = keyInfo{id: key.ID, line: formatKey(key), comment: key.Comment}
	}
	for _, key := range accountKeys {
		allUserKeysMap[key.ID] = keyInfo{id: key.ID, line: formatKey(key), comment: key.Comment}
	}

	// Convert map to slice for sorting by comment to ensure stable output
	var sortedKeys []keyInfo
	for _, ki := range allUserKeysMap {
		sortedKeys = append(sortedKeys, ki)
	}
	sort.Slice(sortedKeys, func(i, j int) bool {
		return sortedKeys[i].comment < sortedKeys[j].comment
	})

	// 5. Add user keys to the content.
	if len(sortedKeys) > 0 {
		content.WriteString("\n\n# User Keys\n")
		var keyLines []string
		for _, key := range sortedKeys {
			keyLines = append(keyLines, key.line)
		}
		content.WriteString(strings.Join(keyLines, "\n"))
	}

	content.WriteString("\n")

	return content.String(), nil
}
