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

	CreatePublicKey(ctx context.Context, identity string, tags []string) (PublicKey, error)

	GetPublicKey(ctx context.Context, id ID) (PublicKey, error)

	GetPublicKeys(ctx context.Context, ids ...ID) ([]PublicKey, error)

	ListPublicKeys(ctx context.Context, tagFilter string) ([]PublicKey, error)

	UpdatePublicKeyTags(ctx context.Context, id ID, tags []string) error

	DeletePublicKeys(ctx context.Context, ids ...ID) error

	// --- Target Management ---

	CreateTarget(ctx context.Context, host string, port int /* , gateway string, plugin string */) (Target, error)

	GetTarget(ctx context.Context, id ID) (Target, error)

	GetTargets(ctx context.Context, ids ...ID) ([]Target, error)

	ListTargets(ctx context.Context) ([]Target, error)

	UpdateTarget(ctx context.Context, id ID, target Target) error

	DeleteTargets(ctx context.Context, ids ...ID) error

	// --- Account Management ---

	CreateAccount(ctx context.Context, targetID ID, name string, deploymentKey string) (Account, error)

	GetAccount(ctx context.Context, id ID) (Account, error)

	GetAccounts(ctx context.Context, ids ...ID) ([]Account, error)

	ListAccountsByTarget(ctx context.Context, targetID ID) ([]Account, error)

	DeleteAccounts(ctx context.Context, ids ...ID) error

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
	id       ID
	identity string
	tags     []string
	// ...
}

// Target represents a remote host or endpoint to deploy keys to.
type Target struct {
	id   ID
	host string
	port int
	// ...
}

// Account represents an account on a target host.
type Account struct {
	id            ID
	targetID      ID
	name          string
	deploymentKey string
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
