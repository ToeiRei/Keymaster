// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core

import (
	"context"
	"fmt"
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
	// Warnings contains non-fatal notes produced during orchestration.
	Warnings []string
	// Errors contains non-fatal errors encountered while executing placeholder steps.
	Errors []string
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
	// This function provides a deterministic orchestration skeleton for
	// bootstrap deployments. It intentionally does not call side-effecting
	// dependencies in this slice — instead it records placeholders so callers
	// can wire the real calls later. If a dependency is nil, the placeholder
	// will be recorded as an error.

	res := BootstrapResult{}
	var warnings []string
	var errors []string

	// Step 1: Create account in database (placeholder)
	warnings = append(warnings, "TODO: create account in DB (AddAccount)")
	if deps.AddAccount == nil {
		errors = append(errors, "AddAccount dependency not provided; account not created")
	} else {
		// Do not call deps.AddAccount here — placeholder only.
		warnings = append(warnings, "AddAccount available but not invoked in core slice (deferred)")
	}

	// Step 2: Assign selected keys to the account (placeholder)
	if len(params.SelectedKeyIDs) > 0 {
		warnings = append(warnings, fmt.Sprintf("TODO: assign %d selected keys to account (AssignKey)", len(params.SelectedKeyIDs)))
		if deps.AssignKey == nil {
			errors = append(errors, "AssignKey dependency not provided; keys not assigned")
		}
	}

	// Step 3: Generate the authorized_keys content for the account (placeholder)
	warnings = append(warnings, "TODO: generate authorized_keys content (GenerateKeysContent)")
	if deps.GenerateKeysContent == nil {
		errors = append(errors, "GenerateKeysContent dependency not provided; cannot build keys content")
	}

	// Step 4: Deploy the authorized_keys to the remote host using a bootstrap deployer (placeholder)
	warnings = append(warnings, "TODO: deploy authorized_keys to remote host (NewBootstrapDeployer + DeployAuthorizedKeys)")
	if deps.NewBootstrapDeployer == nil {
		errors = append(errors, "NewBootstrapDeployer dependency not provided; cannot deploy")
	}

	// Step 5: Update the account with the current system key serial (placeholder)
	warnings = append(warnings, "TODO: update account with system key serial (DB update)")
	if deps.GetActiveSystemKey == nil {
		warnings = append(warnings, "GetActiveSystemKey not provided; skipping system key lookup")
	} else {
		// We will not call GetActiveSystemKey in this slice to avoid side-effects.
		warnings = append(warnings, "GetActiveSystemKey available but not invoked in core slice (deferred)")
	}

	// Step 6: Log successful bootstrap (audit) (placeholder)
	warnings = append(warnings, "TODO: emit audit event for bootstrap (LogAudit)")
	if deps.LogAudit == nil {
		warnings = append(warnings, "LogAudit dependency not provided; audit will not be logged")
	}

	// Step 7: Cleanup bootstrap session (placeholder)
	warnings = append(warnings, "TODO: cleanup bootstrap session resources")

	// Populate result metadata. Note: Account and KeysDeployed are zeroed as
	// the real side-effects have been deferred to the deploy/db packages.
	res.Warnings = warnings
	res.Errors = errors

	if len(errors) > 0 {
		return res, fmt.Errorf("bootstrap orchestration incomplete: %v", errors)
	}
	return res, nil
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
