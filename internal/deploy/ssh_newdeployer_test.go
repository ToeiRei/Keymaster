// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package deploy

import (
	"crypto/ed25519"
	"fmt"
	"net"
	"testing"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	genssh "github.com/toeirei/keymaster/internal/crypto/ssh"
)

// (nopReadWriteCloser removed; reuse helpers from other tests)

// reuse mockSftp declared in other deploy tests

func TestNewDeployer_PrivateKeyFailsAgentSucceeds(t *testing.T) {
	// Save and restore package-level hooks
	origDial := sshDial
	origNewSftp := newSftpClient
	origAgent := sshAgentGetter
	defer func() { sshDial = origDial; newSftpClient = origNewSftp; sshAgentGetter = origAgent }()

	// 1) Prepare a system private key that would be used but cause a dial failure
	_, privPEM, err := genssh.GenerateAndMarshalEd25519Key("test", "")
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	// 2) Prepare an in-memory agent and add an ed25519 private key to it
	keyring := agent.NewKeyring()
	_, priv, _ := ed25519.GenerateKey(nil)
	if err := keyring.Add(agent.AddedKey{PrivateKey: priv, Comment: "test"}); err != nil {
		t.Fatalf("failed to add key to agent: %v", err)
	}

	// 3) Make sshDial fail on first call (system key), succeed on second (agent)
	call := 0
	sshDial = func(network, addr string, cfg *ssh.ClientConfig) (sshClientIface, error) {
		call++
		if call == 1 {
			return nil, fmt.Errorf("simulated system key dial failure")
		}
		return &ssh.Client{}, nil
	}

	// 4) Override newSftpClient to return a mock implementation (accept nil client)
	newSftpClient = func(c sshClientIface) (sftpRaw, error) { return &mockSftp{}, nil }

	// 5) Make agent available via sshAgentGetter
	sshAgentGetter = func() agent.Agent { return keyring }

	// 6) Call NewDeployerWithConfig — system key will fail, agent should succeed
	d, err := NewDeployerWithConfig("example.com", "user", privPEM, nil, DefaultConnectionConfig(), false)
	if err != nil {
		t.Fatalf("expected success via agent fallback, got error: %v", err)
	}
	if d == nil {
		t.Fatalf("expected non-nil Deployer")
	}
	// avoid Close calling into zero-valued ssh.Client in tests
	d.client = nil
	d.Close()
}

func TestGetRemoteHostKey_Default(t *testing.T) {
	orig := sshDial
	defer func() { sshDial = orig }()
	// Generate a temporary public key and ensure HostKeyCallback receives it
	pubStr, _, err := genssh.GenerateAndMarshalEd25519Key("k", "")
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	pk, _, _, _, err := ssh.ParseAuthorizedKey([]byte(pubStr))
	if err != nil {
		t.Fatalf("parse pubkey: %v", err)
	}

	sshDial = func(network, addr string, cfg *ssh.ClientConfig) (sshClientIface, error) {
		if cfg != nil && cfg.HostKeyCallback != nil {
			_ = cfg.HostKeyCallback("example.com:22", &net.TCPAddr{}, pk)
		}
		return nil, ErrHostKeySuccessfullyRetrieved
	}

	got, err := GetRemoteHostKey("example.com")
	if err != nil {
		t.Fatalf("GetRemoteHostKey returned error: %v", err)
	}
	if string(ssh.MarshalAuthorizedKey(got)) != string(ssh.MarshalAuthorizedKey(pk)) {
		t.Fatalf("retrieved key does not match expected key")
	}
}

