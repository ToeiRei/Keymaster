// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// package ssh provides convenience wrappers around the golang.org/x/crypto/ssh package
// for handling SSH key formatting and operations.
package ssh // import "github.com/toeirei/keymaster/internal/core/crypto/ssh"

import (
	"crypto/ed25519"
	"encoding/pem"
	"fmt"

	"golang.org/x/crypto/ssh"
)

// The following variables are re-exported from golang.org/x/crypto/ssh for convenience,
// centralizing the SSH-related utilities used throughout Keymaster.

// NewPublicKey creates a new ssh.PublicKey from a crypto.PublicKey.
var NewPublicKey = ssh.NewPublicKey

// MarshalAuthorizedKey serializes a public key to the authorized_keys wire format.
var MarshalAuthorizedKey = ssh.MarshalAuthorizedKey

// FingerprintSHA256 returns the SHA256 fingerprint of the public key.
var FingerprintSHA256 = ssh.FingerprintSHA256

// MarshalEd25519PrivateKey converts an ed25519 private key to PEM format.
// It wraps the functionality from golang.org/x/crypto/ssh to produce
// a PEM block in the modern OpenSSH private key format.
func MarshalEd25519PrivateKey(key ed25519.PrivateKey, comment string) (*pem.Block, error) {
	// The MarshalPrivateKey function handles the complex OpenSSH-specific binary format.
	// It takes a crypto.Signer, which ed25519.PrivateKey implements.
	pemBlock, err := ssh.MarshalPrivateKey(key, comment)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ed25519 private key: %w", err)
	}
	return pemBlock, nil
}
