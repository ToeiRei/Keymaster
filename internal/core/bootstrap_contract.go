// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core

import (
	"context"
	"strings"

	"github.com/toeirei/keymaster/internal/model"
)

// BootstrapParams contains the information required to perform a bootstrap
// deployment. It mirrors the data gathered in the TUI bootstrap workflow
// but contains no logic.
type BootstrapParams struct {
	Username       string
	Hostname       string
	Label          string
	Tags           string
	SelectedKeyIDs []int
	// TempPrivateKey is the PEM-encoded private key used for the bootstrap
	// SSH connection. May be empty in some flows.
	TempPrivateKey string
	// HostKey is the expected host key (authorized_keys format) used for
	// host key verification during bootstrap deployment.
	HostKey string
}

// BootstrapResult contains the outcome of a bootstrap deployment.
// It is intentionally a simple data bag so UIs can present results.
type BootstrapResult struct {
	Account model.Account
	// KeysDeployed contains the IDs of keys that were assigned/deployed.
	KeysDeployed []int
	// RemoteDeployed indicates whether remote deployment to the host succeeded.
	RemoteDeployed bool
}

// BootstrapAuditEvent represents an audit-log event emitted by bootstrap
// orchestration. UIs or callers can translate this into DB audit entries.
type BootstrapAuditEvent struct {
	Action  string
	Details string
}

// BootstrapDeployer is the minimal interface the core requires to deploy
// authorized_keys to a remote host. The concrete implementation lives in
// the deploy package; core depends only on this interface to remain UI-agnostic.
type BootstrapDeployer interface {
	DeployAuthorizedKeys(content string) error
	Close() error
}

// BootstrapDeps lists side-effecting functions that core orchestration will
// call. Callers (UIs or higher-level services) must provide implementations
// appropriate to the environment (real DB, test doubles, etc.).
type BootstrapDeps struct {
	// AddAccount creates an account and returns its ID.
	AddAccount func(username, hostname, label, tags string) (int, error)

	// DeleteAccount removes an account by ID. Called on failure cleanup.
	DeleteAccount func(accountID int) error

	// AssignKey assigns a key to an account.
	AssignKey func(keyID, accountID int) error

	// GenerateKeysContent produces the authorized_keys content for an account.
	GenerateKeysContent func(accountID int) (string, error)

	// NewBootstrapDeployer creates a deployer configured with an expected host key.
	NewBootstrapDeployer func(hostname, username, privateKey, expectedHostKey string) (BootstrapDeployer, error)

	// GetActiveSystemKey fetches the active system key serial and public key.
	GetActiveSystemKey func() (*model.SystemKey, error)

	// LogAudit records an audit event related to bootstrap.
	LogAudit func(e BootstrapAuditEvent) error
}

// PerformBootstrapDeployment orchestrates the bootstrap deployment using the
// provided params and side-effecting dependencies. This is a stub in this
// change: it returns zero values and must be implemented later.
func PerformBootstrapDeployment(ctx context.Context, params BootstrapParams, deps BootstrapDeps) (BootstrapResult, error) {
	// High-level orchestration steps (copied from TUI executeDeployment):
	// 1. Create account in database
	// 2. Assign selected keys to the account (including global keys)
	// 3. Generate the authorized_keys content for the account
	// 4. Deploy the authorized_keys to the remote host using a bootstrap deployer
	// 5. Update the account with the current system key serial
	// 6. Log successful bootstrap (audit)
	// 7. Cleanup bootstrap session

	// Implementation note: this function is the core orchestration entrypoint.
	// It should call into the provided deps for side effects (DB, deployer,
	// audit). For this change we only add the orchestration plan and expose a
	// small, pure helper below. The full implementation will be added later.

	return BootstrapResult{}, nil
}

// FilterKeysForBootstrap separates all keys into user-selectable and global
// keys for bootstrap UI flows. It is a pure function and does not perform any
// DB or network operations. `systemKeyData` may be empty to indicate no
// active system key is available.
func FilterKeysForBootstrap(allKeys []model.PublicKey, systemKeyData string) (userSelectable []model.PublicKey, global []model.PublicKey) {
	for _, key := range allKeys {
		// Skip if this is a system key by comparing key data when available
		if systemKeyData != "" && strings.Contains(key.KeyData, systemKeyData) {
			continue
		}
		// Skip if this looks like a system key comment
		if strings.Contains(key.Comment, "Keymaster System Key") {
			continue
		}

		if key.IsGlobal {
			global = append(global, key)
		} else {
			userSelectable = append(userSelectable, key)
		}
	}
	return
}

// Pure helpers for authorized_keys generation live in internal/keys to avoid
// import cycles with the deploy package. See internal/keys for builders.
