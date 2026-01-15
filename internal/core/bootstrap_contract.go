// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/toeirei/keymaster/internal/bootstrap"
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
	// SessionID, if set, links this operation to an existing bootstrap session
	// so core can update or remove the persisted session record.
	SessionID string
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

// Auditor is the minimal interface core requires to emit audit log events.
// Callers should provide an implementation that records audit events in the
// appropriate environment (DB, test double, etc.).
type Auditor interface {
	LogAction(action, details string) error
}

// BootstrapDeployer is the minimal interface the core requires to deploy
// authorized_keys to a remote host. The concrete implementation lives in
// the deploy package; core depends only on this interface to remain UI-agnostic.
type BootstrapDeployer interface {
	DeployAuthorizedKeys(content string) error
	Close()
}

// AccountStore defines the minimal account-related operations core orchestration
// may request. Implementations live outside core (DB, UI, tests).
type AccountStore interface {
	AddAccount(username, hostname, label, tags string) (int, error)
	DeleteAccount(accountID int) error
}

// KeyStore defines the minimal key-related operations core orchestration
// may request. Implementations live outside core (DB, key manager, tests).
type KeyStore interface {
	GetGlobalPublicKeys() ([]model.PublicKey, error)
	GetKeysForAccount(accountID int) ([]model.PublicKey, error)
	AssignKeyToAccount(keyID, accountID int) error
}

// SystemKeyStore provides operations for creating and rotating system keys.
type SystemKeyStore interface {
	CreateSystemKey(publicKey, privateKey string) (int, error)
	RotateSystemKey(publicKey, privateKey string) (int, error)
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

	// AccountStore is an optional interface implementation that callers may
	// provide instead of function hooks. It is not required but convenient
	// for callers that already have an Account store implementation.
	AccountStore AccountStore

	// KeyStore is an optional interface implementation for key operations.
	KeyStore KeyStore

	// GenerateKeysContent produces the authorized_keys content for an account.
	GenerateKeysContent func(accountID int) (string, error)

	// NewBootstrapDeployer creates a deployer configured with an expected host key.
	NewBootstrapDeployer func(hostname, username, privateKey, expectedHostKey string) (BootstrapDeployer, error)

	// GetActiveSystemKey fetches the active system key serial and public key.
	GetActiveSystemKey func() (*model.SystemKey, error)

	// LogAudit records an audit event related to bootstrap.
	LogAudit func(e BootstrapAuditEvent) error

	// SessionStore is an optional interface implementation for persisting
	// bootstrap session state. If provided, core will update/delete session
	// records as part of lifecycle management.
	SessionStore SessionStore

	// Auditor is an optional interface implementation callers may provide
	// for simpler audit writes. If provided, core may call Auditor.LogAction
	// instead of or in addition to LogAudit.
	Auditor Auditor
}

// SessionStore defines persistence operations for bootstrap sessions that
// core orchestration may request. Implementations live outside core and
// typically delegate to the DB layer.
type SessionStore interface {
	SaveBootstrapSession(id, username, hostname, label, tags, tempPublicKey string, expiresAt time.Time, status string) error
	GetBootstrapSession(id string) (*model.BootstrapSession, error)
	DeleteBootstrapSession(id string) error
	UpdateBootstrapSessionStatus(id string, status string) error
	GetExpiredBootstrapSessions() ([]*model.BootstrapSession, error)
	GetOrphanedBootstrapSessions() ([]*model.BootstrapSession, error)
}

// PerformBootstrapDeployment orchestrates the bootstrap deployment using the
// provided params and side-effecting dependencies. This is a stub in this
// change: it returns zero values and must be implemented later.
func PerformBootstrapDeployment(ctx context.Context, params BootstrapParams, deps BootstrapDeps) (BootstrapResult, error) {
	// Validate inputs first using pure helpers. Validation is deterministic
	// and side-effect free; move additional checks here as needed.
	if err := ValidateBootstrapParams(params.Username, params.Hostname, params.Label, params.Tags); err != nil {
		return BootstrapResult{}, err
	}

	// This implementation performs the standard bootstrap commit steps using
	// the provided side-effecting dependencies. Core remains environment-
	// agnostic by calling only functions on the provided deps and interfaces.

	res := BootstrapResult{}

	// Step 1: Create account
	var accountID int
	var err error
	if deps.AccountStore != nil {
		accountID, err = deps.AccountStore.AddAccount(params.Username, params.Hostname, params.Label, params.Tags)
	} else if deps.AddAccount != nil {
		accountID, err = deps.AddAccount(params.Username, params.Hostname, params.Label, params.Tags)
	} else {
		return res, fmt.Errorf("no account creation dependency provided")
	}
	if err != nil {
		return res, fmt.Errorf("failed to create account: %w", err)
	}

	account := model.Account{ID: accountID, Username: params.Username, Hostname: params.Hostname, Label: params.Label, Tags: params.Tags, IsActive: true}

	// Ensure we cleanup on failure by deleting the account if possible.
	cleanupAccount := func() {
		if deps.DeleteAccount != nil {
			_ = deps.DeleteAccount(accountID)
		} else if deps.AccountStore != nil {
			_ = deps.AccountStore.DeleteAccount(accountID)
		}
	}

	// Step 2: Assign selected keys
	if len(params.SelectedKeyIDs) > 0 {
		for _, kid := range params.SelectedKeyIDs {
			if deps.KeyStore != nil {
				if err := deps.KeyStore.AssignKeyToAccount(kid, accountID); err != nil {
					cleanupAccount()
					return res, fmt.Errorf("failed to assign key %d: %w", kid, err)
				}
			} else if deps.AssignKey != nil {
				if err := deps.AssignKey(kid, accountID); err != nil {
					cleanupAccount()
					return res, fmt.Errorf("failed to assign key %d: %w", kid, err)
				}
			} else {
				// Not fatal: warn and continue
			}
		}
	}

	// Step 3: Build authorized_keys content
	if deps.GenerateKeysContent == nil {
		cleanupAccount()
		return res, fmt.Errorf("GenerateKeysContent dependency not provided")
	}
	content, err := deps.GenerateKeysContent(accountID)
	if err != nil {
		cleanupAccount()
		return res, fmt.Errorf("failed to generate authorized_keys content: %w", err)
	}

	// Step 4: Deploy to remote host using bootstrap deployer if provided
	deployed := false
	if deps.NewBootstrapDeployer != nil {
		d, derr := deps.NewBootstrapDeployer(params.Hostname, params.Username, params.TempPrivateKey, params.HostKey)
		if derr != nil {
			cleanupAccount()
			return res, fmt.Errorf("failed to create bootstrap deployer: %w", derr)
		}
		if d != nil {
			if err := d.DeployAuthorizedKeys(content); err != nil {
				d.Close()
				cleanupAccount()
				return res, fmt.Errorf("failed to deploy authorized_keys: %w", err)
			}
			d.Close()
			deployed = true
		}
	}

	// Step 5: Optionally record system key serial (best-effort)
	if deps.GetActiveSystemKey != nil {
		if sk, _ := deps.GetActiveSystemKey(); sk != nil {
			// We don't perform DB update here; callers may update account serial
			// via their own stores. Record in result warnings if needed.
			res.Warnings = append(res.Warnings, fmt.Sprintf("system key serial available: %d", sk.Serial))
		}
	}

	// Step 6: Audit
	auditDetails := fmt.Sprintf("bootstrap: account=%s@%s id=%d deployed=%v", params.Username, params.Hostname, accountID, deployed)
	if deps.Auditor != nil {
		_ = deps.Auditor.LogAction("BOOTSTRAP_SUCCESS", auditDetails)
	} else if deps.LogAudit != nil {
		_ = deps.LogAudit(BootstrapAuditEvent{Action: "BOOTSTRAP_SUCCESS", Details: auditDetails})
	}

	// Populate result
	res.Account = account
	res.RemoteDeployed = deployed
	res.KeysDeployed = params.SelectedKeyIDs

	// Update or remove persisted bootstrap session state if a store was provided.
	if params.HostKey != "" || params.TempPrivateKey != "" {
		// noop - params contain keys but session lifecycle is controlled by SessionStore below.
	}

	if paramsTemp := params; paramsTemp.SessionID != "" {
		if deps.SessionStore != nil {
			if deployed {
				_ = deps.SessionStore.DeleteBootstrapSession(paramsTemp.SessionID)
			} else {
				_ = deps.SessionStore.UpdateBootstrapSessionStatus(paramsTemp.SessionID, string(bootstrap.StatusFailed))
			}
		}
		bootstrap.UnregisterSession(paramsTemp.SessionID)
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

