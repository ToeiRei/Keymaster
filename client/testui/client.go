// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package testui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/user"
	"slices"
	"strings"
	"time"

	"github.com/jinzhu/copier"
	"github.com/toeirei/keymaster/client"
	"github.com/toeirei/keymaster/tags"
	"github.com/toeirei/keymaster/ui/tui/util"
	"github.com/toeirei/keymaster/util/slicest"
)

type Client struct {
	// local temporary repository for testing ui features
	publicKeys   map[client.PublicKeyId]client.PublicKey
	accounts     map[client.AccountId]client.Account
	links        map[client.LinkId]client.Link
	auditLogs    []client.AuditLog
	remoteStates map[client.AccountId]string

	// id counter to simulate serial
	publicKeyIdCounter client.PublicKeyId
	accountIdCounter   client.AccountId
	linkIdCounter      client.LinkId
	auditLogIdCounter  client.AuditLogId
}

// *[Client] implements [client.Client]
var _ client.Client = (*Client)(nil)

// --- utils ---

func (c *Client) writeAuditLog(
	action string,
	details string,
	extraMetadata *map[string]string,
) error {
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}

	osuser, err := user.Current()
	if err != nil {
		return err
	}

	c.auditLogIdCounter++
	c.auditLogs = append(c.auditLogs, client.AuditLog{
		c.auditLogIdCounter,
		time.Now(),

		client.AuditLogMetadata{
			hostname,
			osuser.Username,
			util.DerefOrZeroValue(extraMetadata),
		},

		action,
		details,
	})

	return nil
}

func (c *Client) accountDeployData(ctx context.Context, account client.Account) (string, error) {
	publicKeys, err := c.ListPublicKeysLinkedToAccount(ctx, account.Id, false)
	if err != nil {
		return "", err
	}

	slices.SortFunc(publicKeys, func(pk1, pk2 client.PublicKey) int { return int(pk1.Id - pk2.Id) })

	return strings.Join(slicest.Map(publicKeys, func(pk client.PublicKey) string {
		return fmt.Sprintf("%s %s %s", pk.Algorithm, pk.Data, pk.Comment)
	}), "\n"), nil
}

func (c *Client) accountDeployCache(account client.Account, deployCache string) string {
	return fmt.Sprintf("%s %s@%s:%d\n%s", account.DeployMethod, account.Username, account.Host, account.Port, deployCache)
}

// --- Lifecycle & Initialization ---

func NewClient() *Client {
	return &Client{
		publicKeys:   make(map[client.PublicKeyId]client.PublicKey),
		accounts:     make(map[client.AccountId]client.Account),
		links:        make(map[client.LinkId]client.Link),
		remoteStates: make(map[client.AccountId]string),
	}
}

func (c *Client) Close(ctx context.Context) error {
	return nil
}

// NOT THREAD SAFE! ONLY FOR TESTING!
func (c *Client) WithTransaction(ctx context.Context, fn func(c client.Client) error) error {
	// create copy of client to use in transaction
	transactionClient := &Client{}
	if err := copier.Copy(transactionClient, c); err != nil {
		return err
	}

	// run callback with transaction client
	err := fn(transactionClient)
	if err != nil {
		return err
	}

	// apply changes
	*c = *transactionClient
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
	publicKey := client.PublicKey{Id: c.publicKeyIdCounter, Algorithm: algorithm, Data: data, Comment: comment, Tags: tags}
	c.publicKeys[publicKey.Id] = publicKey

	c.writeAuditLog("public_key.create", fmt.Sprintf("%#v", publicKey), nil)
	return publicKey, nil
}

func (c *Client) GetPublicKey(ctx context.Context, id client.PublicKeyId) (client.PublicKey, error) {
	if publicKey, ok := c.publicKeys[id]; ok {
		return publicKey, nil
	}
	return client.PublicKey{}, fmt.Errorf("public key with id %v not found", id)
}

func (c *Client) GetPublicKeys(ctx context.Context, ids ...client.PublicKeyId) ([]client.PublicKey, error) {
	return slicest.MapX(ids, func(id client.PublicKeyId) (client.PublicKey, error) {
		return c.GetPublicKey(ctx, id)
	})
}

func (c *Client) ListPublicKeys(ctx context.Context, tagMatcher string) ([]client.PublicKey, error) {
	if len(tagMatcher) == 0 {
		return slicest.MapValues(c.publicKeys), nil
	}

	expr, err := tags.ParseMatcher(tagMatcher)
	if err != nil {
		return nil, err
	}

	return slicest.Filter(slicest.MapValues(c.publicKeys), func(publicKey client.PublicKey) bool {
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

func (c *Client) UpdateLink(ctx context.Context, id client.LinkId, accountId client.AccountId, tagMatcher string, expiresAt time.Time) (client.Link, error) {
	if link, ok := c.links[id]; ok {
		link.AccountId = accountId
		link.TagMatcher = tagMatcher
		link.ExpiresAt = expiresAt
		c.links[id] = link

		c.writeAuditLog("link.update", fmt.Sprintf("%#v", link), nil)
		return link, nil
	}
	return client.Link{}, fmt.Errorf("account with id %v not found", id)
}

func (c *Client) UpdatePublicKey(ctx context.Context, id client.PublicKeyId, comment string, tags tags.Tags) (client.PublicKey, error) {
	if publicKey, ok := c.publicKeys[id]; ok {
		publicKey.Comment = comment
		publicKey.Tags = tags
		c.publicKeys[id] = publicKey

		c.writeAuditLog("public_key.update", fmt.Sprintf("%#v", publicKey), nil)
		return publicKey, nil
	}
	return client.PublicKey{}, fmt.Errorf("public key with id %v not found", id)
}

func (c *Client) DeletePublicKeys(ctx context.Context, ids ...client.PublicKeyId) error {
	for _, id := range ids {
		if _, ok := c.publicKeys[id]; !ok {
			return fmt.Errorf("public key with id %v not found", id)
		}
	}

	for _, id := range ids {
		delete(c.publicKeys, id)
		c.writeAuditLog("public_key.delete", fmt.Sprintf("%#v", id), nil)
	}

	return nil
}

// --- Account Management ---

func (c *Client) CreateAccount(ctx context.Context, username string, host string, port int, deploymentMethod string, deploymentSecret string) (client.Account, error) {
	c.accountIdCounter++
	account := client.Account{Id: c.accountIdCounter, Username: username, Host: host, Port: port, DeployMethod: deploymentMethod, DeploySecret: deploymentSecret, DeployCache: ""}
	c.accounts[account.Id] = account

	c.writeAuditLog("account.create", fmt.Sprintf("%#v", account), nil)
	return account, nil
}

func (c *Client) GetAccount(ctx context.Context, id client.AccountId) (client.Account, error) {
	if account, ok := c.accounts[id]; ok {
		return account, nil
	}
	return client.Account{}, fmt.Errorf("account with id %v not found", id)
}

func (c *Client) GetAccounts(ctx context.Context, ids ...client.AccountId) ([]client.Account, error) {
	return slicest.MapX(ids, func(id client.AccountId) (client.Account, error) {
		return c.GetAccount(ctx, id)
	})
}

func (c *Client) ListAccounts(ctx context.Context) ([]client.Account, error) {
	return slicest.MapValues(c.accounts), nil
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
	return slicest.FilterX(slicest.MapValues(c.accounts), func(account client.Account) (bool, error) {
		return c.IsAccountDirty(ctx, account)
	})
}

func (c *Client) UpdateAccount(ctx context.Context, id client.AccountId, username string, host string, port int, deploymentMethod string, deploymentSecret string) (client.Account, error) {
	if account, ok := c.accounts[id]; ok {
		account.Username = username
		account.Username = username
		account.Host = host
		account.Port = port
		account.DeployMethod = deploymentMethod
		account.DeploySecret = deploymentSecret
		c.accounts[id] = account

		c.writeAuditLog("account.update", fmt.Sprintf("%#v", account), nil)
		return account, nil
	}
	return client.Account{}, fmt.Errorf("account with id %v not found", id)
}

func (c *Client) DeleteAccounts(ctx context.Context, ids ...client.AccountId) error {
	for _, id := range ids {
		if _, ok := c.accounts[id]; !ok {
			return fmt.Errorf("account with id %v not found", id)
		}
	}

	for _, id := range ids {
		delete(c.accounts, id)
		c.writeAuditLog("account.delete", fmt.Sprintf("%#v", id), nil)
	}

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
	link := client.Link{Id: c.linkIdCounter, AccountId: accountId, TagMatcher: tagMatcher, ExpiresAt: expiresAt}
	c.links[link.Id] = link

	c.writeAuditLog("link.create", fmt.Sprintf("%#v", link), nil)
	return link, nil
}

func (c *Client) GetLink(ctx context.Context, id client.LinkId) (client.Link, error) {
	if link, ok := c.links[id]; ok {
		return link, nil
	}
	return client.Link{}, fmt.Errorf("link with id %v not found", id)
}

func (c *Client) GetLinks(ctx context.Context, ids ...client.LinkId) ([]client.Link, error) {
	return slicest.MapX(ids, func(id client.LinkId) (client.Link, error) {
		return c.GetLink(ctx, id)
	})
}

func (c *Client) ListLinksForAccount(ctx context.Context, accountId client.AccountId, expired bool) ([]client.Link, error) {
	return slicest.Filter(slicest.MapValues(c.links), func(link client.Link) bool {
		return link.AccountId == accountId && (expired || time.Now().Before(link.ExpiresAt))
	}), nil
}

func (c *Client) ListLinksForPublicKey(ctx context.Context, publicKeyId client.PublicKeyId, expired bool) ([]client.Link, error) {
	publicKey, err := c.GetPublicKey(ctx, publicKeyId)
	if err != nil {
		return nil, err
	}

	return slicest.FilterX(slicest.MapValues(c.links), func(link client.Link) (bool, error) {
		expr, err := tags.ParseMatcher(link.TagMatcher)
		if err != nil {
			return false, err
		}

		return expr.Eval(publicKey.Tags) && (expired || time.Now().Before(link.ExpiresAt)), nil
	})
}

func (c *Client) DeleteLinks(ctx context.Context, ids ...client.LinkId) error {
	for _, id := range ids {
		if _, ok := c.links[id]; !ok {
			return fmt.Errorf("link with id %v not found", id)
		}
	}

	for _, id := range ids {
		delete(c.links, id)
		c.writeAuditLog("link.delete", fmt.Sprintf("%#v", id), nil)
	}

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
			return account.Id, &client.DeployProgressAccount{Progress: 0, Status: "not started", Err: nil}
		}),
	}

	checkContextCanceled := func(accountId client.AccountId, deployProgress client.DeployProgressAccounts) bool {
		if ctx.Err() != nil {
			deployProgress.Accounts[accountId].Progress = 1
			if errors.Is(ctx.Err(), context.Canceled) {
				deployProgress.Accounts[accountId].Status = "canceled"
				deployProgress.Accounts[accountId].Err = errors.New("canceled")
			} else {
				deployProgress.Accounts[accountId].Status = "error"
				deployProgress.Accounts[accountId].Err = ctx.Err()
			}
			return true
		}
		return false
	}

	go func() {
		defer close(deployProgressChan)

	accountLoop:
		for i, account := range accounts {
			if checkContextCanceled(account.Id, deployProgress) {
				continue accountLoop
			}

			deployProgress.Accounts[account.Id].Status = "deploying"
			deployProgressChan <- deployProgress

			// simulate deplay
			for _i := range 5 {
				time.Sleep(time.Millisecond * 100)
				if checkContextCanceled(account.Id, deployProgress) {
					continue accountLoop
				}
				deployProgress.Accounts[account.Id].Progress = float64(_i+1) / 10
				deployProgressChan <- deployProgress
			}

			// potential error i guess
			ok := true
			if !ok {
				deployProgress.Accounts[account.Id].Status = "error"
				deployProgress.Accounts[account.Id].Progress = 1
				deployProgress.Accounts[account.Id].Err = fmt.Errorf("Some weird error on account with id %v", account.Id)
				deployProgressChan <- deployProgress
				continue
			}

			// simulate deplay
			for _i := range 5 {
				time.Sleep(time.Millisecond * 100)
				if checkContextCanceled(account.Id, deployProgress) {
					continue accountLoop
				}
				deployProgress.Accounts[account.Id].Progress = float64(_i+6) / 10
				deployProgressChan <- deployProgress
			}

			// simulate deploying data to remote
			c.remoteStates[account.Id] = c.accountDeployCache(account, deployDatas[i])

			c.writeAuditLog("account.deploy", fmt.Sprintf("%#v", account), nil)

			// update accounts deploy cache
			_account := c.accounts[account.Id]
			_account.DeployCache = c.remoteStates[account.Id]
			c.accounts[account.Id] = _account

			deployProgress.Accounts[account.Id].Status = "finished"
			deployProgress.Accounts[account.Id].Progress = 1
			deployProgressChan <- deployProgress
		}
		deployProgressChan <- deployProgress
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
			return account.Id, &client.VerifyProgressAccount{Progress: 0, Status: "not started", Err: nil}
		}),
	}

	checkContextCanceled := func(accountId client.AccountId, deployProgress client.DeployProgressAccounts) bool {
		if ctx.Err() != nil {
			deployProgress.Accounts[accountId].Progress = 1
			if errors.Is(ctx.Err(), context.Canceled) {
				deployProgress.Accounts[accountId].Status = "canceled"
				deployProgress.Accounts[accountId].Err = errors.New("canceled")
			} else {
				deployProgress.Accounts[accountId].Status = "error"
				deployProgress.Accounts[accountId].Err = ctx.Err()
			}
			return true
		}
		return false
	}

	go func() {
		defer close(verifyProgressChan)

	accountLoop:
		for i, account := range accounts {
			if checkContextCanceled(account.Id, verifyProgress) {
				continue accountLoop
			}

			verifyProgress.Accounts[account.Id].Status = "verifing"
			verifyProgressChan <- verifyProgress

			// simulate deplay
			for _i := range 5 {
				time.Sleep(time.Millisecond * 100)
				if checkContextCanceled(account.Id, verifyProgress) {
					continue accountLoop
				}
				verifyProgress.Accounts[account.Id].Progress = float64(_i+1) / 10
				verifyProgressChan <- verifyProgress
			}

			ok := true
			if !ok {
				verifyProgress.Accounts[account.Id].Status = "error"
				verifyProgress.Accounts[account.Id].Progress = 1
				verifyProgress.Accounts[account.Id].Err = fmt.Errorf("Some weird error on account with id %v", account.Id)
				verifyProgressChan <- verifyProgress
				continue
			}

			// simulate deplay
			for _i := range 5 {
				time.Sleep(time.Millisecond * 100)
				if checkContextCanceled(account.Id, verifyProgress) {
					continue accountLoop
				}
				verifyProgress.Accounts[account.Id].Progress = float64(_i+6) / 10
				verifyProgressChan <- verifyProgress
			}

			// simulate getting remoteState from remote
			remoteState, hasState := c.remoteStates[account.Id]

			if !hasState || c.accountDeployCache(account, deployDatas[i]) != remoteState {
				// update accounts deploy cache to reflect remotes state
				_account := c.accounts[account.Id]
				_account.DeployCache = c.remoteStates[account.Id]
				c.accounts[account.Id] = _account

				verifyProgress.Accounts[account.Id].Status = "error"
				verifyProgress.Accounts[account.Id].Err = errors.New("account is out of sync")
			} else {
				verifyProgress.Accounts[account.Id].Status = "finished"
			}

			c.writeAuditLog("account.verify", fmt.Sprintf("%#v", account), nil)

			verifyProgress.Accounts[account.Id].Progress = 1
			verifyProgressChan <- verifyProgress
		}
		verifyProgressChan <- verifyProgress
	}()

	return verifyProgressChan, nil
}

// --- Other ---

func (c *Client) ListAuditLogs(ctx context.Context, limit int) ([]client.AuditLog, error) {
	var logs []client.AuditLog
	if limit <= 0 {
		logs = c.auditLogs
	} else {
		logs = c.auditLogs[len(c.auditLogs)-min(len(c.auditLogs), limit):]
	}

	return append([]client.AuditLog(nil), logs...), nil
}

func (c *Client) ListExistingTags(ctx context.Context) tags.Tags {
	tags := make(map[tags.Tag]struct{}) // abuse map as set
	for _, publicKey := range c.publicKeys {
		for _, tag := range publicKey.Tags {
			tags[tag] = struct{}{}
		}
	}
	return slicest.MapKeys(tags)
}

func (c *Client) OnboardHost(ctx context.Context, host string, port int /* , gateway string, plugin string */, accountUsername string, deploymentKey string) (chan client.OnboardHostProgress, error) {
	return nil, errors.New("client.OnboardHost not implemented")
}

func (c *Client) DecommisionAccount(ctx context.Context, id client.AccountId) (chan client.DecommisionAccountProgress, error) {
	return nil, errors.New("client.DecommisionAccount not implemented")
}
