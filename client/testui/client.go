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
	"github.com/toeirei/keymaster/tags"
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
	publicKeys, err := c.ListPublicKeysLinkedToAccount(ctx, account.Id, false)
	if err != nil {
		return "", err
	}

	return strings.Join(slicest.Map(publicKeys, func(pk client.PublicKey) string {
		return fmt.Sprintf("%s %s %s", pk.Algorithm, pk.Data, pk.Comment)
	}), "\n"), nil
}

func (c *Client) accountDeployCache(account client.Account, deployCache string) string {
	return fmt.Sprintf("%s %s@%s:%d\n%s", account.DeployMethod, account.Username, account.Host, account.Port, deployCache)
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

func (c *Client) CreatePublicKey(ctx context.Context, key string, comment string, tags tags.Tags) (client.PublicKey, error) {
	c.publicKeyIdCounter++
	keyParts := strings.Split(key, " ")
	if len(keyParts) < 2 {
		return client.PublicKey{}, errors.New("invalid key provided")
	}
	// algorithm, data := keyParts[0], strings.Join(slicest.SliceTo(keyParts, 1, len(keyParts)), " ")
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
	return slicest.Filter(c.publicKeys, func(publicKey client.PublicKey) bool {
		return slices.Contains(ids, publicKey.Id)
	}), nil
}

func (c *Client) ListPublicKeys(ctx context.Context, tagMatcher string) ([]client.PublicKey, error) {
	if len(tagMatcher) == 0 {
		return slices.Clone(c.publicKeys), nil
	}

	expr, err := tags.ParseMatcher(tagMatcher)
	if err != nil {
		return nil, err
	}

	return slicest.Filter(c.publicKeys, func(publicKey client.PublicKey) bool {
		return expr.Eval(publicKey.Tags)
	}), nil
}

func (c *Client) ListPublicKeysLinkedToAccount(ctx context.Context, accountId client.AccountId, expired bool) ([]client.PublicKey, error) {
	links, err := c.ListLinksForAccount(ctx, accountId, expired)
	if err != nil {
		return nil, err
	}

	publicKeyss, err := slicest.MapXI(links, func(_ int, link client.Link) ([]client.PublicKey, error) {
		return c.ListPublicKeys(ctx, link.TagMatcher)
	})
	if err != nil {
		return nil, err
	}

	return slicest.Flatten(publicKeyss), nil
}

func (c *Client) UpdateLink(ctx context.Context, id client.LinkId, accountId client.AccountId, tagMatcher string, expiresAt time.Time) error {
	if i, ok := slices.BinarySearchFunc(c.links, id, func(link client.Link, id client.LinkId) int {
		return int(link.Id - id)
	}); ok {
		c.links[i].AccountId = accountId
		c.links[i].TagMatcher = tagMatcher
		c.links[i].ExpiresAt = expiresAt
		return nil
	}
	return fmt.Errorf("account with id %v not found", id)
}

func (c *Client) UpdatePublicKey(ctx context.Context, id client.PublicKeyId, comment string, tags tags.Tags) error {
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
	c.publicKeys = slicest.Filter(c.publicKeys, func(publicKey client.PublicKey) bool { return !slices.Contains(ids, publicKey.Id) })
	return nil
}

// --- Account Management ---

func (c *Client) CreateAccount(ctx context.Context, username string, host string, port int, deploymentMethod string, deploymentSecret string) (client.Account, error) {
	c.accountIdCounter++
	account := client.Account{c.accountIdCounter, username, host, port, deploymentMethod, deploymentSecret, ""}
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
	accounts := slicest.Filter(c.accounts, func(account client.Account) bool {
		return slices.Contains(ids, account.Id)
	})

	if len(accounts) != len(ids) {
		return nil, fmt.Errorf(
			"accounts with the following ids could not be found: %s",
			strings.Join(
				slicest.Map(
					slicest.Filter(ids, func(id client.AccountId) bool {
						_, ok := slices.BinarySearchFunc(accounts, id, func(account client.Account, id client.AccountId) int {
							return int(account.Id - id)
						})
						return !ok
					}),
					func(id client.AccountId) string { return fmt.Sprint(id) },
				),
				", ",
			),
		)
	}

	return accounts, nil
}

func (c *Client) ListAccounts(ctx context.Context) ([]client.Account, error) {
	return slices.Clone(c.accounts), nil
}

func (c *Client) ListAccountsLinkedToPublicKey(ctx context.Context, publicKeyId client.PublicKeyId, expired bool) ([]client.Account, error) {
	links, err := c.ListLinksForPublicKey(ctx, publicKeyId, expired)
	if err != nil {
		return nil, err
	}

	accountIds := slicest.Map(links, func(link client.Link) client.AccountId { return link.AccountId })

	return c.GetAccounts(ctx, accountIds...)
}

func (c *Client) ListAccountsDirty(ctx context.Context) ([]client.Account, error) {
	return slicest.FilterX(c.accounts, func(account client.Account) (bool, error) {
		return c.IsAccountDirty(ctx, account)
	})
}

func (c *Client) UpdateAccount(ctx context.Context, id client.AccountId, username string, host string, port int, deploymentMethod string, deploymentSecret string) error {
	if i, ok := slices.BinarySearchFunc(c.accounts, id, func(account client.Account, id client.AccountId) int {
		return int(account.Id - id)
	}); ok {
		c.accounts[i].Username = username
		c.accounts[i].Username = username
		c.accounts[i].Host = host
		c.accounts[i].Port = port
		c.accounts[i].DeployMethod = deploymentMethod
		c.accounts[i].DeploySecret = deploymentSecret
		return nil
	}
	return fmt.Errorf("account with id %v not found", id)
}

func (c *Client) DeleteAccounts(ctx context.Context, ids ...client.AccountId) error {
	c.accounts = slicest.Filter(c.accounts, func(account client.Account) bool { return !slices.Contains(ids, account.Id) })
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

func (c *Client) CreateLink(ctx context.Context, accountId client.AccountId, tagMatcher string, expiresAt time.Time) (client.Link, error) {
	c.linkIdCounter++
	link := client.Link{c.linkIdCounter, accountId, tagMatcher, expiresAt}
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
	return slicest.Filter(c.links, func(link client.Link) bool {
		return slices.Contains(ids, link.Id)
	}), nil
}

func (c *Client) ListLinksForAccount(ctx context.Context, accountId client.AccountId, expired bool) ([]client.Link, error) {
	return slicest.Filter(c.links, func(link client.Link) bool {
		return link.AccountId == accountId && (expired || time.Now().Before(link.ExpiresAt))
	}), nil
}

func (c *Client) ListLinksForPublicKey(ctx context.Context, publicKeyId client.PublicKeyId, expired bool) ([]client.Link, error) {
	publicKey, err := c.GetPublicKey(ctx, publicKeyId)
	if err != nil {
		return nil, err
	}

	return slicest.FilterX(c.links, func(link client.Link) (bool, error) {
		expr, err := tags.ParseMatcher(link.TagMatcher)
		if err != nil {
			return false, err
		}

		return expr.Eval(publicKey.Tags) && (expired || time.Now().Before(link.ExpiresAt)), nil
	})
}

func (c *Client) DeleteLinks(ctx context.Context, ids ...client.LinkId) error {
	c.links = slicest.Filter(c.links, func(link client.Link) bool { return !slices.Contains(ids, link.Id) })
	return nil
}

// --- Deploy & Verify ---

func (c *Client) DeployAccount(ctx context.Context, accountId client.AccountId) (chan client.DeployProgressAccount, error) {
	dpc, err := c.DeployAccounts(ctx, accountId)
	if err != nil {
		return nil, err
	}

	// convert channel to only report the single accounts progress
	dbac := make(chan client.DeployProgressAccount)
	go func() {
		defer close(dbac)

		for dp := range dpc {
			dbac <- *dp.Accounts[accountId]
		}
	}()

	return dbac, nil
}

func (c *Client) DeployAccounts(ctx context.Context, accountIds ...client.AccountId) (chan client.DeployProgressAccounts, error) {
	accounts, err := c.GetAccounts(ctx, accountIds...)
	if err != nil {
		return nil, err
	}

	deployDatas, err := slicest.MapX(accounts, func(a client.Account) (string, error) {
		return c.accountDeployData(ctx, a)
	})
	if err != nil {
		return nil, err
	}

	deployProgressChan := make(chan client.DeployProgressAccounts)
	deployProgress := client.DeployProgressAccounts{
		Accounts: slicest.ToMap(accounts, func(account client.Account) (client.AccountId, *client.DeployProgressAccount) {
			return account.Id, &client.DeployProgressAccount{0, "not started", nil}
		}),
	}

	go func() {
		defer close(deployProgressChan)

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

func (c *Client) VerifyAccount(ctx context.Context, accountId client.AccountId) (chan client.VerifyProgressAccount, error) {
	dpc, err := c.VerifyAccounts(ctx, accountId)
	if err != nil {
		return nil, err
	}

	// convert channel to only report the single accounts progress
	dbac := make(chan client.VerifyProgressAccount)
	go func() {
		defer close(dbac)

		for dp := range dpc {
			dbac <- *dp.Accounts[accountId]
		}
	}()

	return dbac, nil
}

func (c *Client) VerifyAccounts(ctx context.Context, accountIds ...client.AccountId) (chan client.VerifyProgressAccounts, error) {
	accounts, err := c.GetAccounts(ctx, accountIds...)
	if err != nil {
		return nil, err
	}

	deployDatas, err := slicest.MapX(accounts, func(a client.Account) (string, error) {
		return c.accountDeployData(ctx, a)
	})
	if err != nil {
		return nil, err
	}

	verifyProgressChan := make(chan client.VerifyProgressAccounts)
	verifyProgress := client.VerifyProgressAccounts{
		Accounts: slicest.ToMap(accounts, func(account client.Account) (client.AccountId, *client.VerifyProgressAccount) {
			return account.Id, &client.VerifyProgressAccount{0, "not started", nil}
		}),
	}

	go func() {
		defer close(verifyProgressChan)

		for i, account := range accounts {
			verifyProgress.Accounts[account.Id].Status = "verifing"
			verifyProgressChan <- verifyProgress

			// simulate deplay
			for _i := range 5 {
				time.Sleep(time.Millisecond * 100)
				verifyProgress.Accounts[account.Id].Progress = float64(_i+1) / 10
				verifyProgressChan <- verifyProgress
			}

			_i, ok := slices.BinarySearchFunc(c.accounts, account.Id, func(a client.Account, id client.AccountId) int { return int(a.Id - id) })
			if !ok {
				verifyProgress.Accounts[account.Id].Status = "error"
				verifyProgress.Accounts[account.Id].Progress = 1
				verifyProgress.Accounts[account.Id].Err = fmt.Errorf("account with id %v not found", account.Id)
				verifyProgressChan <- verifyProgress
				continue
			}

			// simulate deplay
			for _i := range 5 {
				time.Sleep(time.Millisecond * 100)
				verifyProgress.Accounts[account.Id].Progress = float64(_i+6) / 10
				verifyProgressChan <- verifyProgress
			}

			// simulate getting deployCache from remote
			deployCache := c.accountDeployCache(account, deployDatas[i])

			if c.accounts[_i].DeployCache != deployCache {
				verifyProgress.Accounts[account.Id].Status = "error"
				verifyProgress.Accounts[account.Id].Err = errors.New("account is out of sync")
			} else {
				verifyProgress.Accounts[account.Id].Status = "finished"
			}

			verifyProgress.Accounts[account.Id].Progress = 1
			verifyProgressChan <- verifyProgress
		}
	}()

	return verifyProgressChan, nil
}

// --- Other ---

func (c *Client) ListExistingTags(ctx context.Context) tags.Tags {
	return slicest.Reduce(c.publicKeys, func(publicKey client.PublicKey, tags tags.Tags) tags.Tags {
		return append(tags, publicKey.Tags...)
	})
}

func (c *Client) OnboardHost(ctx context.Context, host string, port int /* , gateway string, plugin string */, accountUsername string, deploymentKey string) (chan client.OnboardHostProgress, error) {
	return nil, errors.New("client.OnboardHost not implemented")
}

func (c *Client) DecommisionAccount(ctx context.Context, id client.AccountId) (chan client.DecommisionAccountProgress, error) {
	return nil, errors.New("client.DecommisionAccount not implemented")
}
