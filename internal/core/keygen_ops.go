// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core

import (
	"fmt"

	"github.com/toeirei/keymaster/internal/crypto/ssh"
	"github.com/toeirei/keymaster/internal/db"
)

// CreateInitialSystemKey generates a new system keypair using the provided
// passphrase and persists it as the initial system key. Returns the public
// key string and the assigned serial on success.
func CreateInitialSystemKey(passphrase string) (string, int, error) {
	pub, priv, err := ssh.GenerateAndMarshalEd25519Key("keymaster-system-key", passphrase)
	if err != nil {
		return "", 0, fmt.Errorf("generate key: %w", err)
	}
	serial, err := db.CreateSystemKey(pub, priv)
	if err != nil {
		return "", 0, fmt.Errorf("save key: %w", err)
	}
	return pub, serial, nil
}

// RotateSystemKey generates a new system keypair using the provided
// passphrase and rotates the existing system key, returning the new serial.
func RotateSystemKey(passphrase string) (int, error) {
	pub, priv, err := ssh.GenerateAndMarshalEd25519Key("keymaster-system-key", passphrase)
	if err != nil {
		return 0, fmt.Errorf("generate key: %w", err)
	}
	serial, err := db.RotateSystemKey(pub, priv)
	if err != nil {
		return 0, fmt.Errorf("save rotated key: %w", err)
	}
	return serial, nil
}
