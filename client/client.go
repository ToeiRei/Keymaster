// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package client

import (
	"context"
	"log"
	"time"

	"github.com/toeirei/keymaster/internal/core"
)

type Client struct {
	config Config
	store  core.Store
	// NOTE:
	// log != audit_log
	// log is not meant for cli out
	log *log.Logger
}

// --- Mock types that will later be imported or defined seperately ---
type ID int
type PublicKey struct{}
type Target struct{}
type Account struct{}
type DeployProgress struct {
	Done bool
	// ...
}
type OnboardHostProgress struct {
	Done bool
	// ...
}
type DecommisionTargetProgress struct {
	Done bool
	// ...
}
type DecommisionAccountProgress struct {
	Done bool
	// ...
}

// --- Lifecycle & Initialization ---

// connect to db,
// auto migrate db to current version,
// initialize store,
// maybe run some offline chores
func New(config Config, logger *log.Logger) (*Client, error)

// cleans up and closes all open connections,
// maybe be an ******* and set c to nil
func (c *Client) Close(ctx context.Context) error

// --- PublicKey Management ---

func (c *Client) CreatePublicKey(ctx context.Context, identity string, tags []string) (ID, error)
func (c *Client) GetPublicKey(ctx context.Context, id ID) (PublicKey, error)
func (c *Client) GetPublicKeys(ctx context.Context, id ...ID) ([]PublicKey, error)
func (c *Client) ListPublicKeys(ctx context.Context, tagFilter string) ([]PublicKey, error)
func (c *Client) UpdatePublicKeyTags(ctx context.Context, id ID, tags []string) error
func (c *Client) DeletePublicKeys(ctx context.Context, id ...ID) error

// --- Target Management ---

func (c *Client) CreateTarget(ctx context.Context, host string, port int /* , gateway string, plugin string */) (ID, error)
func (c *Client) GetTarget(ctx context.Context, id ID) (Target, error)
func (c *Client) GetTargets(ctx context.Context, id ...ID) ([]Target, error)
func (c *Client) ListTargets(ctx context.Context) ([]Target, error)
func (c *Client) UpdateTarget(ctx context.Context, id ID, target Target) error
func (c *Client) DeleteTargets(ctx context.Context, id ...ID) error

// --- Account Management ---

func (c *Client) CreateAccount(ctx context.Context, targetID ID, name string, deploymentKey string) (ID, error)
func (c *Client) GetAccount(ctx context.Context, id ID) (Account, error)
func (c *Client) ListAccountsByTarget(ctx context.Context, targetID ID) ([]Account, error)
func (c *Client) GetDirtyAccounts(ctx context.Context) ([]Account, error)

// --- Tag to Account Management ---

// LinkTagToAccount maps a tag filter (e.g. "device:mobile&company:telekom") to an account
func (c *Client) LinkTagAccount(ctx context.Context, accountID ID, filter string, expiresAt time.Time) (ID, error)
func (c *Client) UnLinkTagAccount(ctx context.Context, linkIDs ...ID) error
func (c *Client) ResolvePublicKeysForAccount(ctx context.Context, accountID ID) ([]PublicKey, error)
func (c *Client) ResolveAccountsForPublicKey(ctx context.Context, publicKeyID ID) ([]Account, error)

// --- Onboarding & Decommision ---

func (c *Client) OnboardHost(ctx context.Context, host string, port int /* , gateway string, plugin string */, accountName string, deploymentKey string) (chan OnboardHostProgress, error)
func (c *Client) DecommisionTarget(ctx context.Context, id ID) (chan DecommisionTargetProgress, error)
func (c *Client) DecommisionAccount(ctx context.Context, id ID) (chan DecommisionAccountProgress, error)

// --- Deploy stuff ---

// Deploy handles the plugin-based deployment to the target
func (c *Client) DeployPublicKeys(ctx context.Context, publicKeyID ...ID) (chan DeployProgress, error)
func (c *Client) DeployTargets(ctx context.Context, targetID ...ID) (chan DeployProgress, error)
func (c *Client) DeployAccounts(ctx context.Context, accountID ...ID) (chan DeployProgress, error)
func (c *Client) DeployAll(ctx context.Context) (chan DeployProgress, error)
