// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package client

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bobg/go-generics/v4/slices"
	"github.com/toeirei/keymaster/util/slicest"
)

//lint:ignore U1000 Placeholder for future test grant fields.
type TestUIGrant struct {
	accountID AccountId
	matcher   string
	expiresAt time.Time
}

type TestUIClient struct {
	// local temporary repository for testing ui features
	publicKeys []PublicKey
	accounts   []Account
	//lint:ignore U1000 Placeholder for future grant-related features.
	grants []TestUIGrant
	// id counter to simulate serial
	publicKeysID PublicKeyId
	accountsID   AccountId
}

// *[TestUIClient] implements [Client]
var _ Client = (*TestUIClient)(nil)

// --- Lifecycle & Initialization ---

func NewTestUIClient() *TestUIClient {
	return &TestUIClient{}
}

func (c *TestUIClient) Close(ctx context.Context) error {
	return nil
}

// --- PublicKey Management ---

func (c *TestUIClient) CreatePublicKey(ctx context.Context, key string, comment string, tags []string) (PublicKey, error) {
	keyParts := strings.Split(key, " ")
	if len(keyParts) < 2 {
		return PublicKey{}, errors.New("invalid key provided")
	}
	// algorithm, data := keyParts[0], strings.Join(slices.SliceTo(keyParts, 1, len(keyParts)), " ")
	algorithm, data := keyParts[0], keyParts[1]
	publicKey := PublicKey{c.publicKeysID, algorithm, data, comment, tags}
	c.publicKeys = append(c.publicKeys, publicKey)
	c.publicKeysID++
	return publicKey, nil
}

func (c *TestUIClient) GetPublicKey(ctx context.Context, id PublicKeyId) (PublicKey, error) {
	if i, ok := slices.BinarySearchFunc(c.publicKeys, id, func(publicKey PublicKey, id PublicKeyId) int {
		return int(publicKey.Id - id)
	}); ok {
		return c.publicKeys[i], nil
	}
	return PublicKey{}, fmt.Errorf("public key with id %v not found", id)
}

func (c *TestUIClient) GetPublicKeys(ctx context.Context, ids ...PublicKeyId) ([]PublicKey, error) {
	return slices.Filter(c.publicKeys, func(publicKey PublicKey) bool {
		return slices.Contains(ids, publicKey.Id)
	}), nil
}

func (c *TestUIClient) ListPublicKeys(ctx context.Context, tagFilter string) ([]PublicKey, error) {
	if tagFilter == "" {
		return slices.Clone(c.publicKeys), nil
	}
	// WARNING does not realy repect the tagFilter
	return slices.Filter(c.publicKeys, func(publicKey PublicKey) bool {
		return slices.Contains(publicKey.Tags, tagFilter)
	}), nil
}

func (c *TestUIClient) UpdatePublicKey(ctx context.Context, id PublicKeyId, comment string, tags []string) error {
	if i, ok := slices.BinarySearchFunc(c.publicKeys, id, func(publicKey PublicKey, id PublicKeyId) int {
		return int(publicKey.Id - id)
	}); ok {
		c.publicKeys[i].Comment = comment
		c.publicKeys[i].Tags = tags
		return nil
	}
	return fmt.Errorf("public key with id %v not found", id)
}

func (c *TestUIClient) DeletePublicKeys(ctx context.Context, ids ...PublicKeyId) error {
	indexs := make([]int, 0, len(c.publicKeys))
	for i, publicKey := range c.publicKeys {
		if slices.Contains(ids, publicKey.Id) {
			indexs = append(indexs, i)
		}
	}
	slices.Reverse(indexs)
	for _, i := range indexs {
		c.publicKeys = slices.Delete(c.publicKeys, i, i)
	}
	return nil
}

// --- Account Management ---

func (c *TestUIClient) CreateAccount(ctx context.Context, name string, host string, port int, deploymentMethod string, deploymentSecret string) (Account, error) {
	account := Account{c.accountsID, name, host, port, deploymentMethod, deploymentSecret, ""}
	c.accounts = append(c.accounts, account)
	c.accountsID++
	return account, nil
}

func (c *TestUIClient) GetAccount(ctx context.Context, id AccountId) (Account, error) {
	if i, ok := slices.BinarySearchFunc(c.accounts, id, func(account Account, id AccountId) int {
		return int(account.Id - id)
	}); ok {
		return c.accounts[i], nil
	}
	return Account{}, fmt.Errorf("account with id %v not found", id)
}

func (c *TestUIClient) GetAccounts(ctx context.Context, ids ...AccountId) ([]Account, error) {
	return slices.Filter(c.accounts, func(account Account) bool {
		return slices.Contains(ids, account.Id)
	}), nil
}

func (c *TestUIClient) ListAccounts(ctx context.Context) ([]Account, error) {
	return slices.Clone(c.accounts), nil
}

func (c *TestUIClient) UpdateAccount(ctx context.Context, id AccountId, name string, host string, port int, deploymentMethod string, deploymentSecret string) error {
	if i, ok := slices.BinarySearchFunc(c.accounts, id, func(account Account, id AccountId) int {
		return int(account.Id - id)
	}); ok {
		c.accounts[i].Name = name
		c.accounts[i].Name = name
		c.accounts[i].Host = host
		c.accounts[i].Port = port
		c.accounts[i].DeployMethod = deploymentMethod
		c.accounts[i].DeploySecret = deploymentSecret
		return nil
	}
	return fmt.Errorf("account with id %v not found", id)
}

func (c *TestUIClient) DeleteAccounts(ctx context.Context, ids ...AccountId) error {
	indexs := make([]int, 0, len(c.accounts))
	for i, account := range c.accounts {
		if slices.Contains(ids, account.Id) {
			indexs = append(indexs, i)
		}
	}
	slices.Reverse(indexs)
	for _, i := range indexs {
		c.accounts = slices.Delete(c.accounts, i, i)
	}
	return nil
}

func (c *TestUIClient) IsAccountDirty(ctx context.Context, account Account) (bool, error) {
	return false, nil
}

func (c *TestUIClient) GetDirtyAccounts(ctx context.Context) ([]Account, error) {
	return slices.Filterx(c.accounts, func(account Account) (bool, error) {
		return c.IsAccountDirty(ctx, account)
	})
}

// --- Tag & Account-PublicKey relation Management ---

func (c *TestUIClient) ListExistingTags(ctx context.Context) []string {
	return slicest.Reduce(c.publicKeys, func(publicKey PublicKey, tags []string) []string {
		return append(tags, publicKey.Tags...)
	})
}

// LinkTagAccount associates a tag filter (e.g. "device:mobile&company:telekom") with
// an `accountID` until `expiresAt`.
func (c *TestUIClient) CreateLink(ctx context.Context, accountID AccountId, filter string, expiresAt time.Time) (Link, error) {
	return Link{}, errors.New("client.LinkTagAccount not implemented")
}

// UnLinkTagAccount removes previously created tag-account links.
func (c *TestUIClient) DeleteLinks(ctx context.Context, linkIDs ...LinkId) error {
	return errors.New("client.UnLinkTagAccount not implemented")
}

func (c *TestUIClient) ResolvePublicKeyLinks(ctx context.Context, accountID AccountId) ([]Link, error) {
	return nil, errors.New("client.ResolvePublicKeyLinks not implemented")
}

func (c *TestUIClient) ResolveAccountLinks(ctx context.Context, publicKeyID PublicKeyId) ([]Link, error) {
	return nil, errors.New("client.ResolveAccountLinks not implemented")
}

// ResolvePublicKeysForAccount returns public keys applicable to `accountID`.
func (c *TestUIClient) ResolvePublicKeysForAccount(ctx context.Context, accountID AccountId) ([]PublicKey, error) {
	return nil, errors.New("client.ResolvePublicKeysForAccount not implemented")
}

// ResolveAccountsForPublicKey returns accounts that a public key applies to.
func (c *TestUIClient) ResolveAccountsForPublicKey(ctx context.Context, publicKeyID PublicKeyId) ([]Account, error) {
	return nil, errors.New("client.ResolveAccountsForPublicKey not implemented")
}

// --- Onboarding & Decommision ---

// OnboardHost starts onboarding of a host and returns a progress channel.
func (c *TestUIClient) OnboardHost(ctx context.Context, host string, port int /* , gateway string, plugin string */, accountName string, deploymentKey string) (chan OnboardHostProgress, error) {
	return nil, errors.New("client.OnboardHost not implemented")
}

// DecommisionAccount decommissions an account and streams progress.
func (c *TestUIClient) DecommisionAccount(ctx context.Context, id AccountId) (chan DecommisionAccountProgress, error) {
	return nil, errors.New("client.DecommisionAccount not implemented")
}

// --- Deploy stuff ---

// DeployPublicKeys deploys public keys to their accounts and reports progress on the returned channel.
func (c *TestUIClient) DeployPublicKeys(ctx context.Context, publicKeyID ...PublicKeyId) (chan DeployProgress, error) {
	return nil, errors.New("client.DeployPublicKeys not implemented")
}

// DeployAccounts deploys to the specified account ids and streams progress.
func (c *TestUIClient) DeployAccounts(ctx context.Context, accountID ...AccountId) (chan DeployProgress, error) {
	return nil, errors.New("client.DeployAccounts not implemented")
}

// DeployAll triggers deployment for all pending targets/accounts.
func (c *TestUIClient) DeployAll(ctx context.Context) (chan DeployProgress, error) {
	ch := make(chan DeployProgress)

	accountProgress := make(map[*Account]float64, len(c.accounts))
	for _, account := range c.accounts {
		accountProgress[&account] = 0
	}

	go func() {
		for i := float64(0); i <= 1; i += 0.2 {
			time.Sleep(time.Second)
			for account := range accountProgress {
				accountProgress[account] = i
			}
			ch <- DeployProgress{i, accountProgress}
		}
		close(ch)
	}()
	return ch, nil
}
