//go:build !windows
// +build !windows

// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// package deploy provides functionality for connecting to remote hosts via SSH
// and managing their authorized_keys files. This file contains the Unix-specific
// implementation for locating the SSH agent.
package deploy // import "github.com/toeirei/keymaster/internal/deploy"

import (
	"net"
	"os"

	"golang.org/x/crypto/ssh/agent"
)

// getSSHAgent attempts to connect to a running SSH agent on Unix-like systems.
// It checks the SSH_AUTH_SOCK environment variable for the socket path and returns
// an agent.Agent client if a connection is successful.
func getSSHAgent() agent.Agent {
	if sshAgentSocket := os.Getenv("SSH_AUTH_SOCK"); sshAgentSocket != "" {
		if conn, err := net.Dial("unix", sshAgentSocket); err == nil {
			return agent.NewClient(conn)
		}
	}
	return nil
}
