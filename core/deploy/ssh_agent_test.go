// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package deploy

import (
	"strings"
	"testing"

	"golang.org/x/crypto/ssh/agent"

	genssh "github.com/toeirei/keymaster/core/crypto/ssh"
	"github.com/toeirei/keymaster/core/security"
)

func TestNewDeployer_NoAgentAvailable(t *testing.T) {
	orig := sshAgentGetter
	defer func() { sshAgentGetter = orig }()

	// Simulate no ssh agent present
	sshAgentGetter = func() agent.Agent { return nil }

	_, err := NewDeployerWithConfig("example.com", "user", security.FromString(""), nil, DefaultConnectionConfig(), false)
	if err == nil || !strings.Contains(err.Error(), "no authentication method available") {
		t.Fatalf("expected no authentication method error, got: %v", err)
	}
}

func TestNewDeployer_EncryptedPrivateKeyRequiresPassphrase(t *testing.T) {
	// Generate an encrypted private key (PEM) via internal helper
	_, priv, err := genssh.GenerateAndMarshalEd25519Key("test", "passphrase")
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	_, err = NewDeployerWithConfig("example.com", "user", security.FromString(priv), nil, DefaultConnectionConfig(), false)
	if err != ErrPassphraseRequired {
		t.Fatalf("expected ErrPassphraseRequired, got: %v", err)
	}
}
