package core

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/model"
	"github.com/toeirei/keymaster/internal/sshkey"
	"github.com/toeirei/keymaster/internal/state"
)

// ImportRemoteKeys connects to a host, reads its authorized_keys, imports new keys
// into the database, and returns the newly imported keys.
func ImportRemoteKeys(account model.Account) (importedKeys []model.PublicKey, skippedCount int, warning string, err error) {
	var connectKey *model.SystemKey
	var privateKey string

	if account.Serial == 0 {
		connectKey, err = db.GetActiveSystemKey()
		if err != nil {
			return nil, 0, "", fmt.Errorf("failed to get active system key for import: %w", err)
		}
		if connectKey == nil {
			warning = "Warning: No active system key. Using SSH agent."
			privateKey = ""
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

	passphrase := state.PasswordCache.Get()
	defer func() {
		for i := range passphrase {
			passphrase[i] = 0
		}
	}()

	deployer, err := NewDeployerFactory(account.Hostname, account.Username, privateKey, passphrase)
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

		km := db.DefaultKeyManager()
		if km == nil {
			skippedCount++
			continue
		}
		newKey, dbErr := km.AddPublicKeyAndGetModel(alg, keyData, comment, false, time.Time{})
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
