// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package client

import (
	"context"
	"fmt"
	"time"

	"github.com/toeirei/keymaster/tags"
	"github.com/toeirei/keymaster/util/slicest"
)

type Client interface {
	// --- Lifecycle & Initialization ---

	Close(ctx context.Context) error

	WithTransaction(ctx context.Context, fn func(ctx context.Context, c Client) error) error

	// --- PublicKey Management ---

	CreatePublicKey(ctx context.Context, key string, comment string, tags tags.Tags) (PublicKey, error)

	GetPublicKey(ctx context.Context, id PublicKeyId) (PublicKey, error)

	GetPublicKeys(ctx context.Context, ids ...PublicKeyId) ([]PublicKey, error)

	ListPublicKeys(ctx context.Context, tagMatcher string) ([]PublicKey, error)
	ListPublicKeysLinkedToAccount(ctx context.Context, accountId AccountId, expired bool) ([]PublicKey, error)

	UpdatePublicKey(ctx context.Context, id PublicKeyId, comment string, tags tags.Tags) (PublicKey, error)

	DeletePublicKeys(ctx context.Context, ids ...PublicKeyId) error

	// --- Account Management ---

	CreateAccount(ctx context.Context, username string, host string, port int, deploymentMethod string, deploymentSecret string) (Account, error)

	GetAccount(ctx context.Context, id AccountId) (Account, error)

	GetAccounts(ctx context.Context, ids ...AccountId) ([]Account, error)

	ListAccounts(ctx context.Context) ([]Account, error)
	ListAccountsDirty(ctx context.Context) ([]Account, error)
	ListAccountsLinkedToPublicKey(ctx context.Context, publicKeyId PublicKeyId, expired bool) ([]Account, error)

	UpdateAccount(ctx context.Context, id AccountId, username string, host string, port int, deploymentMethod string, deploymentSecret string) (Account, error)

	DeleteAccounts(ctx context.Context, ids ...AccountId) error

	IsAccountDirty(ctx context.Context, account Account) (bool, error)

	// --- Link Management ---

	CreateLink(ctx context.Context, accountId AccountId, tagMatcher string, expiresAt time.Time) (Link, error)

	GetLink(ctx context.Context, id LinkId) (Link, error)

	GetLinks(ctx context.Context, ids ...LinkId) ([]Link, error)

	ListLinksForAccount(ctx context.Context, accountId AccountId, expired bool) ([]Link, error)
	ListLinksForPublicKey(ctx context.Context, publicKeyId PublicKeyId, expired bool) ([]Link, error)

	UpdateLink(ctx context.Context, id LinkId, accountId AccountId, tagMatcher string, expiresAt time.Time) (Link, error)

	DeleteLinks(ctx context.Context, ids ...LinkId) error

	// --- Deploy & Verify ---

	DeployAccount(ctx context.Context, accountId AccountId) (chan DeployProgressAccount, error)

	DeployAccounts(ctx context.Context, accountIds ...AccountId) (chan DeployProgressAccounts, error)

	VerifyAccount(ctx context.Context, accountId AccountId) (chan VerifyProgressAccount, error)

	VerifyAccounts(ctx context.Context, accountIds ...AccountId) (chan VerifyProgressAccounts, error)

	// --- Other ---

	ListAuditLogs(ctx context.Context, limit int) ([]AuditLog, error) // TODO doesn't account for filtering and pagination

	ListExistingTags(ctx context.Context) tags.Tags

	OnboardHost(ctx context.Context, host string, port int /* , gateway string, plugin string */, accountUsername string, deploymentKey string) (chan OnboardHostProgress, error)

	DecommisionAccount(ctx context.Context, id AccountId) (chan DecommisionAccountProgress, error)
}

// id is a local identifier type used by the client API.
type id = int

// PublicKey represents a public key record.
type PublicKeyId id
type PublicKey struct {
	Id        PublicKeyId
	Algorithm string
	Data      string
	Comment   string
	Tags      tags.Tags
	// ...
}

// Account represents an account on a target host.
type AccountId id
type Account struct {
	Id           AccountId
	Username     string
	Host         string
	Port         int
	DeployMethod string // ssh, cisco, ...
	DeploySecret string
	DeployCache  string
	// ...
}

func (a Account) String() string {
	return fmt.Sprintf("%s %s@%s:%d", a.DeployMethod, a.Username, a.Host, a.Port)
}

type LinkId id
type Link struct {
	Id         LinkId
	AccountId  AccountId
	TagMatcher string
	ExpiresAt  time.Time
	// ...
}

type AuditLogId id
type AuditLog struct {
	Id        AuditLogId
	Timestamp time.Time

	Metadata AuditLogMetadata

	Action  string
	Details string
}
type AuditLogMetadata struct {
	Hostname string
	Hostuser string
	Referer  string
}

func (a AuditLogMetadata) String() string {
	return fmt.Sprintf("%s %s@%s", a.Referer, a.Hostuser, a.Hostname)
}

type DeployProgressAccount struct {
	Progress float64
	Status   string
	Err      error
}

type DeployProgressAccounts struct {
	Accounts map[AccountId]*DeployProgressAccount
}

func (dp DeployProgressAccounts) Progress() float64 {
	return slicest.Reduce(
		slicest.MapValues(dp.Accounts),
		func(dap *DeployProgressAccount, total float64) float64 { return total + dap.Progress },
	) / float64(len(dp.Accounts))
}

type VerifyProgressAccount = DeployProgressAccount
type VerifyProgressAccounts = DeployProgressAccounts

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
