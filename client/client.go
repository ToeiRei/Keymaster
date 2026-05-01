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

	Close(ctx context.Context) error

	// --- PublicKey Management ---

	CreatePublicKey(ctx context.Context, key string, comment string, tags []string) (PublicKey, error)

	GetPublicKey(ctx context.Context, id PublicKeyId) (PublicKey, error)

	GetPublicKeys(ctx context.Context, ids ...PublicKeyId) ([]PublicKey, error)

	ListPublicKeys(ctx context.Context, tagFilter string) ([]PublicKey, error)

	UpdatePublicKey(ctx context.Context, id PublicKeyId, comment string, tags []string) error

	DeletePublicKeys(ctx context.Context, ids ...PublicKeyId) error

	// --- Account Management ---

	CreateAccount(ctx context.Context, name string, host string, port int, deploymentMethod string, deploymentSecret string) (Account, error)

	GetAccount(ctx context.Context, id AccountId) (Account, error)

	GetAccounts(ctx context.Context, ids ...AccountId) ([]Account, error)

	ListAccounts(ctx context.Context) ([]Account, error)

	ListDirtyAccounts(ctx context.Context) ([]Account, error)

	UpdateAccount(ctx context.Context, id AccountId, name string, host string, port int, deploymentMethod string, deploymentSecret string) error

	DeleteAccounts(ctx context.Context, ids ...AccountId) error

	IsAccountDirty(ctx context.Context, account Account) (bool, error)

	// --- Link Management ---

	CreateLink(ctx context.Context, accountID AccountId, tagFilter string, expiresAt time.Time) (Link, error)

	GetLink(ctx context.Context, id LinkId) (Link, error)

	GetLinks(ctx context.Context, ids ...LinkId) ([]Link, error)

	ListLinksAccount(ctx context.Context, accountID AccountId) ([]Link, error)

	ListLinksPublicKey(ctx context.Context, publicKeyID PublicKeyId) ([]Link, error)

	ListPublicKeysForAccount(ctx context.Context, accountID AccountId) ([]PublicKey, error)

	ListAccountsForPublicKey(ctx context.Context, publicKeyID PublicKeyId) ([]Account, error)

	UpdateLink(ctx context.Context, id LinkId, accountId AccountId, tagFilter string, expiresAt time.Time) error

	DeleteLinks(ctx context.Context, ids ...LinkId) error

	// --- Other ---

	ListExistingTags(ctx context.Context) []string

	OnboardHost(ctx context.Context, host string, port int /* , gateway string, plugin string */, accountName string, deploymentKey string) (chan OnboardHostProgress, error)

	DecommisionAccount(ctx context.Context, id AccountId) (chan DecommisionAccountProgress, error)

	DeployPublicKeys(ctx context.Context, publicKeyID ...PublicKeyId) (chan DeployProgress, error)

	DeployAccounts(ctx context.Context, accountID ...AccountId) (chan DeployProgress, error)

	DeployAll(ctx context.Context) (chan DeployProgress, error)
}

// ID is a local identifier type used by the client API.
type id = int

// PublicKey represents a public key record.
type PublicKeyId id
type PublicKey struct {
	Id        PublicKeyId
	Algorithm string
	Data      string
	Comment   string
	Tags      []string
	// ...
}

// Account represents an account on a target host.
type AccountId id
type Account struct {
	Id           AccountId
	Name         string
	Host         string
	Port         int
	DeployMethod string // ssh, cisco, ...
	DeploySecret string
	DeployCache  string
	// ...
}

type LinkId id
type Link struct {
	Id        LinkId
	AccountId AccountId
	TagFilter string
	ExpiresAt time.Time
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
