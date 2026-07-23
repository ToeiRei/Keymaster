// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package client

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/toeirei/keymaster/connector"
)

type Client interface {
	// --- Lifecycle & Initialization ---

	Close(ctx context.Context) error

	WithTransaction(ctx context.Context, fn func(ctx context.Context, c Client) error) error

	// --- PublicKey Management ---

	CreatePublicKey(ctx context.Context, key string, comment string, isGlobal bool, expiresAt time.Time) (PublicKey, error)

	GetPublicKey(ctx context.Context, id PublicKeyId) (PublicKey, error)

	GetPublicKeys(ctx context.Context, ids ...PublicKeyId) ([]PublicKey, error)

	ListPublicKeys(ctx context.Context) ([]PublicKey, error)
	ListPublicKeysLinkedToAccount(ctx context.Context, accountId AccountId, expired bool) ([]PublicKey, error)

	UpdatePublicKey(ctx context.Context, id PublicKeyId, comment string, isGlobal bool, expiresAt time.Time) (PublicKey, error)

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

	CreateLink(ctx context.Context, accountId AccountId, publicKeyId PublicKeyId, expiresAt time.Time) (Link, error)

	GetLink(ctx context.Context, accountId AccountId, publicKeyId PublicKeyId) (Link, error)

	ListLinksForAccount(ctx context.Context, accountId AccountId, expired bool) ([]Link, error)
	ListLinksForPublicKey(ctx context.Context, publicKeyId PublicKeyId, expired bool) ([]Link, error)

	UpdateLink(ctx context.Context, accountId AccountId, publicKeyId PublicKeyId, expiresAt time.Time) (Link, error)

	DeleteLink(ctx context.Context, accountId AccountId, publicKeyId PublicKeyId) error

	// --- Deploy & Verify ---

	DeployAccount(ctx context.Context, userRequester UserRequester, accountId AccountId) (chan DeployProgressAccount, error)

	DeployAccounts(ctx context.Context, userRequester UserRequester, accountIds ...AccountId) (chan DeployProgressAccounts, error)

	VerifyAccount(ctx context.Context, userRequester UserRequester, accountId AccountId) (chan VerifyProgressAccount, error)

	VerifyAccounts(ctx context.Context, userRequester UserRequester, accountIds ...AccountId) (chan VerifyProgressAccounts, error)

	// --- Other ---

	ListAuditLogs(ctx context.Context, limit int) ([]AuditLog, error) // TODO doesn't account for filtering and pagination

	ListConnectorKeys(ctx context.Context) ([]string, error)

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
	IsGlobal  bool
	ExpiresAt time.Time
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

type Link struct {
	AccountId   AccountId
	PublicKeyId PublicKeyId
	ExpiresAt   time.Time
	// ...
}

type AuditLogId id
type AuditLog struct {
	Id        AuditLogId
	Timestamp time.Time
	Action    string
	Details   AuditLogDetails
	Metadata  AuditLogMetadata
}
type AuditLogDetail struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
type AuditLogDetails []AuditLogDetail
type AuditLogMetadata struct {
	Hostname string
	Hostuser string
	Referer  string
}

func escapeAuditLogDetailStr(str string) string {
	// escape string if it containes spaces or "
	if strings.ContainsAny(str, "\" ") {
		return "\"" + strings.ReplaceAll(str, "\"", "\\\"") + "\""
	}
	return str
}

func (a AuditLogDetail) String() string {
	return escapeAuditLogDetailStr(a.Key) + "=" + escapeAuditLogDetailStr(a.Value)
}

func (a AuditLogDetails) String() string {
	strs := make([]string, 0, len(a))
	for _, d := range a {
		strs = append(strs, d.String())
	}

	return strings.Join(strs, " ")
}

func (a AuditLogMetadata) String() string {
	return fmt.Sprintf("%s %s@%s", a.Referer, a.Hostuser, a.Hostname)
}

type (
	ProgressAccount        = connector.Progress
	DeployProgressAccount  = ProgressAccount
	VerifyProgressAccount  = ProgressAccount
	DeployProgressAccounts = ProgressAccounts
	VerifyProgressAccounts = ProgressAccounts
	UserRequester          = connector.UserRequester
)

type ProgressAccounts struct {
	Accounts map[AccountId]*ProgressAccount
}

func (dp ProgressAccounts) Progress() float64 {
	var total float64
	for _, dap := range dp.Accounts {
		total += dap.Progress
	}

	return total / float64(len(dp.Accounts))
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
