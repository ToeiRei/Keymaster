// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core

import (
	"fmt"

	"github.com/toeirei/keymaster/internal/crypto/ssh"
)

// CreateInitialSystemKey generates a new system keypair using the provided
// passphrase and persists it using the provided SystemKeyStore. Returns the
// public key string and the assigned serial on success.
func CreateInitialSystemKey(store SystemKeyStore, passphrase string) (string, int, error) {
	pub, priv, err := ssh.GenerateAndMarshalEd25519Key("keymaster-system-key", passphrase)
	if err != nil {
		return "", 0, fmt.Errorf("generate key: %w", err)
	}
	if store == nil {
		return pub, 0, nil
	}
	serial, err := store.CreateSystemKey(pub, priv)
	if err != nil {
		return "", 0, fmt.Errorf("save key: %w", err)
	}
	return pub, serial, nil
}

// RotateSystemKey generates a new system keypair using the provided
// passphrase and rotates the existing system key via the provided store,
// returning the new serial.
func RotateSystemKey(store SystemKeyStore, passphrase string) (int, error) {
	pub, priv, err := ssh.GenerateAndMarshalEd25519Key("keymaster-system-key", passphrase)
	if err != nil {
		return 0, fmt.Errorf("generate key: %w", err)
	}
	if store == nil {
		return 0, nil
	}
	serial, err := store.RotateSystemKey(pub, priv)
	if err != nil {
		return 0, fmt.Errorf("save rotated key: %w", err)
	}
	return serial, nil
}
