// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package testui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bobg/go-generics/v4/slices"
	"github.com/jinzhu/copier"
	"github.com/toeirei/keymaster/client"
	"github.com/toeirei/keymaster/util/slicest"
)

type Client struct {
	// local temporary repository for testing ui features
	publicKeys []client.PublicKey
	accounts   []client.Account
	links      []client.Link

	// id counter to simulate serial
	publicKeyIdCounter client.PublicKeyId
	accountIdCounter   client.AccountId
	linkIdCounter      client.LinkId
}

// *[Client] implements [client.Client]
var _ client.Client = (*Client)(nil)

// --- utils ---

func (c *Client) accountDeployData(ctx context.Context, account client.Account) (string, error) {
	publicKeys, err := c.ListPublicKeysForAccount(ctx, account.Id)
	if err != nil {
		return "", err
	}

	return strings.Join(slicest.Map(publicKeys, func(pk client.PublicKey) string {
		return fmt.Sprintf("%s %s %s", pk.Algorithm, pk.Data, pk.Comment)
	}), "\n"), nil
}

func (c *Client) accountDeployCache(account client.Account, deployCache string) string {
	return fmt.Sprintf("%s %s@%s:%d\n%s", account.DeployMethod, account.Name, account.Host, account.Port, deployCache)
}

func (c *Client) deployAccounts(ctx context.Context, accounts ...client.Account) (chan client.DeployProgress, error) {
	deployDatas, err := slicest.MapX(accounts, func(a client.Account) (string, error) {
		return c.accountDeployData(ctx, a)
	})
	if err != nil {
		return nil, err
	}

	deployProgressChan := make(chan client.DeployProgress)
	deployProgress := client.DeployProgress{
		Accounts: slicest.ToMap(accounts, func(account client.Account) (client.AccountId, *client.DeployAccountProgress) {
			return account.Id, &client.DeployAccountProgress{0, "not started", nil}
		}),
	}

	go func() {
		for i, account := range accounts {
			deployProgress.Accounts[account.Id].Status = "deploying"
			deployProgressChan <- deployProgress

			// simulate deplay
			for _i := range 5 {
				time.Sleep(time.Millisecond * 100)
				deployProgress.Accounts[account.Id].Progress = float64(_i+1) / 10
				deployProgressChan <- deployProgress
			}

			_i, ok := slices.BinarySearchFunc(c.accounts, account.Id, func(a client.Account, id client.AccountId) int { return int(a.Id - id) })
			if !ok {
				deployProgress.Accounts[account.Id].Status = "error"
				deployProgress.Accounts[account.Id].Progress = 1
				deployProgress.Accounts[account.Id].Err = fmt.Errorf("account with id %v not found", account.Id)
				deployProgressChan <- deployProgress
				continue
			}

			// simulate deplay
			for _i := range 5 {
				time.Sleep(time.Millisecond * 100)
				deployProgress.Accounts[account.Id].Progress = float64(_i+6) / 10
				deployProgressChan <- deployProgress
			}

			c.accounts[_i].DeployCache = c.accountDeployCache(account, deployDatas[i])
			deployProgress.Accounts[account.Id].Status = "finished"
			deployProgress.Accounts[account.Id].Progress = 1
			deployProgressChan <- deployProgress
		}
	}()

	return deployProgressChan, nil
}

// --- Lifecycle & Initialization ---

func NewClient() *Client {
	return &Client{}
}

func (c *Client) Close(ctx context.Context) error {
	return nil
}

// NOT THREAD SAFE! ONLY FOR TESTING!
func (c *Client) WithTransaction(ctx context.Context, fn func(c client.Client) error) error {
	// create copy of client to use in transaction
	var transactionClient *Client
	copier.Copy(transactionClient, c)

	// run callback with transaction client
	err := fn(c)
	if err != nil {
		return err
	}

	// apply changes
	c = transactionClient
	return nil
}

// --- client.PublicKey Management ---

func (c *Client) CreatePublicKey(ctx context.Context, key string, comment string, tags []string) (client.PublicKey, error) {
	c.publicKeyIdCounter++
	keyParts := strings.Split(key, " ")
	if len(keyParts) < 2 {
		return client.PublicKey{}, errors.New("invalid key provided")
	}
	// algorithm, data := keyParts[0], strings.Join(slices.SliceTo(keyParts, 1, len(keyParts)), " ")
	algorithm, data := keyParts[0], keyParts[1]
	publicKey := client.PublicKey{c.publicKeyIdCounter, algorithm, data, comment, tags}
	c.publicKeys = append(c.publicKeys, publicKey)
	return publicKey, nil
}

func (c *Client) GetPublicKey(ctx context.Context, id client.PublicKeyId) (client.PublicKey, error) {
	if i, ok := slices.BinarySearchFunc(c.publicKeys, id, func(publicKey client.PublicKey, id client.PublicKeyId) int {
		return int(publicKey.Id - id)
	}); ok {
		return c.publicKeys[i], nil
	}
	return client.PublicKey{}, fmt.Errorf("public key with id %v not found", id)
}

func (c *Client) GetPublicKeys(ctx context.Context, ids ...client.PublicKeyId) ([]client.PublicKey, error) {
	return slices.Filter(c.publicKeys, func(publicKey client.PublicKey) bool {
		return slices.Contains(ids, publicKey.Id)
	}), nil
}

func (c *Client) ListPublicKeys(ctx context.Context, tagFilter string) ([]client.PublicKey, error) {
	if tagFilter == "" {
		return slices.Clone(c.publicKeys), nil
	}
	// WARNING does not realy repect the tagFilter
	return slices.Filter(c.publicKeys, func(publicKey client.PublicKey) bool {
		return slices.Contains(publicKey.Tags, tagFilter)
	}), nil
}

func (c *Client) UpdatePublicKey(ctx context.Context, id client.PublicKeyId, comment string, tags []string) error {
	if i, ok := slices.BinarySearchFunc(c.publicKeys, id, func(publicKey client.PublicKey, id client.PublicKeyId) int {
		return int(publicKey.Id - id)
	}); ok {
		c.publicKeys[i].Comment = comment
		c.publicKeys[i].Tags = tags
		return nil
	}
	return fmt.Errorf("public key with id %v not found", id)
}

func (c *Client) DeletePublicKeys(ctx context.Context, ids ...client.PublicKeyId) error {
	c.publicKeys = slices.Filter(c.publicKeys, func(publicKey client.PublicKey) bool { return !slices.Contains(ids, publicKey.Id) })
	return nil
}

// --- Account Management ---

func (c *Client) CreateAccount(ctx context.Context, name string, host string, port int, deploymentMethod string, deploymentSecret string) (client.Account, error) {
	c.accountIdCounter++
	account := client.Account{c.accountIdCounter, name, host, port, deploymentMethod, deploymentSecret, ""}
	c.accounts = append(c.accounts, account)
	return account, nil
}

func (c *Client) GetAccount(ctx context.Context, id client.AccountId) (client.Account, error) {
	if i, ok := slices.BinarySearchFunc(c.accounts, id, func(account client.Account, id client.AccountId) int {
		return int(account.Id - id)
	}); ok {
		return c.accounts[i], nil
	}
	return client.Account{}, fmt.Errorf("account with id %v not found", id)
}

func (c *Client) GetAccounts(ctx context.Context, ids ...client.AccountId) ([]client.Account, error) {
	return slices.Filter(c.accounts, func(account client.Account) bool {
		return slices.Contains(ids, account.Id)
	}), nil
}

func (c *Client) ListAccounts(ctx context.Context) ([]client.Account, error) {
	return slices.Clone(c.accounts), nil
}

func (c *Client) ListDirtyAccounts(ctx context.Context) ([]client.Account, error) {
	return slices.Filterx(c.accounts, func(account client.Account) (bool, error) {
		return c.IsAccountDirty(ctx, account)
	})
}

func (c *Client) UpdateAccount(ctx context.Context, id client.AccountId, name string, host string, port int, deploymentMethod string, deploymentSecret string) error {
	if i, ok := slices.BinarySearchFunc(c.accounts, id, func(account client.Account, id client.AccountId) int {
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

func (c *Client) DeleteAccounts(ctx context.Context, ids ...client.AccountId) error {
	c.accounts = slices.Filter(c.accounts, func(account client.Account) bool { return !slices.Contains(ids, account.Id) })
	return nil
}

func (c *Client) IsAccountDirty(ctx context.Context, account client.Account) (bool, error) {
	deployData, err := c.accountDeployData(ctx, account)
	if err != nil {
		return true, err
	}

	return c.accountDeployCache(account, deployData) != account.DeployCache, nil
}

// --- client.Link Management ---

func (c *Client) CreateLink(ctx context.Context, accountId client.AccountId, tagFilter string, expiresAt time.Time) (client.Link, error) {
	c.linkIdCounter++
	link := client.Link{c.linkIdCounter, accountId, tagFilter, expiresAt}
	c.links = append(c.links, link)
	return link, nil
}

func (c *Client) GetLink(ctx context.Context, id client.LinkId) (client.Link, error) {
	if i, ok := slices.BinarySearchFunc(c.links, id, func(link client.Link, id client.LinkId) int {
		return int(link.Id - id)
	}); ok {
		return c.links[i], nil
	}
	return client.Link{}, fmt.Errorf("link with id %v not found", id)
}

func (c *Client) GetLinks(ctx context.Context, ids ...client.LinkId) ([]client.Link, error) {
	return slices.Filter(c.links, func(link client.Link) bool {
		return slices.Contains(ids, link.Id)
	}), nil
}

func (c *Client) ListLinksAccount(ctx context.Context, accountId client.AccountId) ([]client.Link, error) {
	return slices.Filter(c.links, func(link client.Link) bool {
		return link.AccountId == accountId
	}), nil
}

func (c *Client) ListLinksPublicKey(ctx context.Context, publicKeyId client.PublicKeyId) ([]client.Link, error) {
	return nil, errors.New("client.ListAccountLinks not implemented")
}

func (c *Client) ListPublicKeysForAccount(ctx context.Context, accountId client.AccountId) ([]client.PublicKey, error) {
	links, err := c.ListLinksAccount(ctx, accountId)
	if err != nil {
		return nil, err
	}

	publicKeyss, err := slices.Mapx(links, func(_ int, link client.Link) ([]client.PublicKey, error) {
		return c.ListPublicKeys(ctx, link.TagFilter)
	})
	if err != nil {
		return nil, err
	}

	return slicest.Flatten(publicKeyss), nil
}

func (c *Client) ListAccountsForPublicKey(ctx context.Context, publicKeyId client.PublicKeyId) ([]client.Account, error) {
	return nil, errors.New("client.ListAccountsForPublicKey not implemented")
}

func (c *Client) UpdateLink(ctx context.Context, id client.LinkId, accountId client.AccountId, tagFilter string, expiresAt time.Time) error {
	if i, ok := slices.BinarySearchFunc(c.links, id, func(link client.Link, id client.LinkId) int {
		return int(link.Id - id)
	}); ok {
		c.links[i].AccountId = accountId
		c.links[i].TagFilter = tagFilter
		c.links[i].ExpiresAt = expiresAt
		return nil
	}
	return fmt.Errorf("account with id %v not found", id)
}

func (c *Client) DeleteLinks(ctx context.Context, ids ...client.LinkId) error {
	c.links = slices.Filter(c.links, func(link client.Link) bool { return !slices.Contains(ids, link.Id) })
	return nil
}

// --- Other ---

func (c *Client) ListExistingTags(ctx context.Context) []string {
	return slicest.Reduce(c.publicKeys, func(publicKey client.PublicKey, tags []string) []string {
		return append(tags, publicKey.Tags...)
	})
}

func (c *Client) OnboardHost(ctx context.Context, host string, port int /* , gateway string, plugin string */, accountName string, deploymentKey string) (chan client.OnboardHostProgress, error) {
	return nil, errors.New("client.OnboardHost not implemented")
}

func (c *Client) DecommisionAccount(ctx context.Context, id client.AccountId) (chan client.DecommisionAccountProgress, error) {
	return nil, errors.New("client.DecommisionAccount not implemented")
}

func (c *Client) DeployPublicKeys(ctx context.Context, publicKeyId ...client.PublicKeyId) (chan client.DeployProgress, error) {
	return nil, errors.New("client.DeployPublicKeys not implemented")
}

func (c *Client) DeployAccounts(ctx context.Context, accountIds ...client.AccountId) (chan client.DeployProgress, error) {
	accounts, err := c.GetAccounts(ctx, accountIds...)
	if err != nil {
		return nil, err
	}

	return c.deployAccounts(ctx, accounts...)
}

func (c *Client) DeployAll(ctx context.Context) (chan client.DeployProgress, error) {
	accounts, err := c.ListAccounts(ctx)
	if err != nil {
		return nil, err
	}

	return c.deployAccounts(ctx, accounts...)
}
