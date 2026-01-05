// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package deploy

import (
	"fmt"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// Test that when sshDial fails with a timeout-like error while using the agent,
// newDeployerInternal returns an error that has been classified as a timeout.
func TestNewDeployer_AgentConnectionTimeoutClassified(t *testing.T) {
	origDial := sshDial
	origAgent := sshAgentGetter
	defer func() { sshDial = origDial; sshAgentGetter = origAgent }()

	// Provide a non-nil agent so code attempts agent path
	sshAgentGetter = func() agent.Agent { return agent.NewKeyring() }

	// Simulate a dial error that indicates a timeout
	sshDial = func(network, addr string, cfg *ssh.ClientConfig) (sshClientIface, error) {
		return nil, fmt.Errorf("i/o timeout")
	}

	_, err := NewDeployerWithConfig("example.com", "user", "", nil, DefaultConnectionConfig(), false)
	if err == nil {
		t.Fatalf("expected error but got nil")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expected classified timeout error, got: %v", err)
	}
}

// Test that when sshDial fails with an authentication-like error while using the agent,
// newDeployerInternal returns an error that has been classified as authentication failure.
func TestNewDeployer_AgentAuthClassified(t *testing.T) {
	origDial := sshDial
	origAgent := sshAgentGetter
	defer func() { sshDial = origDial; sshAgentGetter = origAgent }()

	sshAgentGetter = func() agent.Agent { return agent.NewKeyring() }

	sshDial = func(network, addr string, cfg *ssh.ClientConfig) (sshClientIface, error) {
		return nil, fmt.Errorf("permission denied")
	}

	_, err := NewDeployerWithConfig("example.com", "user", "", nil, DefaultConnectionConfig(), false)
	if err == nil {
		t.Fatalf("expected error but got nil")
	}
	if !strings.Contains(err.Error(), "authentication failed") {
		t.Fatalf("expected classified authentication error, got: %v", err)
	}
}
