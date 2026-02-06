// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/toeirei/keymaster/internal/core/model"
	"github.com/toeirei/keymaster/internal/core/sshkey"
	"github.com/toeirei/keymaster/internal/core/state"
	"github.com/toeirei/keymaster/internal/security"
)

// ImportRemoteKeys connects to a host, reads its authorized_keys, imports new keys
// into the database, and returns the newly imported keys.
func ImportRemoteKeys(account model.Account) (importedKeys []model.PublicKey, skippedCount int, warning string, err error) {
	var connectKey *model.SystemKey
	var privateKeySecret security.Secret

	kr := DefaultKeyReader()
	if kr == nil {
		// no reader configured — proceed but warn and use SSH agent
		warning = "Warning: No active system key. Using SSH agent."
		privateKeySecret = nil
	} else {
		if account.Serial == 0 {
			connectKey, err = kr.GetActiveSystemKey()
			if err != nil {
				return nil, 0, "", fmt.Errorf("failed to get active system key for import: %w", err)
			}
			if connectKey == nil {
				warning = "Warning: No active system key. Using SSH agent."
				privateKeySecret = nil
			} else {
				privateKeySecret = SystemKeyToSecret(connectKey)
			}
		} else {
			connectKey, err = kr.GetSystemKeyBySerial(account.Serial)
			if err != nil {
				return nil, 0, "", fmt.Errorf("failed to get system key %d for import: %w", account.Serial, err)
			}
			if connectKey == nil {
				return nil, 0, "", fmt.Errorf("db inconsistency: no system key found for serial %d", account.Serial)
			}
			privateKeySecret = SystemKeyToSecret(connectKey)
		}
	}

	passphrase := state.PasswordCache.Get()
	defer func() {
		for i := range passphrase {
			passphrase[i] = 0
		}
	}()

	deployer, err := NewDeployerFactory(account.Hostname, account.Username, privateKeySecret, passphrase)
	if err != nil {
		return nil, 0, warning, fmt.Errorf("connection failed: %w", err)
	}
	defer deployer.Close()

	content, err := deployer.GetAuthorizedKeys()
	if err != nil {
		return nil, 0, warning, fmt.Errorf("could not read remote authorized_keys: %w", err)
	}

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

		ki := DefaultKeyImporter()
		if ki == nil {
			skippedCount++
			continue
		}
		newKey, dbErr := ki.AddPublicKeyAndGetModel(alg, keyData, comment, false, time.Time{})
		if dbErr != nil {
			skippedCount++
			continue
		}

		if newKey != nil {
			importedKeys = append(importedKeys, *newKey)
		} else {
			skippedCount++
		}
	}

	return importedKeys, skippedCount, warning, nil
}
