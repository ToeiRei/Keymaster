// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package client

import (
	"context"
	"time"
)

type Client interface {
	// --- Lifecycle & Initialization ---

	// Close cleans up resources held by the client and closes any open
	// connections. Calls should pass a context for cancellation/timeouts.
	Close(ctx context.Context) error

	// --- PublicKey Management ---

	CreatePublicKey(ctx context.Context, key string, comment *string, tags []string) (PublicKey, error)

	GetPublicKey(ctx context.Context, id ID) (PublicKey, error)

	GetPublicKeys(ctx context.Context, ids ...ID) ([]PublicKey, error)

	ListPublicKeys(ctx context.Context, tagFilter string) ([]PublicKey, error)

	UpdatePublicKey(ctx context.Context, id ID, comment string, tags []string) error

	// DEPRECATED
	UpdatePublicKeyTags(ctx context.Context, id ID, tags []string) error

	DeletePublicKeys(ctx context.Context, ids ...ID) error

	// --- Target Management ---

	CreateTarget(ctx context.Context, host string, port int /* , gateway string, plugin string */) (Target, error)

	GetTarget(ctx context.Context, id ID) (Target, error)

	GetTargets(ctx context.Context, ids ...ID) ([]Target, error)

	ListTargets(ctx context.Context) ([]Target, error)

	UpdateTarget(ctx context.Context, id ID, host string, port int) error

	DeleteTargets(ctx context.Context, ids ...ID) error

	// --- Account Management ---

	CreateAccount(ctx context.Context, targetID ID, name string, deploymentKey string) (Account, error)

	GetAccount(ctx context.Context, id ID) (Account, error)

	GetAccounts(ctx context.Context, ids ...ID) ([]Account, error)

	ListAccountsByTarget(ctx context.Context, targetID ID) ([]Account, error)

	DeleteAccounts(ctx context.Context, ids ...ID) error

	IsAccountDirty(ctx context.Context, account Account) (bool, error)

	GetDirtyAccounts(ctx context.Context) ([]Account, error)

	// --- Tag to Account Management ---

	LinkTagAccount(ctx context.Context, accountID ID, filter string, expiresAt time.Time) (ID, error)

	UnLinkTagAccount(ctx context.Context, linkIDs ...ID) error

	ResolvePublicKeysForAccount(ctx context.Context, accountID ID) ([]PublicKey, error)

	ResolveAccountsForPublicKey(ctx context.Context, publicKeyID ID) ([]Account, error)

	// --- Onboarding & Decommision ---

	OnboardHost(ctx context.Context, host string, port int /* , gateway string, plugin string */, accountName string, deploymentKey string) (chan OnboardHostProgress, error)

	DecommisionTarget(ctx context.Context, id ID) (chan DecommisionTargetProgress, error)

	DecommisionAccount(ctx context.Context, id ID) (chan DecommisionAccountProgress, error)

	// --- Deploy stuff ---

	DeployPublicKeys(ctx context.Context, publicKeyID ...ID) (chan DeployProgress, error)

	DeployTargets(ctx context.Context, targetID ...ID) (chan DeployProgress, error)

	DeployAccounts(ctx context.Context, accountID ...ID) (chan DeployProgress, error)

	DeployAll(ctx context.Context) (chan DeployProgress, error)
}

// ID is a local identifier type used by the client API.
type ID int

// PublicKey represents a public key record.
type PublicKey struct {
	Id        ID
	Algorithm string
	Data      string
	Comment   string
	Tags      []string
	// ...
}

// Target represents a remote host or endpoint to deploy keys to.
type Target struct {
	Id   ID
	Host string
	Port int
	// PluginType   string // shh,cmd,cisco-switch
	// PluginConfig string // freetext plugin configuration (parsed and used by plugin, not core)
	// PluginData   string // data/cache seved by plugin
	// ...
}

// Account represents an account on a target host.
type Account struct {
	Id                               ID
	TargetID                         ID
	Name                             string
	DeploymentKey                    string
	DeploymentLastAuthorizedKeysHash *string
	// PluginConfig string // freetext plugin configuration OVERWRITE (parsed and used by plugin, not core)
	// PluginData   string // data/cache seved by plugin
	// ...
}

// DeployProgress reports incremental progress for a deployment operation.
type DeployProgress struct {
	Percent float32
	Targets map[Target]float32
	// ...
}

// OnboardHostProgress reports progress during host onboarding.
type OnboardHostProgress struct {
	Percent float32
	// ...
}

// DecommisionTargetProgress reports progress for target decommissioning.
type DecommisionTargetProgress struct {
	Percent float32
	// ...
}

// DecommisionAccountProgress reports progress for account decommissioning.
type DecommisionAccountProgress struct {
	Percent float32
	// ...
}
