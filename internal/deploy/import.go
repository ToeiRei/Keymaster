package deploy

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"

	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/model"
	"github.com/toeirei/keymaster/internal/sshkey"
)

// ImportRemoteKeys connects to a host, reads its authorized_keys, imports new keys
// into the database, and returns the newly imported keys.
func ImportRemoteKeys(account model.Account) (importedKeys []model.PublicKey, skippedCount int, warning string, err error) {
	// 1. Get the correct system key to connect with.
	var connectKey *model.SystemKey
	var privateKey string

	if account.Serial == 0 {
		// For a new host, we try to use the active key.
		connectKey, err = db.GetActiveSystemKey()
		if err != nil {
			return nil, 0, "", fmt.Errorf("failed to get active system key for import: %w", err)
		}
		if connectKey == nil {
			// Not a fatal error. We can proceed with just the agent and issue a warning.
			warning = "Warning: No active system key. Using SSH agent."
			privateKey = "" // Explicitly empty
		} else {
			privateKey = connectKey.PrivateKey
		}
	} else {
		connectKey, err = db.GetSystemKeyBySerial(account.Serial)
		if err != nil {
			return nil, 0, "", fmt.Errorf("failed to get system key %d for import: %w", account.Serial, err)
		}
		if connectKey == nil {
			return nil, 0, "", fmt.Errorf("db inconsistency: no system key found for serial %d", account.Serial)
		}
		privateKey = connectKey.PrivateKey
	}

	// 2. Connect using the deployer.
	deployer, err := NewDeployer(account.Hostname, account.Username, privateKey)
	if err != nil {
		return nil, 0, warning, fmt.Errorf("connection failed: %w", err)
	}
	defer deployer.Close()

	// 3. Get remote content.
	content, err := deployer.GetAuthorizedKeys()
	if err != nil {
		return nil, 0, warning, fmt.Errorf("could not read remote authorized_keys: %w", err)
	}

	// 4. Parse and import.
	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		alg, keyData, comment, parseErr := sshkey.Parse(line)
		if parseErr != nil || comment == "" {
			skippedCount++
			continue
		}

		// Try to add the key. AddPublicKeyAndGetModel returns (nil, nil) for duplicates.
		newKey, dbErr := db.AddPublicKeyAndGetModel(alg, keyData, comment)
		if dbErr != nil {
			// A real DB error occurred, log it or handle it. For now, we just skip.
			skippedCount++
			continue
		}

		if newKey != nil {
			// It was a new key, add it to our list.
			importedKeys = append(importedKeys, *newKey)
		} else {
			// It was a duplicate.
			skippedCount++
		}
	}

	return importedKeys, skippedCount, warning, nil
}
