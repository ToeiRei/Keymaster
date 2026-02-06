// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// package ssh provides cryptographic helpers for SSH key operations.
// This file contains logic for generating new SSH key pairs.
package ssh // import "github.com/toeirei/keymaster/internal/core/crypto/ssh"

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"strings"

	"golang.org/x/crypto/ssh"
)

// GenerateAndMarshalEd25519Key creates a new ed25519 key pair and returns them
// as formatted strings: the public key in authorized_keys format and the private
// key in PEM format. If a non-empty passphrase is provided, the private key will
// be encrypted with it.
func GenerateAndMarshalEd25519Key(comment string, passphrase string) (publicKeyString string, privateKeyString string, err error) {
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate ed25519 key pair: %w", err)
	}

	sshPubKey, err := ssh.NewPublicKey(pubKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to create SSH public key: %w", err)
	}
	pubKeyBytes := ssh.MarshalAuthorizedKey(sshPubKey)
	publicKeyString = fmt.Sprintf("%s %s", strings.TrimSpace(string(pubKeyBytes)), comment)

	var pemBlock *pem.Block
	if passphrase == "" {
		pemBlock, err = ssh.MarshalPrivateKey(privKey, "")
	} else {
		pemBlock, err = ssh.MarshalPrivateKeyWithPassphrase(privKey, "", []byte(passphrase))
	}

	if err != nil {
		return "", "", fmt.Errorf("failed to marshal private key: %w", err)
	}

	privateKeyString = string(pem.EncodeToMemory(pemBlock))
	return publicKeyString, privateKeyString, nil
}
