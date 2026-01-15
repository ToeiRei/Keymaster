//go:build windows
// +build windows

// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// package deploy provides functionality for connecting to remote hosts via SSH
// and managing their authorized_keys files. This file contains the Windows-specific
// implementation for locating the SSH agent.
package deploy // import "github.com/toeirei/keymaster/internal/deploy"

import (
	"net"
	"os"

	"github.com/Microsoft/go-winio"
	"github.com/davidmz/go-pageant"
	"golang.org/x/crypto/ssh/agent"
)

// getSSHAgent attempts to connect to a running SSH agent on Windows.
// It first tries to connect to Pageant-compatible agents (like PuTTY's). If that
// fails, it falls back to checking for the OpenSSH agent via named pipes, using
// the SSH_AUTH_SOCK environment variable or a default pipe name.
func getSSHAgent() agent.Agent {
	// 1. Try Pageant-like agents (PuTTY, gpg-agent)
	if pageant.Available() {
		return pageant.New()
	}

	// 2. Try OpenSSH agent named pipes
	var agentConn net.Conn
	var err error
	if sshAgentSocket := os.Getenv("SSH_AUTH_SOCK"); sshAgentSocket != "" {
		agentConn, err = winio.DialPipe(sshAgentSocket, nil)
	} else {
		// If no env var, try the default OpenSSH for Windows named pipe as a fallback.
		agentConn, err = winio.DialPipe(`\\.\pipe\openssh-ssh-agent`, nil)
	}

	if err == nil && agentConn != nil {
		return agent.NewClient(agentConn)
	}

	return nil
}
