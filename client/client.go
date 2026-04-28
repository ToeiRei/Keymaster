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

	// Close cleans up resources held by the client and closes any open connections.
	Close(ctx context.Context) error

	// --- PublicKey Management ---

	CreatePublicKey(ctx context.Context, key string, comment string, tags []string) (PublicKey, error)

	GetPublicKey(ctx context.Context, id ID) (PublicKey, error)

	GetPublicKeys(ctx context.Context, ids ...ID) ([]PublicKey, error)

	ListPublicKeys(ctx context.Context, tagFilter string) ([]PublicKey, error)

	UpdatePublicKey(ctx context.Context, id ID, comment string, tags []string) error

	DeletePublicKeys(ctx context.Context, ids ...ID) error

	// --- Account Management ---

	CreateAccount(ctx context.Context, name string, host string, port int, deploymentMethod string, deploymentSecret string) (Account, error)

	GetAccount(ctx context.Context, id ID) (Account, error)

	GetAccounts(ctx context.Context, ids ...ID) ([]Account, error)

	ListAccounts(ctx context.Context) ([]Account, error)

	UpdateAccount(ctx context.Context, id ID, name string, host string, port int, deploymentMethod string, deploymentSecret string) error

	DeleteAccounts(ctx context.Context, ids ...ID) error

	IsAccountDirty(ctx context.Context, account Account) (bool, error)

	GetDirtyAccounts(ctx context.Context) ([]Account, error)

	// --- Tag & Account-PublicKey relation Management ---

	ListExistingTags(ctx context.Context) []string

	LinkTagAccount(ctx context.Context, accountID ID, filter string, expiresAt time.Time) (ID, error)

	UnLinkTagAccount(ctx context.Context, linkIDs ...ID) error

	ResolvePublicKeyLinks(ctx context.Context, accountID ID) ([]Link, error)

	ResolveAccountLinks(ctx context.Context, publicKeyID ID) ([]Link, error)

	ResolvePublicKeysForAccount(ctx context.Context, accountID ID) ([]PublicKey, error)

	ResolveAccountsForPublicKey(ctx context.Context, publicKeyID ID) ([]Account, error)

	// --- Onboarding & Decommision ---

	OnboardHost(ctx context.Context, host string, port int /* , gateway string, plugin string */, accountName string, deploymentKey string) (chan OnboardHostProgress, error)

	DecommisionAccount(ctx context.Context, id ID) (chan DecommisionAccountProgress, error)

	// --- Deploy stuff ---

	DeployPublicKeys(ctx context.Context, publicKeyID ...ID) (chan DeployProgress, error)

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

// Account represents an account on a target host.
type Account struct {
	Id           ID
	Name         string
	Host         string
	Port         int
	DeployMethod string // ssh, cisco, ...
	DeploySecret string
	DeployCache  string
	// ...
}

type Link struct {
	Id        ID
	accountID ID
	tagFilter string
	expiresAt time.Time
	// ...
}

// DeployProgress reports incremental progress for a deployment operation.
type DeployProgress struct {
	Percent float64
	Targets map[*Account]float64
	// ...
}

// OnboardHostProgress reports progress during host onboarding.
type OnboardHostProgress struct {
	Percent float64
	// ...
}

// DecommisionTargetProgress reports progress for target decommissioning.
type DecommisionTargetProgress struct {
	Percent float64
	// ...
}

// DecommisionAccountProgress reports progress for account decommissioning.
type DecommisionAccountProgress struct {
	Percent float64
	// ...
}
