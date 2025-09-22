//go:build !windows
// +build !windows

package deploy

import (
	"net"
	"os"

	"golang.org/x/crypto/ssh/agent"
)

// getSSHAgent attempts to connect to a running SSH agent on Unix-like systems.
// It checks the SSH_AUTH_SOCK environment variable for the socket path.
func getSSHAgent() agent.Agent {
	if sshAgentSocket := os.Getenv("SSH_AUTH_SOCK"); sshAgentSocket != "" {
		if conn, err := net.Dial("unix", sshAgentSocket); err == nil {
			return agent.NewClient(conn)
		}
	}
	return nil
}
