// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package client

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/toeirei/keymaster/config"
	"github.com/toeirei/keymaster/core"
	"github.com/toeirei/keymaster/core/db"
)

type BunClient struct {
	//lint:ignore U1000 Placeholder for future configuration wiring.
	config config.Config
	//lint:ignore U1000 Placeholder for future store wiring.
	store core.Store
	// NOTE:
	// log != audit_log
	// log is not meant for cli out
	//lint:ignore U1000 Placeholder for future logging wiring.
	log *log.Logger
}

// *BunClient implements Client
var _ Client = (*BunClient)(nil)

// --- Lifecycle & Initialization ---

// New creates and initializes a new `BunClient` from the provided `Config` and
// `logger`. The implementation should connect to the backing store, run any
// migrations and return a ready-to-use client. Currently unimplemented.
func NewBunClient(config config.Config, logger *log.Logger) (*BunClient, error) {
	// db.New(config.Database.Type, config.Database.Dsn)
	store, err := db.NewStoreFromDSN(config.Database.Type, config.Database.Dsn)
	_ = store // TODO can't use store yet, as it does not implement core.Store (wich it shouldn't but hey)
	if err != nil {
		return nil, err
	}

	return &BunClient{
		config: config,
		log:    logger,
		// store:  core.Store(store),
	}, nil
}

// Close cleans up resources held by the client and closes any open
// connections. Calls should pass a context for cancellation/timeouts.
func (c *BunClient) Close(ctx context.Context) error {
	return errors.New("client.Close not implemented")
}

// --- PublicKey Management ---

func (c *BunClient) CreatePublicKey(ctx context.Context, identity string, tags []string) (PublicKey, error) {
	return PublicKey{}, errors.New("client.CreatePublicKey not implemented")
}

func (c *BunClient) GetPublicKey(ctx context.Context, id ID) (PublicKey, error) {
	return PublicKey{}, errors.New("client.GetPublicKey not implemented")
}

func (c *BunClient) GetPublicKeys(ctx context.Context, ids ...ID) ([]PublicKey, error) {
	return nil, errors.New("client.GetPublicKeys not implemented")
}

func (c *BunClient) ListPublicKeys(ctx context.Context, tagFilter string) ([]PublicKey, error) {
	return nil, errors.New("client.ListPublicKeys not implemented")
}

func (c *BunClient) UpdatePublicKeyTags(ctx context.Context, id ID, tags []string) error {
	return errors.New("client.UpdatePublicKeyTags not implemented")
}

func (c *BunClient) DeletePublicKeys(ctx context.Context, ids ...ID) error {
	return errors.New("client.DeletePublicKeys not implemented")
}

// --- Target Management ---

func (c *BunClient) CreateTarget(ctx context.Context, host string, port int /* , gateway string, plugin string */) (Target, error) {
	return Target{}, errors.New("client.CreateTarget not implemented")
}

func (c *BunClient) GetTarget(ctx context.Context, id ID) (Target, error) {
	return Target{}, errors.New("client.GetTarget not implemented")
}

func (c *BunClient) GetTargets(ctx context.Context, ids ...ID) ([]Target, error) {
	return nil, errors.New("client.GetTargets not implemented")
}

func (c *BunClient) ListTargets(ctx context.Context) ([]Target, error) {
	return nil, errors.New("client.ListTargets not implemented")
}

func (c *BunClient) UpdateTarget(ctx context.Context, id ID, target Target) error {
	return errors.New("client.UpdateTarget not implemented")
}

func (c *BunClient) DeleteTargets(ctx context.Context, ids ...ID) error {
	return errors.New("client.DeleteTargets not implemented")
}

// --- Account Management ---

func (c *BunClient) CreateAccount(ctx context.Context, targetID ID, name string, deploymentKey string) (Account, error) {
	return Account{}, errors.New("client.CreateAccount not implemented")
}

func (c *BunClient) GetAccount(ctx context.Context, id ID) (Account, error) {
	return Account{}, errors.New("client.GetAccount not implemented")
}

func (c *BunClient) GetAccounts(ctx context.Context, ids ...ID) ([]Account, error) {
	return nil, errors.New("client.GetAccounts not implemented")
}

func (c *BunClient) ListAccountsByTarget(ctx context.Context, targetID ID) ([]Account, error) {
	return nil, errors.New("client.ListAccountsByTarget not implemented")
}

func (c *BunClient) DeleteAccounts(ctx context.Context, ids ...ID) error {
	return errors.New("client.DeleteAccounts not implemented")
}

func (c *BunClient) GetDirtyAccounts(ctx context.Context) ([]Account, error) {
	return nil, errors.New("client.GetDirtyAccounts not implemented")
}

// --- Tag to Account Management ---

// LinkTagAccount associates a tag filter (e.g. "device:mobile&company:telekom") with
// an `accountID` until `expiresAt`.
func (c *BunClient) LinkTagAccount(ctx context.Context, accountID ID, filter string, expiresAt time.Time) (ID, error) {
	return 0, errors.New("client.LinkTagAccount not implemented")
}

// UnLinkTagAccount removes previously created tag-account links.
func (c *BunClient) UnLinkTagAccount(ctx context.Context, linkIDs ...ID) error {
	return errors.New("client.UnLinkTagAccount not implemented")
}

// ResolvePublicKeysForAccount returns public keys applicable to `accountID`.
func (c *BunClient) ResolvePublicKeysForAccount(ctx context.Context, accountID ID) ([]PublicKey, error) {
	return nil, errors.New("client.ResolvePublicKeysForAccount not implemented")
}

// ResolveAccountsForPublicKey returns accounts that a public key applies to.
func (c *BunClient) ResolveAccountsForPublicKey(ctx context.Context, publicKeyID ID) ([]Account, error) {
	return nil, errors.New("client.ResolveAccountsForPublicKey not implemented")
}

// --- Onboarding & Decommision ---

// OnboardHost starts onboarding of a host and returns a progress channel.
func (c *BunClient) OnboardHost(ctx context.Context, host string, port int /* , gateway string, plugin string */, accountName string, deploymentKey string) (chan OnboardHostProgress, error) {
	return nil, errors.New("client.OnboardHost not implemented")
}

// DecommisionTarget decommissions a deployment target and streams progress.
func (c *BunClient) DecommisionTarget(ctx context.Context, id ID) (chan DecommisionTargetProgress, error) {
	return nil, errors.New("client.DecommisionTarget not implemented")
}

// DecommisionAccount decommissions an account and streams progress.
func (c *BunClient) DecommisionAccount(ctx context.Context, id ID) (chan DecommisionAccountProgress, error) {
	return nil, errors.New("client.DecommisionAccount not implemented")
}

// --- Deploy stuff ---

// DeployPublicKeys deploys public keys to their target accounts and reports
// progress on the returned channel.
func (c *BunClient) DeployPublicKeys(ctx context.Context, publicKeyID ...ID) (chan DeployProgress, error) {
	return nil, errors.New("client.DeployPublicKeys not implemented")
}

// DeployTargets deploys to the specified target ids and streams progress.
func (c *BunClient) DeployTargets(ctx context.Context, targetID ...ID) (chan DeployProgress, error) {
	return nil, errors.New("client.DeployTargets not implemented")
}

// DeployAccounts deploys to the specified account ids and streams progress.
func (c *BunClient) DeployAccounts(ctx context.Context, accountID ...ID) (chan DeployProgress, error) {
	return nil, errors.New("client.DeployAccounts not implemented")
}

// DeployAll triggers deployment for all pending targets/accounts.
func (c *BunClient) DeployAll(ctx context.Context) (chan DeployProgress, error) {
	return nil, errors.New("client.DeployAll not implemented")
}
