package core

import (
	"context"

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
	return BootstrapResult{}, nil
}
