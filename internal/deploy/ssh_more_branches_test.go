package deploy

import (
	"fmt"
	"net"
	"os"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	genssh "github.com/toeirei/keymaster/internal/crypto/ssh"
)

// Success path when a valid unencrypted private key is provided.
func TestNewDeployer_PrivateKeySuccess(t *testing.T) {
	origDial := sshDial
	origNewSftp := newSftpClient
	defer func() { sshDial = origDial; newSftpClient = origNewSftp }()

	// Generate an unencrypted private key
	_, priv, err := genssh.GenerateAndMarshalEd25519Key("test", "")
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	sshDial = func(network, addr string, cfg *ssh.ClientConfig) (sshClientIface, error) {
		return &ssh.Client{}, nil
	}
	newSftpClient = func(c sshClientIface) (sftpRaw, error) { return &mockSftp{}, nil }

	d, err := NewDeployerWithConfig("example.com", "user", priv, nil, DefaultConnectionConfig(), false)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if d == nil {
		t.Fatalf("expected non-nil Deployer")
	}
	d.client = nil
	d.Close()
}

// When sftp client creation fails after SSH connect, expect descriptive error.
func TestNewDeployer_SftpCreationFails(t *testing.T) {
	origDial := sshDial
	origNewSftp := newSftpClient
	defer func() { sshDial = origDial; newSftpClient = origNewSftp }()

	_, priv, err := genssh.GenerateAndMarshalEd25519Key("test", "")
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	sshDial = func(network, addr string, cfg *ssh.ClientConfig) (sshClientIface, error) {
		return &ssh.Client{}, nil
	}
	newSftpClient = func(c sshClientIface) (sftpRaw, error) { return nil, fmt.Errorf("sftp init failed") }

	_, err = NewDeployerWithConfig("example.com", "user", priv, nil, DefaultConnectionConfig(), false)
	if err == nil || !strings.Contains(err.Error(), "failed to create sftp client") {
		t.Fatalf("expected sftp creation error, got: %v", err)
	}
}

// Host key mismatch returned by HostKeyCallback should be classified as a host key error.
func TestNewDeployer_HostKeyMismatchClassified(t *testing.T) {
	origDial := sshDial
	defer func() { sshDial = origDial }()

	// Provide a non-nil agent to reach the agent connection attempt (or private key path)
	sshAgentGetter = func() agent.Agent { return agent.NewKeyring() }

	sshDial = func(network, addr string, cfg *ssh.ClientConfig) (sshClientIface, error) {
		if cfg != nil && cfg.HostKeyCallback != nil {
			// Load a valid test host key and present it to the callback so that
			// the marshal operation inside the callback does not panic.
			keyBytes, rerr := os.ReadFile("testdata/ssh_host_ed25519_key.pub")
			if rerr == nil {
				if pk, _, _, _, perr := ssh.ParseAuthorizedKey(keyBytes); perr == nil {
					_ = cfg.HostKeyCallback("example.com:22", &net.TCPAddr{}, pk)
				}
			}
		}
		return nil, fmt.Errorf("!!! HOST KEY MISMATCH FOR example.com !!!")
	}

	_, err := NewDeployerWithConfig("example.com", "user", "", nil, DefaultConnectionConfig(), false)
	if err == nil {
		t.Fatalf("expected error but got nil")
	}
	if !strings.Contains(err.Error(), "host key verification failed") {
		t.Fatalf("expected classified host key error, got: %v", err)
	}
}
