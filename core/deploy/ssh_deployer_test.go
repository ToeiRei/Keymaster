// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package deploy

import (
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"testing"

	"github.com/toeirei/keymaster/core/security"
	"golang.org/x/crypto/ssh"
)

// mockSftp is a minimal sftpRaw implementation used in tests.
type mockSftp struct{}

func (m *mockSftp) Create(path string) (io.ReadWriteCloser, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockSftp) Stat(p string) (os.FileInfo, error)        { return nil, fmt.Errorf("not implemented") }
func (m *mockSftp) Mkdir(path string) error                   { return nil }
func (m *mockSftp) Chmod(path string, mode os.FileMode) error { return nil }
func (m *mockSftp) Remove(path string) error                  { return nil }
func (m *mockSftp) Rename(oldpath, newpath string) error      { return nil }
func (m *mockSftp) Open(path string) (io.ReadWriteCloser, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockSftp) Close() error { return nil }

// TestNewDeployerWithExpectedHostKey_Success ensures that when the presented
// host key matches the expected key, a Deployer is returned successfully.
func TestNewDeployerWithExpectedHostKey_Success(t *testing.T) {
	priv, err := os.ReadFile("../../testdata/ssh_host_ed25519_key")
	if err != nil {
		t.Fatalf("failed to read private key: %v", err)
	}
	pubData, err := os.ReadFile("../../testdata/ssh_host_ed25519_key.pub")
	if err != nil {
		t.Fatalf("failed to read public key: %v", err)
	}
	pk, _, _, _, err := ssh.ParseAuthorizedKey(pubData)
	if err != nil {
		t.Fatalf("failed to parse public key: %v", err)
	}

	// Override sshDial and newSftpClient
	od := sshDial
	on := newSftpClient
	defer func() { sshDial = od; newSftpClient = on }()

	sshDial = func(network, addr string, config *ssh.ClientConfig) (sshClientIface, error) {
		if config != nil && config.HostKeyCallback != nil {
			// Simulate successful host key verification
			_ = config.HostKeyCallback("example.com:22", &net.TCPAddr{}, pk)
		}
		return &ssh.Client{}, nil
	}

	newSftpClient = func(c sshClientIface) (sftpRaw, error) {
		return &mockSftp{}, nil
	}

	d, err := NewBootstrapDeployerWithExpectedKey("example.com", "user", security.FromBytes(priv), strings.TrimSpace(string(ssh.MarshalAuthorizedKey(pk))))
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if d == nil {
		t.Fatalf("expected non-nil Deployer")
	}
	// Avoid calling Close on a zero-valued ssh.Client returned from our fake dial.
	d.client = nil
	d.Close()
}

// TestNewDeployerWithExpectedHostKey_Mismatch ensures that when the presented
// host key does not match the expected key, an error is returned.
func TestNewDeployerWithExpectedHostKey_Mismatch(t *testing.T) {
	priv, err := os.ReadFile("../../testdata/ssh_host_ed25519_key")
	if err != nil {
		t.Fatalf("failed to read private key: %v", err)
	}
	pubData, err := os.ReadFile("../../testdata/ssh_host_rsa_key.pub")
	if err != nil {
		t.Fatalf("failed to read alternate public key: %v", err)
	}
	pk, _, _, _, err := ssh.ParseAuthorizedKey(pubData)
	if err != nil {
		t.Fatalf("failed to parse public key: %v", err)
	}

	od := sshDial
	defer func() { sshDial = od }()

	sshDial = func(network, addr string, config *ssh.ClientConfig) (sshClientIface, error) {
		if config != nil && config.HostKeyCallback != nil {
			if err := config.HostKeyCallback("example.com:22", &net.TCPAddr{}, pk); err != nil {
				return nil, err
			}
		}
		return &ssh.Client{}, nil
	}

	_, err = NewBootstrapDeployerWithExpectedKey("example.com", "user", security.FromBytes(priv), "ssh-ed25519 AAAA-invalid-key")
	if err == nil {
		t.Fatalf("expected error due to host key mismatch, got nil")
	}
}
