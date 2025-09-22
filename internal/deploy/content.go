package deploy

import (
	"fmt"
	"strings"

	"github.com/toeirei/keymaster/internal/db"
)

// GenerateKeysContent constructs the full authorized_keys file content
// for a given account.
func GenerateKeysContent(accountID int) (string, error) {
	var b strings.Builder

	// 1. Add the *active* Keymaster system key. This shows the ideal state.
	systemKey, err := db.GetActiveSystemKey()
	if err != nil {
		return "", fmt.Errorf("failed to get active system key: %w", err)
	}
	if systemKey == nil {
		return "", fmt.Errorf("no active system key found. Please generate one")
	}
	b.WriteString(fmt.Sprintf("# Keymaster System Key (Serial: %d)\n", systemKey.Serial))
	b.WriteString(systemKey.PublicKey)
	b.WriteString("\n\n")

	// 2. Add all global public keys
	globalKeys, err := db.GetGlobalPublicKeys()
	if err != nil {
		return "", fmt.Errorf("failed to get global keys: %w", err)
	}
	if len(globalKeys) > 0 {
		b.WriteString("# Global Public Keys\n")
		for _, key := range globalKeys {
			b.WriteString(key.String())
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// 3. Add all user-assigned public keys
	userKeys, err := db.GetKeysForAccount(accountID)
	if err != nil {
		return "", fmt.Errorf("failed to get user keys for account %d: %w", accountID, err)
	}
	if len(userKeys) > 0 {
		b.WriteString("# User-assigned Public Keys\n")
		for _, key := range userKeys {
			b.WriteString(key.String())
			b.WriteString("\n")
		}
	} else {
		b.WriteString("# No user-assigned public keys for this account.\n")
	}

	return b.String(), nil
}
