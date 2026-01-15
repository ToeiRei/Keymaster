// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package deploy

import (
	"net"
	"os"
	"testing"

	"golang.org/x/crypto/ssh"
)

// TestGetRemoteHostKey_WithInjectedDial verifies that GetRemoteHostKeyWithTimeout
// correctly retrieves a host key when the package-level sshDial is overridden
// to simulate a handshake that returns the host key via the HostKeyCallback.
func TestGetRemoteHostKey_WithInjectedDial(t *testing.T) {
	// Load a known public key from testdata
	data, err := os.ReadFile("../../testdata/ssh_host_ed25519_key.pub")
	if err != nil {
		t.Fatalf("failed to read public key testdata: %v", err)
	}
	pk, _, _, _, err := ssh.ParseAuthorizedKey(data)
	if err != nil {
		t.Fatalf("failed to parse authorized key: %v", err)
	}

	// Override sshDial and restore afterwards
	orig := sshDial
	defer func() { sshDial = orig }()

	sshDial = func(network, addr string, config *ssh.ClientConfig) (sshClientIface, error) {
		// Simulate calling the HostKeyCallback as the real handshake would.
		if config != nil && config.HostKeyCallback != nil {
			_ = config.HostKeyCallback("example.com:22", &net.TCPAddr{}, pk)
		}
		// Simulate ssh.Dial failing with our sentinel error so GetRemoteHostKeyWithTimeout
		// detects the success path.
		return nil, ErrHostKeySuccessfullyRetrieved
	}

	got, err := GetRemoteHostKeyWithTimeout("example.com", DefaultHostKeyTimeout)
	if err != nil {
		t.Fatalf("GetRemoteHostKeyWithTimeout returned error: %v", err)
	}

	if string(ssh.MarshalAuthorizedKey(got)) != string(ssh.MarshalAuthorizedKey(pk)) {
		t.Fatalf("retrieved key does not match expected key")
	}
}
