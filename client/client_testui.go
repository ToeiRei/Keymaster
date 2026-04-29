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

type TestUIClient struct {
	// local temporary repository for testing ui features
	publicKeys []PublicKey
	accounts   []Account
	links      []Link

	// id counter to simulate serial
	publicKeyIdCounter PublicKeyId
	accountIdCounter   AccountId
	linkIdCounter      LinkId
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
	c.publicKeyIdCounter++
	keyParts := strings.Split(key, " ")
	if len(keyParts) < 2 {
		return PublicKey{}, errors.New("invalid key provided")
	}
	// algorithm, data := keyParts[0], strings.Join(slices.SliceTo(keyParts, 1, len(keyParts)), " ")
	algorithm, data := keyParts[0], keyParts[1]
	publicKey := PublicKey{c.publicKeyIdCounter, algorithm, data, comment, tags}
	c.publicKeys = append(c.publicKeys, publicKey)
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
	c.publicKeys = slices.Filter(c.publicKeys, func(publicKey PublicKey) bool { return !slices.Contains(ids, publicKey.Id) })
	return nil
}

// --- Account Management ---

func (c *TestUIClient) CreateAccount(ctx context.Context, name string, host string, port int, deploymentMethod string, deploymentSecret string) (Account, error) {
	c.accountIdCounter++
	account := Account{c.accountIdCounter, name, host, port, deploymentMethod, deploymentSecret, ""}
	c.accounts = append(c.accounts, account)
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

func (c *TestUIClient) ListDirtyAccounts(ctx context.Context) ([]Account, error) {
	return slices.Filterx(c.accounts, func(account Account) (bool, error) {
		return c.IsAccountDirty(ctx, account)
	})
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
	c.accounts = slices.Filter(c.accounts, func(account Account) bool { return !slices.Contains(ids, account.Id) })
	return nil
}

func (c *TestUIClient) IsAccountDirty(ctx context.Context, account Account) (bool, error) {
	return false, nil
}

// --- Link Management ---

func (c *TestUIClient) CreateLink(ctx context.Context, accountID AccountId, tagFilter string, expiresAt time.Time) (Link, error) {
	c.linkIdCounter++
	link := Link{c.linkIdCounter, accountID, tagFilter, expiresAt}
	c.links = append(c.links, link)
	return link, nil
}

func (c *TestUIClient) GetLink(ctx context.Context, id LinkId) (Link, error) {
	if i, ok := slices.BinarySearchFunc(c.links, id, func(link Link, id LinkId) int {
		return int(link.Id - id)
	}); ok {
		return c.links[i], nil
	}
	return Link{}, fmt.Errorf("link with id %v not found", id)
}

func (c *TestUIClient) GetLinks(ctx context.Context, ids ...LinkId) ([]Link, error) {
	return slices.Filter(c.links, func(link Link) bool {
		return slices.Contains(ids, link.Id)
	}), nil
}

func (c *TestUIClient) ListPublicKeyLinks(ctx context.Context, accountID AccountId) ([]Link, error) {
	return nil, errors.New("client.ListPublicKeyLinks not implemented")
}

func (c *TestUIClient) ListAccountLinks(ctx context.Context, publicKeyID PublicKeyId) ([]Link, error) {
	return nil, errors.New("client.ListAccountLinks not implemented")
}

func (c *TestUIClient) ListPublicKeysForAccount(ctx context.Context, accountID AccountId) ([]PublicKey, error) {
	return nil, errors.New("client.ListPublicKeysForAccount not implemented")
}

func (c *TestUIClient) ListAccountsForPublicKey(ctx context.Context, publicKeyID PublicKeyId) ([]Account, error) {
	return nil, errors.New("client.ListAccountsForPublicKey not implemented")
}

func (c *TestUIClient) UpdateLink(ctx context.Context, id LinkId, accountId AccountId, tagFilter string, expiresAt time.Time) error {
	if i, ok := slices.BinarySearchFunc(c.links, id, func(link Link, id LinkId) int {
		return int(link.Id - id)
	}); ok {
		c.links[i].AccountId = accountId
		c.links[i].TagFilter = tagFilter
		c.links[i].ExpiresAt = expiresAt
		return nil
	}
	return fmt.Errorf("account with id %v not found", id)
}

func (c *TestUIClient) DeleteLinks(ctx context.Context, ids ...LinkId) error {
	c.links = slices.Filter(c.links, func(link Link) bool { return !slices.Contains(ids, link.Id) })
	return nil
}

// --- Other ---

func (c *TestUIClient) ListExistingTags(ctx context.Context) []string {
	return slicest.Reduce(c.publicKeys, func(publicKey PublicKey, tags []string) []string {
		return append(tags, publicKey.Tags...)
	})
}

func (c *TestUIClient) OnboardHost(ctx context.Context, host string, port int /* , gateway string, plugin string */, accountName string, deploymentKey string) (chan OnboardHostProgress, error) {
	return nil, errors.New("client.OnboardHost not implemented")
}

func (c *TestUIClient) DecommisionAccount(ctx context.Context, id AccountId) (chan DecommisionAccountProgress, error) {
	return nil, errors.New("client.DecommisionAccount not implemented")
}

func (c *TestUIClient) DeployPublicKeys(ctx context.Context, publicKeyID ...PublicKeyId) (chan DeployProgress, error) {
	return nil, errors.New("client.DeployPublicKeys not implemented")
}

func (c *TestUIClient) DeployAccounts(ctx context.Context, accountID ...AccountId) (chan DeployProgress, error) {
	return nil, errors.New("client.DeployAccounts not implemented")
}

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
