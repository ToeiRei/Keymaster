// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package client

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/toeirei/keymaster/core"
)

type Client struct {
	//lint:ignore U1000 Placeholder for future configuration wiring.
	config Config
	//lint:ignore U1000 Placeholder for future store wiring.
	store core.Store
	// NOTE:
	// log != audit_log
	// log is not meant for cli out
	//lint:ignore U1000 Placeholder for future logging wiring.
	log *log.Logger
}

// --- Mock types that will later be imported or defined seperately ---
type ID int
type PublicKey struct {
	// ...
}
type Target struct {
	// ...
}
type Account struct {
	// ...
}
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
func New(config Config, logger *log.Logger) (*Client, error) {
	return nil, errors.New("client.New not implemented")
}

// cleans up and closes all open connections,
// maybe be an ******* and set c to nil
func (c *Client) Close(ctx context.Context) error {
	return errors.New("client.Close not implemented")
}

// --- PublicKey Management ---

func (c *Client) CreatePublicKey(ctx context.Context, identity string, tags []string) (ID, error) {
	return 0, errors.New("client.CreatePublicKey not implemented")
}

func (c *Client) GetPublicKey(ctx context.Context, id ID) (PublicKey, error) {
	return PublicKey{}, errors.New("client.GetPublicKey not implemented")
}

func (c *Client) GetPublicKeys(ctx context.Context, id ...ID) ([]PublicKey, error) {
	return nil, errors.New("client.GetPublicKeys not implemented")
}

func (c *Client) ListPublicKeys(ctx context.Context, tagFilter string) ([]PublicKey, error) {
	return nil, errors.New("client.ListPublicKeys not implemented")
}

func (c *Client) UpdatePublicKeyTags(ctx context.Context, id ID, tags []string) error {
	return errors.New("client.UpdatePublicKeyTags not implemented")
}

func (c *Client) DeletePublicKeys(ctx context.Context, id ...ID) error {
	return errors.New("client.DeletePublicKeys not implemented")
}

// --- Target Management ---

func (c *Client) CreateTarget(ctx context.Context, host string, port int /* , gateway string, plugin string */) (ID, error) {
	return 0, errors.New("client.CreateTarget not implemented")
}

func (c *Client) GetTarget(ctx context.Context, id ID) (Target, error) {
	return Target{}, errors.New("client.GetTarget not implemented")
}

func (c *Client) GetTargets(ctx context.Context, id ...ID) ([]Target, error) {
	return nil, errors.New("client.GetTargets not implemented")
}

func (c *Client) ListTargets(ctx context.Context) ([]Target, error) {
	return nil, errors.New("client.ListTargets not implemented")
}

func (c *Client) UpdateTarget(ctx context.Context, id ID, target Target) error {
	return errors.New("client.UpdateTarget not implemented")
}

func (c *Client) DeleteTargets(ctx context.Context, id ...ID) error {
	return errors.New("client.DeleteTargets not implemented")
}

// --- Account Management ---

func (c *Client) CreateAccount(ctx context.Context, targetID ID, name string, deploymentKey string) (ID, error) {
	return 0, errors.New("client.CreateAccount not implemented")
}

func (c *Client) GetAccount(ctx context.Context, id ID) (Account, error) {
	return Account{}, errors.New("client.GetAccount not implemented")
}

func (c *Client) ListAccountsByTarget(ctx context.Context, targetID ID) ([]Account, error) {
	return nil, errors.New("client.ListAccountsByTarget not implemented")
}

func (c *Client) GetDirtyAccounts(ctx context.Context) ([]Account, error) {
	return nil, errors.New("client.GetDirtyAccounts not implemented")
}

// --- Tag to Account Management ---

// LinkTagToAccount maps a tag filter (e.g. "device:mobile&company:telekom") to an account
func (c *Client) LinkTagAccount(ctx context.Context, accountID ID, filter string, expiresAt time.Time) (ID, error) {
	return 0, errors.New("client.LinkTagAccount not implemented")
}

func (c *Client) UnLinkTagAccount(ctx context.Context, linkIDs ...ID) error {
	return errors.New("client.UnLinkTagAccount not implemented")
}

func (c *Client) ResolvePublicKeysForAccount(ctx context.Context, accountID ID) ([]PublicKey, error) {
	return nil, errors.New("client.ResolvePublicKeysForAccount not implemented")
}

func (c *Client) ResolveAccountsForPublicKey(ctx context.Context, publicKeyID ID) ([]Account, error) {
	return nil, errors.New("client.ResolveAccountsForPublicKey not implemented")
}

// --- Onboarding & Decommision ---

func (c *Client) OnboardHost(ctx context.Context, host string, port int /* , gateway string, plugin string */, accountName string, deploymentKey string) (chan OnboardHostProgress, error) {
	return nil, errors.New("client.OnboardHost not implemented")
}

func (c *Client) DecommisionTarget(ctx context.Context, id ID) (chan DecommisionTargetProgress, error) {
	return nil, errors.New("client.DecommisionTarget not implemented")
}

func (c *Client) DecommisionAccount(ctx context.Context, id ID) (chan DecommisionAccountProgress, error) {
	return nil, errors.New("client.DecommisionAccount not implemented")
}

// --- Deploy stuff ---

// Deploy handles the plugin-based deployment to the target
func (c *Client) DeployPublicKeys(ctx context.Context, publicKeyID ...ID) (chan DeployProgress, error) {
	return nil, errors.New("client.DeployPublicKeys not implemented")
}

func (c *Client) DeployTargets(ctx context.Context, targetID ...ID) (chan DeployProgress, error) {
	return nil, errors.New("client.DeployTargets not implemented")
}

func (c *Client) DeployAccounts(ctx context.Context, accountID ...ID) (chan DeployProgress, error) {
	return nil, errors.New("client.DeployAccounts not implemented")
}

func (c *Client) DeployAll(ctx context.Context) (chan DeployProgress, error) {
	return nil, errors.New("client.DeployAll not implemented")
}
