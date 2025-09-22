//go:build windows
// +build windows

package deploy

import (
	"net"
	"os"

	"github.com/Microsoft/go-winio"
	"github.com/davidmz/go-pageant"
	"golang.org/x/crypto/ssh/agent"
)

// getSSHAgent attempts to connect to a running SSH agent on Windows.
// It tries Pageant-compatible agents first, then falls back to OpenSSH-style named pipes.
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
