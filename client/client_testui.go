// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package client

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/bobg/go-generics/v4/slices"
)

type TestUIGrant struct {
	accountID ID
	matcher   string
	expiresAt time.Time
}

type TestUIClient struct {
	// logger kinda still here... not used tho
	log *log.Logger
	// local temporary repository for testing ui features
	publicKeys []PublicKey
	targets    []Target
	accounts   []Account
	grants     []TestUIGrant
	// id counter to simulate serial
	publicKeysID ID
	targetsID    ID
	accountsID   ID
}

// *TestUIClient implements Client
var _ Client = (*TestUIClient)(nil)

// --- Lifecycle & Initialization ---

func NewTestUIClient(logger *log.Logger) *TestUIClient {
	return &TestUIClient{
		log: logger,
	}
}

func (c *TestUIClient) Close(ctx context.Context) error {
	return nil
}

// --- PublicKey Management ---

func (c *TestUIClient) CreatePublicKey(ctx context.Context, identity string, tags []string) (PublicKey, error) {
	publicKey := PublicKey{c.publicKeysID, identity, tags}
	c.publicKeys = append(c.publicKeys, publicKey)
	c.publicKeysID++
	return publicKey, nil
}

func (c *TestUIClient) GetPublicKey(ctx context.Context, id ID) (PublicKey, error) {
	if i, ok := slices.BinarySearchFunc(c.publicKeys, id, func(publicKey PublicKey, id ID) int {
		return int(publicKey.id - id)
	}); ok {
		return c.publicKeys[i], nil
	}
	return PublicKey{}, fmt.Errorf("PublicKey with id %v not found", id)
}

func (c *TestUIClient) GetPublicKeys(ctx context.Context, ids ...ID) ([]PublicKey, error) {
	return slices.Filter(c.publicKeys, func(publicKey PublicKey) bool {
		return slices.Contains(ids, publicKey.id)
	}), nil
}

func (c *TestUIClient) ListPublicKeys(ctx context.Context, tagFilter string) ([]PublicKey, error) {
	// WARNING does not realy repect the tagFilter
	return slices.Filter(c.publicKeys, func(publicKey PublicKey) bool {
		return slices.Contains(publicKey.tags, tagFilter)
	}), nil
}

func (c *TestUIClient) UpdatePublicKeyTags(ctx context.Context, id ID, tags []string) error {
	if i, ok := slices.BinarySearchFunc(c.publicKeys, id, func(publicKey PublicKey, id ID) int {
		return int(publicKey.id - id)
	}); ok {
		c.publicKeys[i].tags = tags
		return nil
	}
	return fmt.Errorf("PublicKey with id %v not found", id)
}

func (c *TestUIClient) DeletePublicKeys(ctx context.Context, ids ...ID) error {
	indexs := make([]int, 0, len(c.publicKeys))
	for i, publicKey := range c.publicKeys {
		if slices.Contains(ids, publicKey.id) {
			indexs = append(indexs, i)
		}
	}
	slices.Reverse(indexs)
	for _, i := range indexs {
		c.publicKeys = slices.Delete(c.publicKeys, i, i)
	}
	return nil
}

// --- Target Management ---

func (c *TestUIClient) CreateTarget(ctx context.Context, host string, port int /* , gateway string, plugin string */) (Target, error) {
	target := Target{c.targetsID, host, port}
	c.targets = append(c.targets, target)
	c.targetsID++
	return target, nil
}

func (c *TestUIClient) GetTarget(ctx context.Context, id ID) (Target, error) {
	if i, ok := slices.BinarySearchFunc(c.targets, id, func(target Target, id ID) int {
		return int(target.id - id)
	}); ok {
		return c.targets[i], nil
	}
	return Target{}, fmt.Errorf("Target with id %v not found", id)
}

func (c *TestUIClient) GetTargets(ctx context.Context, ids ...ID) ([]Target, error) {
	return slices.Filter(c.targets, func(target Target) bool {
		return slices.Contains(ids, target.id)
	}), nil
}

func (c *TestUIClient) ListTargets(ctx context.Context) ([]Target, error) {
	return slices.Filter(c.targets, func(target Target) bool { return true }), nil
}

func (c *TestUIClient) UpdateTarget(ctx context.Context, id ID, target Target) error {
	if i, ok := slices.BinarySearchFunc(c.targets, id, func(target Target, id ID) int {
		return int(target.id - id)
	}); ok {
		c.targets[i].host = target.host
		c.targets[i].port = target.port
		return nil
	}
	return fmt.Errorf("Target with id %v not found", id)
}

func (c *TestUIClient) DeleteTargets(ctx context.Context, ids ...ID) error {
	indexs := make([]int, 0, len(c.targets))
	for i, target := range c.targets {
		if slices.Contains(ids, target.id) {
			indexs = append(indexs, i)
		}
	}
	slices.Reverse(indexs)
	for _, i := range indexs {
		c.targets = slices.Delete(c.targets, i, i)
	}
	return nil
}

// --- Account Management ---

func (c *TestUIClient) CreateAccount(ctx context.Context, targetID ID, name string, deploymentKey string) (Account, error) {
	account := Account{c.accountsID, targetID, name, deploymentKey}
	c.accounts = append(c.accounts, account)
	c.accountsID++
	return account, nil
}

func (c *TestUIClient) GetAccount(ctx context.Context, id ID) (Account, error) {
	if i, ok := slices.BinarySearchFunc(c.accounts, id, func(account Account, id ID) int {
		return int(account.id - id)
	}); ok {
		return c.accounts[i], nil
	}
	return Account{}, fmt.Errorf("Account with id %v not found", id)
}

func (c *TestUIClient) GetAccounts(ctx context.Context, ids ...ID) ([]Account, error) {
	return slices.Filter(c.accounts, func(account Account) bool {
		return slices.Contains(ids, account.id)
	}), nil
}

func (c *TestUIClient) ListAccountsByTarget(ctx context.Context, targetID ID) ([]Account, error) {
	return slices.Filter(c.accounts, func(account Account) bool {
		return account.targetID == targetID
	}), nil
}

func (c *TestUIClient) DeleteAccounts(ctx context.Context, ids ...ID) error {
	indexs := make([]int, 0, len(c.accounts))
	for i, account := range c.accounts {
		if slices.Contains(ids, account.id) {
			indexs = append(indexs, i)
		}
	}
	slices.Reverse(indexs)
	for _, i := range indexs {
		c.accounts = slices.Delete(c.accounts, i, i)
	}
	return nil
}

func (c *TestUIClient) GetDirtyAccounts(ctx context.Context) ([]Account, error) {
	return nil, errors.New("client.GetDirtyAccounts not implemented")
}

// --- Tag to Account Management ---

// LinkTagAccount associates a tag filter (e.g. "device:mobile&company:telekom") with
// an `accountID` until `expiresAt`.
func (c *TestUIClient) LinkTagAccount(ctx context.Context, accountID ID, filter string, expiresAt time.Time) (ID, error) {
	return 0, errors.New("client.LinkTagAccount not implemented")
}

// UnLinkTagAccount removes previously created tag-account links.
func (c *TestUIClient) UnLinkTagAccount(ctx context.Context, linkIDs ...ID) error {
	return errors.New("client.UnLinkTagAccount not implemented")
}

// ResolvePublicKeysForAccount returns public keys applicable to `accountID`.
func (c *TestUIClient) ResolvePublicKeysForAccount(ctx context.Context, accountID ID) ([]PublicKey, error) {
	return nil, errors.New("client.ResolvePublicKeysForAccount not implemented")
}

// ResolveAccountsForPublicKey returns accounts that a public key applies to.
func (c *TestUIClient) ResolveAccountsForPublicKey(ctx context.Context, publicKeyID ID) ([]Account, error) {
	return nil, errors.New("client.ResolveAccountsForPublicKey not implemented")
}

// --- Onboarding & Decommision ---

// OnboardHost starts onboarding of a host and returns a progress channel.
func (c *TestUIClient) OnboardHost(ctx context.Context, host string, port int /* , gateway string, plugin string */, accountName string, deploymentKey string) (chan OnboardHostProgress, error) {
	return nil, errors.New("client.OnboardHost not implemented")
}

// DecommisionTarget decommissions a deployment target and streams progress.
func (c *TestUIClient) DecommisionTarget(ctx context.Context, id ID) (chan DecommisionTargetProgress, error) {
	return nil, errors.New("client.DecommisionTarget not implemented")
}

// DecommisionAccount decommissions an account and streams progress.
func (c *TestUIClient) DecommisionAccount(ctx context.Context, id ID) (chan DecommisionAccountProgress, error) {
	return nil, errors.New("client.DecommisionAccount not implemented")
}

// --- Deploy stuff ---

// DeployPublicKeys deploys public keys to their target accounts and reports
// progress on the returned channel.
func (c *TestUIClient) DeployPublicKeys(ctx context.Context, publicKeyID ...ID) (chan DeployProgress, error) {
	return nil, errors.New("client.DeployPublicKeys not implemented")
}

// DeployTargets deploys to the specified target ids and streams progress.
func (c *TestUIClient) DeployTargets(ctx context.Context, targetID ...ID) (chan DeployProgress, error) {
	return nil, errors.New("client.DeployTargets not implemented")
}

// DeployAccounts deploys to the specified account ids and streams progress.
func (c *TestUIClient) DeployAccounts(ctx context.Context, accountID ...ID) (chan DeployProgress, error) {
	return nil, errors.New("client.DeployAccounts not implemented")
}

// DeployAll triggers deployment for all pending targets/accounts.
func (c *TestUIClient) DeployAll(ctx context.Context) (chan DeployProgress, error) {
	ch := make(chan DeployProgress)

	targetProgress := make(map[Target]float32, len(c.targets))
	for _, target := range c.targets {
		targetProgress[target] = 0
	}

	go func() {
		for i := float32(0); i <= 1; i += 0.2 {
			time.Sleep(time.Second)
			for target := range targetProgress {
				targetProgress[target] = i
			}
			ch <- DeployProgress{i, targetProgress}
		}
		close(ch)
	}()
	return ch, nil
}
