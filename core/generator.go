// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package core

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/toeirei/keymaster/core/keys"
	"github.com/toeirei/keymaster/core/model"
)

// SystemKeyRestrictions defines the SSH options applied to the Keymaster system key.
const SystemKeyRestrictions = `command="internal-sftp",no-port-forwarding,no-x11-forwarding,no-agent-forwarding,no-pty`

// GenerateKeysContent constructs the authorized_keys file content for a given account.
func GenerateKeysContent(accountID int) (string, error) {
	kr := DefaultKeyReader()
	if kr == nil {
		return "", fmt.Errorf("no KeyReader available")
	}
	activeKey, err := kr.GetActiveSystemKey()
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
	kr := DefaultKeyReader()
	if kr == nil {
		return "", fmt.Errorf("no KeyReader available")
	}
	systemKey, err := kr.GetSystemKeyBySerial(serial)
	if err != nil {
		return "", fmt.Errorf("could not retrieve active system key: %w", err)
	}
	if systemKey == nil {
		return "", fmt.Errorf("no active system key found. please generate one first")
	}

	kl := DefaultKeyLister()
	if kl == nil {
		return "", fmt.Errorf("no key lister available")
	}
	globalKeys, err := kl.GetGlobalPublicKeys()
	if err != nil {
		return "", fmt.Errorf("could not retrieve global public keys: %w", err)
	}
	accountKeys, err := kl.GetKeysForAccount(accountID)
	if err != nil {
		return "", fmt.Errorf("could not retrieve keys for account ID %d: %w", accountID, err)
	}

	return keys.BuildAuthorizedKeysContent(systemKey, globalKeys, accountKeys)
}

// GenerateSelectiveKeysContent constructs authorized_keys content excluding specific keys.
func GenerateSelectiveKeysContent(accountID int, serial int, excludeKeyIDs []int, removeSystemKey bool) (string, error) {
	var content strings.Builder

	if !removeSystemKey {
		if serial == 0 {
			kr := DefaultKeyReader()
			if kr == nil {
				return "", fmt.Errorf("could not retrieve active system key: no KeyReader")
			}
			activeKey, err := kr.GetActiveSystemKey()
			if err != nil {
				return "", fmt.Errorf("could not retrieve active system key: %w", err)
			}
			if activeKey == nil {
				return "", fmt.Errorf("no active system key found. please generate one first")
			}
			serial = activeKey.Serial
		}

		kr := DefaultKeyReader()
		if kr == nil {
			return "", fmt.Errorf("could not retrieve system key: no KeyReader")
		}
		systemKey, err := kr.GetSystemKeyBySerial(serial)
		if err != nil {
			return "", fmt.Errorf("could not retrieve system key: %w", err)
		}
		if systemKey == nil {
			return "", fmt.Errorf("no system key found for serial %d", serial)
		}

		content.WriteString(fmt.Sprintf("# Keymaster Managed Keys (Serial: %d)\n", systemKey.Serial))
		restrictedSystemKey := fmt.Sprintf("%s %s", SystemKeyRestrictions, systemKey.PublicKey)
		content.WriteString(restrictedSystemKey)
	}

	kl := DefaultKeyLister()
	if kl == nil {
		return "", fmt.Errorf("no key lister available")
	}
	globalKeys, err := kl.GetGlobalPublicKeys()
	if err != nil {
		return "", fmt.Errorf("could not retrieve global public keys: %w", err)
	}

	accountKeys, err := kl.GetKeysForAccount(accountID)
	if err != nil {
		return "", fmt.Errorf("could not retrieve keys for account ID %d: %w", accountID, err)
	}

	type keyInfo struct {
		id      int
		line    string
		comment string
	}
	allUserKeysMap := make(map[int]keyInfo)
	excludeSet := make(map[int]bool)
	for _, keyID := range excludeKeyIDs {
		excludeSet[keyID] = true
	}

	formatKey := func(key model.PublicKey) string {
		if key.Comment != "" {
			return fmt.Sprintf("%s %s %s", key.Algorithm, key.KeyData, key.Comment)
		}
		return fmt.Sprintf("%s %s", key.Algorithm, key.KeyData)
	}

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

	for _, key := range globalKeys {
		if !excludeSet[key.ID] {
			allUserKeysMap[key.ID] = keyInfo{id: key.ID, line: formatKey(key), comment: key.Comment}
		}
	}
	for _, key := range accountKeys {
		if !excludeSet[key.ID] {
			allUserKeysMap[key.ID] = keyInfo{id: key.ID, line: formatKey(key), comment: key.Comment}
		}
	}

	var sortedKeys []keyInfo
	for _, ki := range allUserKeysMap {
		sortedKeys = append(sortedKeys, ki)
	}
	sort.Slice(sortedKeys, func(i, j int) bool { return sortedKeys[i].comment < sortedKeys[j].comment })

	for _, ki := range sortedKeys {
		content.WriteString("\n")
		content.WriteString(ki.line)
	}
	if len(sortedKeys) > 0 {
		content.WriteString("\n")
	}
	return content.String(), nil
}
