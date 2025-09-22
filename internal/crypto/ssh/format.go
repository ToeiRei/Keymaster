package ssh

import (
	"crypto/ed25519"
	"encoding/pem"
	"fmt"

	"golang.org/x/crypto/ssh"
)

// Re-export for convenience from golang.org/x/crypto/ssh
var NewPublicKey = ssh.NewPublicKey
var MarshalAuthorizedKey = ssh.MarshalAuthorizedKey

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
