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

// Package client provides a high-level client API for interacting with
// Keymaster programmatically. The concrete implementation is currently
// partial; most methods are placeholders and return not-implemented errors.

// --- Mock types that will later be imported or defined seperately ---
// ID is a local identifier type used by the client API.
type ID int

// PublicKey represents a public key record.
type PublicKey struct {
	// ...
}

// Target represents a remote host or endpoint to deploy keys to.
type Target struct {
	// ...
}

// Account represents an account on a target host.
type Account struct {
	// ...
}

// DeployProgress reports incremental progress for a deployment operation.
type DeployProgress struct {
	Done bool
	// ...
}

// OnboardHostProgress reports progress during host onboarding.
type OnboardHostProgress struct {
	Done bool
	// ...
}

// DecommisionTargetProgress reports progress for target decommissioning.
type DecommisionTargetProgress struct {
	Done bool
	// ...
}

// DecommisionAccountProgress reports progress for account decommissioning.
type DecommisionAccountProgress struct {
	Done bool
	// ...
}

// --- Lifecycle & Initialization ---

// connect to db,
// auto migrate db to current version,
// initialize store,
// maybe run some offline chores
// New creates and initializes a new `Client` from the provided `Config` and
// `logger`. The implementation should connect to the backing store, run any
// migrations and return a ready-to-use client. Currently unimplemented.
func New(config Config, logger *log.Logger) (*Client, error) {
	return nil, errors.New("client.New not implemented")
}

// cleans up and closes all open connections,
// maybe be an ******* and set c to nil
// Close cleans up resources held by the client and closes any open
// connections. Calls should pass a context for cancellation/timeouts.
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
	// ListTargets returns all known deployment targets.
	return nil, errors.New("client.ListTargets not implemented")
}

func (c *Client) UpdateTarget(ctx context.Context, id ID, target Target) error {
	// UpdateTarget updates the target record identified by `id`.
	return errors.New("client.UpdateTarget not implemented")
}

func (c *Client) DeleteTargets(ctx context.Context, id ...ID) error {
	// DeleteTargets removes targets identified by the given ids.
	return errors.New("client.DeleteTargets not implemented")
}

// --- Account Management ---

// CreateAccount creates an account on `targetID` with the given name and
// deployment key.
func (c *Client) CreateAccount(ctx context.Context, targetID ID, name string, deploymentKey string) (ID, error) {
	return 0, errors.New("client.CreateAccount not implemented")
}

// GetAccount returns account metadata for `id`.
func (c *Client) GetAccount(ctx context.Context, id ID) (Account, error) {
	return Account{}, errors.New("client.GetAccount not implemented")
}

// ListAccountsByTarget lists accounts associated with `targetID`.
func (c *Client) ListAccountsByTarget(ctx context.Context, targetID ID) ([]Account, error) {
	return nil, errors.New("client.ListAccountsByTarget not implemented")
}

// GetDirtyAccounts returns accounts that require reconciliation or deployment.
func (c *Client) GetDirtyAccounts(ctx context.Context) ([]Account, error) {
	return nil, errors.New("client.GetDirtyAccounts not implemented")
}

// --- Tag to Account Management ---

// LinkTagToAccount maps a tag filter (e.g. "device:mobile&company:telekom") to an account
// LinkTagAccount associates a tag filter with an `accountID` until `expiresAt`.
func (c *Client) LinkTagAccount(ctx context.Context, accountID ID, filter string, expiresAt time.Time) (ID, error) {
	return 0, errors.New("client.LinkTagAccount not implemented")
}

// UnLinkTagAccount removes previously created tag-account links.
func (c *Client) UnLinkTagAccount(ctx context.Context, linkIDs ...ID) error {
	return errors.New("client.UnLinkTagAccount not implemented")
}

// ResolvePublicKeysForAccount returns public keys applicable to `accountID`.
func (c *Client) ResolvePublicKeysForAccount(ctx context.Context, accountID ID) ([]PublicKey, error) {
	return nil, errors.New("client.ResolvePublicKeysForAccount not implemented")
}

// ResolveAccountsForPublicKey returns accounts that a public key applies to.
func (c *Client) ResolveAccountsForPublicKey(ctx context.Context, publicKeyID ID) ([]Account, error) {
	return nil, errors.New("client.ResolveAccountsForPublicKey not implemented")
}

// --- Onboarding & Decommision ---

// OnboardHost starts onboarding of a host and returns a progress channel.
func (c *Client) OnboardHost(ctx context.Context, host string, port int /* , gateway string, plugin string */, accountName string, deploymentKey string) (chan OnboardHostProgress, error) {
	return nil, errors.New("client.OnboardHost not implemented")
}

// DecommisionTarget decommissions a deployment target and streams progress.
func (c *Client) DecommisionTarget(ctx context.Context, id ID) (chan DecommisionTargetProgress, error) {
	return nil, errors.New("client.DecommisionTarget not implemented")
}

// DecommisionAccount decommissions an account and streams progress.
func (c *Client) DecommisionAccount(ctx context.Context, id ID) (chan DecommisionAccountProgress, error) {
	return nil, errors.New("client.DecommisionAccount not implemented")
}

// --- Deploy stuff ---

// DeployPublicKeys deploys public keys to their target accounts and reports
// progress on the returned channel.
func (c *Client) DeployPublicKeys(ctx context.Context, publicKeyID ...ID) (chan DeployProgress, error) {
	return nil, errors.New("client.DeployPublicKeys not implemented")
}

// DeployTargets deploys to the specified target ids and streams progress.
func (c *Client) DeployTargets(ctx context.Context, targetID ...ID) (chan DeployProgress, error) {
	return nil, errors.New("client.DeployTargets not implemented")
}

// DeployAccounts deploys to the specified account ids and streams progress.
func (c *Client) DeployAccounts(ctx context.Context, accountID ...ID) (chan DeployProgress, error) {
	return nil, errors.New("client.DeployAccounts not implemented")
}

// DeployAll triggers deployment for all pending targets/accounts.
func (c *Client) DeployAll(ctx context.Context) (chan DeployProgress, error) {
	return nil, errors.New("client.DeployAll not implemented")
}
