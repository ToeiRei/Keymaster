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
	"github.com/toeirei/keymaster/connector"
	_ "github.com/toeirei/keymaster/connector/mock" // activate the mock connector so ListConnectorKeys reports it
	_ "github.com/toeirei/keymaster/connector/ssh"  // activate the ssh connector so ListConnectorKeys reports it
	"github.com/toeirei/keymaster/ui/i18n"
	"github.com/toeirei/keymaster/util/slicest"
)

type Client struct {
	// local temporary repository for testing ui features
	publicKeys   map[client.PublicKeyId]client.PublicKey
	accounts     map[client.AccountId]client.Account
	links        map[linkKey]client.Link
	auditLogs    []client.AuditLog
	remoteStates map[client.AccountId]string
	deployCaches map[client.AccountId]string

	// id counter to simulate serial
	publicKeyIdCounter client.PublicKeyId
	accountIdCounter   client.AccountId
	auditLogIdCounter  client.AuditLogId
}

// linkKey identifies a link by its (account, public key) pair.
type linkKey struct {
	accountId   client.AccountId
	publicKeyId client.PublicKeyId
}

// *[Client] implements [client.Client]
var _ client.Client = (*Client)(nil)

// --- utils ---

func (c *Client) writeAuditLog(action string, details client.AuditLogDetails) error {
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
		Id:        c.auditLogIdCounter,
		Timestamp: time.Now(),
		Metadata: client.AuditLogMetadata{
			Hostname: hostname,
			Hostuser: osuser.Username,
			Referer:  "testui",
		},
		Action:  action,
		Details: details,
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
	return fmt.Sprintf("%s %s@%s:%d\n%s", account.Connector, account.Username, account.Host, account.Port, deployCache)
}

// newSecret builds a connector secret from per-field values, using the connector
// that owns its shape. Accounts live in memory here, so the secret is held as
// the connector's type rather than serialized and parsed back.
func newSecret(connectorKey string, values map[string]string) (connector.Secret, error) {
	con, err := connector.Resolve(connectorKey)
	if err != nil {
		return nil, err
	}

	return con.ParseSecretFromFields(values)
}

// auditAccount renders an account for the audit log without its connector secret.
// Not every connector's secret redacts itself under %#v, which only consults
// fmt.Formatter and GoStringer, so drop it rather than trust the type.
func auditAccount(account client.Account) string {
	account.ConnectorSecret = nil
	return fmt.Sprintf("%#v", account)
}

// --- Lifecycle & Initialization ---

func NewClient() *Client {
	return &Client{
		publicKeys:   make(map[client.PublicKeyId]client.PublicKey),
		accounts:     make(map[client.AccountId]client.Account),
		links:        make(map[linkKey]client.Link),
		remoteStates: make(map[client.AccountId]string),
		deployCaches: make(map[client.AccountId]string),
	}
}

func (c *Client) Close(ctx context.Context) error {
	return nil
}

// NOT THREAD SAFE! ONLY FOR TESTING!
func (c *Client) WithTransaction(ctx context.Context, fn func(ctx context.Context, c client.Client) error) error {
	// create copy of client to use in transaction
	transactionClient := &Client{}
	if err := copier.Copy(transactionClient, c); err != nil {
		return err
	}

	// prepare cancelable context
	cctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	// run callback with transaction client
	err := fn(cctx, transactionClient)
	if err != nil {
		cancel(err)
		return err
	}

	// apply changes
	*c = *transactionClient
	return nil
}

// --- client.PublicKey Management ---

func (c *Client) CreatePublicKey(ctx context.Context, key string, comment string, isGlobal bool, expiresAt time.Time) (client.PublicKey, error) {
	c.publicKeyIdCounter++
	keyParts := strings.Split(key, " ")
	if len(keyParts) < 2 {
		return client.PublicKey{}, errors.New("invalid key provided")
	}
	// algorithm, data := keyParts[0], strings.Join(slicest.SliceTo(keyParts, 1, len(keyParts)), " ")
	algorithm, data := keyParts[0], keyParts[1]
	publicKey := client.PublicKey{Id: c.publicKeyIdCounter, Algorithm: algorithm, Data: data, Comment: comment, IsGlobal: isGlobal, ExpiresAt: expiresAt}
	c.publicKeys[publicKey.Id] = publicKey

	_ = c.writeAuditLog("public_key.create", client.AuditLogDetails{{"publicKey", fmt.Sprintf("%#v", publicKey)}})
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

func (c *Client) ListPublicKeys(ctx context.Context) ([]client.PublicKey, error) {
	return slicest.MapValues(c.publicKeys), nil
}

func (c *Client) ListPublicKeysLinkedToAccount(ctx context.Context, accountId client.AccountId, expired bool) ([]client.PublicKey, error) {
	links, err := c.ListLinksForAccount(ctx, accountId, expired)
	if err != nil {
		return nil, err
	}

	seen := make(map[client.PublicKeyId]struct{})
	var result []client.PublicKey

	// keys directly linked to the account
	for _, link := range links {
		if _, ok := seen[link.PublicKeyId]; ok {
			continue
		}
		publicKey, err := c.GetPublicKey(ctx, link.PublicKeyId)
		if err != nil {
			return nil, err
		}
		seen[publicKey.Id] = struct{}{}
		result = append(result, publicKey)
	}

	// global keys apply to every account, as long as the key has not expired
	for _, publicKey := range c.publicKeys {
		if !publicKey.IsGlobal {
			continue
		}
		if _, ok := seen[publicKey.Id]; ok {
			continue
		}
		if !publicKeyActive(publicKey, expired) {
			continue
		}
		seen[publicKey.Id] = struct{}{}
		result = append(result, publicKey)
	}

	return result, nil
}

// publicKeyActive reports whether a public key is currently valid: a zero
// ExpiresAt never expires, and the expired flag bypasses the check entirely.
func publicKeyActive(publicKey client.PublicKey, expired bool) bool {
	return expired || publicKey.ExpiresAt.IsZero() || time.Now().Before(publicKey.ExpiresAt)
}

func (c *Client) UpdateLink(ctx context.Context, accountId client.AccountId, publicKeyId client.PublicKeyId, expiresAt time.Time) (client.Link, error) {
	key := linkKey{accountId, publicKeyId}
	if link, ok := c.links[key]; ok {
		link.ExpiresAt = expiresAt
		c.links[key] = link

		_ = c.writeAuditLog("link.update", client.AuditLogDetails{{"link", fmt.Sprintf("%#v", link)}})
		return link, nil
	}
	return client.Link{}, fmt.Errorf("link not found: account %v, public key %v", accountId, publicKeyId)
}

func (c *Client) UpdatePublicKey(ctx context.Context, id client.PublicKeyId, comment string, isGlobal bool, expiresAt time.Time) (client.PublicKey, error) {
	if publicKey, ok := c.publicKeys[id]; ok {
		publicKey.Comment = comment
		publicKey.IsGlobal = isGlobal
		publicKey.ExpiresAt = expiresAt
		c.publicKeys[id] = publicKey

		_ = c.writeAuditLog("public_key.update", client.AuditLogDetails{{"publicKey", fmt.Sprintf("%#v", publicKey)}})
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
		_ = c.writeAuditLog("public_key.delete", client.AuditLogDetails{{"id", fmt.Sprintf("%#v", id)}})
	}

	return nil
}

// --- Account Management ---

func (c *Client) CreateAccount(ctx context.Context, username string, host string, port int, connectorKey string, connectorSecret map[string]string) (client.Account, error) {
	secret, err := newSecret(connectorKey, connectorSecret)
	if err != nil {
		return client.Account{}, err
	}

	c.accountIdCounter++
	account := client.Account{Id: c.accountIdCounter, Username: username, Host: host, Port: port, Connector: connectorKey, ConnectorSecret: secret}
	c.accounts[account.Id] = account

	_ = c.writeAuditLog("account.create", client.AuditLogDetails{{"account", auditAccount(account)}})
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
	// A global key reaches every account, as long as the key has not expired.
	publicKey, err := c.GetPublicKey(ctx, publicKeyId)
	if err != nil {
		return nil, err
	}
	if publicKey.IsGlobal {
		if !publicKeyActive(publicKey, expired) {
			return []client.Account{}, nil
		}
		return c.ListAccounts(ctx)
	}

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

func (c *Client) UpdateAccount(ctx context.Context, id client.AccountId, username string, host string, port int) (client.Account, error) {
	if account, ok := c.accounts[id]; ok {
		account.Username = username
		account.Host = host
		account.Port = port
		c.accounts[id] = account

		_ = c.writeAuditLog("account.update", client.AuditLogDetails{{"account", auditAccount(account)}})
		return account, nil
	}
	return client.Account{}, fmt.Errorf("account with id %v not found", id)
}

func (c *Client) UpdateAccountConnectorForce(ctx context.Context, id client.AccountId, connectorKey string, connectorSecret map[string]string) (client.Account, error) {
	secret, err := newSecret(connectorKey, connectorSecret)
	if err != nil {
		return client.Account{}, err
	}

	account, ok := c.accounts[id]
	if !ok {
		return client.Account{}, fmt.Errorf("account with id %v not found", id)
	}

	account.Connector = connectorKey
	account.ConnectorSecret = secret
	c.accounts[id] = account
	// Nothing has confirmed what the target holds, so drop the cache and let the
	// account read dirty until the next deploy or verify.
	delete(c.deployCaches, id)

	_ = c.writeAuditLog("account.update.connector_force", client.AuditLogDetails{{"account", auditAccount(account)}})
	return account, nil
}

func (c *Client) DeleteAccounts(ctx context.Context, ids ...client.AccountId) error {
	for _, id := range ids {
		if _, ok := c.accounts[id]; !ok {
			return fmt.Errorf("account with id %v not found", id)
		}
	}

	for _, id := range ids {
		delete(c.accounts, id)
		_ = c.writeAuditLog("account.delete", client.AuditLogDetails{{"id", fmt.Sprintf("%#v", id)}})
	}

	return nil
}

func (c *Client) IsAccountDirty(ctx context.Context, account client.Account) (bool, error) {
	deployData, err := c.accountDeployData(ctx, account)
	if err != nil {
		return true, err
	}

	return c.accountDeployCache(account, deployData) != c.deployCaches[account.Id], nil
}

// --- client.Link Management ---

func (c *Client) CreateLink(ctx context.Context, accountId client.AccountId, publicKeyId client.PublicKeyId, expiresAt time.Time) (client.Link, error) {
	link := client.Link{AccountId: accountId, PublicKeyId: publicKeyId, ExpiresAt: expiresAt}
	c.links[linkKey{accountId, publicKeyId}] = link

	_ = c.writeAuditLog("link.create", client.AuditLogDetails{{"link", fmt.Sprintf("%#v", link)}})
	return link, nil
}

func (c *Client) GetLink(ctx context.Context, accountId client.AccountId, publicKeyId client.PublicKeyId) (client.Link, error) {
	if link, ok := c.links[linkKey{accountId, publicKeyId}]; ok {
		return link, nil
	}
	return client.Link{}, fmt.Errorf("link not found: account %v, public key %v", accountId, publicKeyId)
}

func (c *Client) ListLinksForAccount(ctx context.Context, accountId client.AccountId, expired bool) ([]client.Link, error) {
	return slicest.Filter(slicest.MapValues(c.links), func(link client.Link) bool {
		return link.AccountId == accountId && (expired || link.ExpiresAt.IsZero() || time.Now().Before(link.ExpiresAt))
	}), nil
}

func (c *Client) ListLinksForPublicKey(ctx context.Context, publicKeyId client.PublicKeyId, expired bool) ([]client.Link, error) {
	return slicest.Filter(slicest.MapValues(c.links), func(link client.Link) bool {
		return link.PublicKeyId == publicKeyId && (expired || link.ExpiresAt.IsZero() || time.Now().Before(link.ExpiresAt))
	}), nil
}

func (c *Client) DeleteLink(ctx context.Context, accountId client.AccountId, publicKeyId client.PublicKeyId) error {
	key := linkKey{accountId, publicKeyId}
	if _, ok := c.links[key]; !ok {
		return fmt.Errorf("link not found: account %v, public key %v", accountId, publicKeyId)
	}

	delete(c.links, key)
	_ = c.writeAuditLog("link.delete", client.AuditLogDetails{{"key", fmt.Sprintf("%#v", key)}})

	return nil
}

// --- Deploy & Verify & Secret Update ---

// UpdateAccountSecret simulates the secret update: ticks, swaps the stored secret,
// then invalidates the cache the way a real confirmation would replace it. There
// is no rollback bookkeeping to keep because there is no target that could end up
// holding the wrong secret.
func (c *Client) UpdateAccountSecret(ctx context.Context, userRequester client.UserRequester, progress chan<- client.UpdateSecretProgressAccount, accountId client.AccountId, connectorSecret map[string]string) error {
	account, ok := c.accounts[accountId]
	if !ok {
		return fmt.Errorf("account with id %v not found", accountId)
	}

	secret, err := newSecret(account.Connector, connectorSecret)
	if err != nil {
		return err
	}

	for _, step := range []struct {
		fraction float64
		status   string
	}{
		{0.1, "client.status.connecting"},
		{0.5, "client.status.deploying"},
		{0.6, "client.status.confirming_secret"},
		{0.95, "client.status.verifying"},
	} {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		time.Sleep(time.Millisecond * 100)
		progress <- client.UpdateSecretProgressAccount{step.fraction, i18n.Text(step.status)}
	}

	account.ConnectorSecret = secret
	c.accounts[accountId] = account
	delete(c.deployCaches, accountId)

	progress <- client.UpdateSecretProgressAccount{1, i18n.Text("client.status.finished")}
	_ = c.writeAuditLog("account.update.secret", client.AuditLogDetails{{"account", auditAccount(account)}})
	return nil
}

func (c *Client) DeployAccount(ctx context.Context, userRequester client.UserRequester, progress chan<- client.DeployProgressAccount, accountId client.AccountId) error {
	return projectSingleAccount(accountId, progress, func(p chan<- client.ProgressAccounts) error {
		return c.DeployAccounts(ctx, userRequester, p, accountId)
	})
}

func (c *Client) DeployAccounts(ctx context.Context, userRequester client.UserRequester, progress chan<- client.DeployProgressAccounts, accountIds ...client.AccountId) error {
	accounts, err := c.GetAccounts(ctx, accountIds...)
	if err != nil {
		return err
	}

	deployDatas, err := slicest.MapX(accounts, func(a client.Account) (string, error) {
		return c.accountDeployData(ctx, a)
	})
	if err != nil {
		return err
	}

	deployProgress := client.DeployProgressAccounts{
		Accounts: slicest.ToMap(accounts, func(account client.Account) (client.AccountId, *client.ProgressAccountWithError) {
			return account.Id, &client.ProgressAccountWithError{ProgressAccount: client.ProgressAccount{0, i18n.Text("client.status.not_started")}}
		}),
	}

	checkContextCanceled := func(accountId client.AccountId, deployProgress client.DeployProgressAccounts) bool {
		if ctx.Err() != nil {
			deployProgress.Accounts[accountId].Progress = 1
			if errors.Is(ctx.Err(), context.Canceled) {
				deployProgress.Accounts[accountId].Status = i18n.Text("client.status.canceled")
				deployProgress.Accounts[accountId].Err = errors.New("canceled")
			} else {
				deployProgress.Accounts[accountId].Status = i18n.Text("client.status.error")
				deployProgress.Accounts[accountId].Err = ctx.Err()
			}
			return true
		}
		return false
	}

accountLoop:
	for i, account := range accounts {
		if checkContextCanceled(account.Id, deployProgress) {
			continue accountLoop
		}

		deployProgress.Accounts[account.Id].Status = i18n.Text("client.status.deploying")
		progress <- deployProgress

		// simulate deplay
		for _i := range 5 {
			time.Sleep(time.Millisecond * 100)
			if checkContextCanceled(account.Id, deployProgress) {
				continue accountLoop
			}
			deployProgress.Accounts[account.Id].Progress = float64(_i+1) / 10
			progress <- deployProgress
		}

		// potential error i guess
		ok := true
		if !ok {
			deployProgress.Accounts[account.Id].Status = i18n.Text("client.status.error")
			deployProgress.Accounts[account.Id].Progress = 1
			deployProgress.Accounts[account.Id].Err = fmt.Errorf("some weird error on account with id %v", account.Id)
			progress <- deployProgress
			continue
		}

		// simulate deplay
		for _i := range 5 {
			time.Sleep(time.Millisecond * 100)
			if checkContextCanceled(account.Id, deployProgress) {
				continue accountLoop
			}
			deployProgress.Accounts[account.Id].Progress = float64(_i+6) / 10
			progress <- deployProgress
		}

		// simulate deploying data to remote
		c.remoteStates[account.Id] = c.accountDeployCache(account, deployDatas[i])

		_ = c.writeAuditLog("account.deploy", client.AuditLogDetails{{"account", fmt.Sprintf("%#v", account)}})

		// update accounts deploy cache
		c.deployCaches[account.Id] = c.remoteStates[account.Id]

		deployProgress.Accounts[account.Id].Status = i18n.Text("client.status.finished")
		deployProgress.Accounts[account.Id].Progress = 1
		progress <- deployProgress
	}
	progress <- deployProgress

	return nil
}

func (c *Client) VerifyAccount(ctx context.Context, userRequester client.UserRequester, progress chan<- client.VerifyProgressAccount, accountId client.AccountId) error {
	return projectSingleAccount(accountId, progress, func(p chan<- client.ProgressAccounts) error {
		return c.VerifyAccounts(ctx, userRequester, p, accountId)
	})
}

func (c *Client) VerifyAccounts(ctx context.Context, userRequester client.UserRequester, progress chan<- client.VerifyProgressAccounts, accountIds ...client.AccountId) error {
	accounts, err := c.GetAccounts(ctx, accountIds...)
	if err != nil {
		return err
	}

	deployDatas, err := slicest.MapX(accounts, func(a client.Account) (string, error) {
		return c.accountDeployData(ctx, a)
	})
	if err != nil {
		return err
	}

	verifyProgress := client.VerifyProgressAccounts{
		Accounts: slicest.ToMap(accounts, func(account client.Account) (client.AccountId, *client.ProgressAccountWithError) {
			return account.Id, &client.ProgressAccountWithError{ProgressAccount: client.ProgressAccount{0, i18n.Text("client.status.not_started")}}
		}),
	}

	checkContextCanceled := func(accountId client.AccountId, deployProgress client.DeployProgressAccounts) bool {
		if ctx.Err() != nil {
			deployProgress.Accounts[accountId].Progress = 1
			if errors.Is(ctx.Err(), context.Canceled) {
				deployProgress.Accounts[accountId].Status = i18n.Text("client.status.canceled")
				deployProgress.Accounts[accountId].Err = errors.New("canceled")
			} else {
				deployProgress.Accounts[accountId].Status = i18n.Text("client.status.error")
				deployProgress.Accounts[accountId].Err = ctx.Err()
			}
			return true
		}
		return false
	}

accountLoop:
	for i, account := range accounts {
		if checkContextCanceled(account.Id, verifyProgress) {
			continue accountLoop
		}

		verifyProgress.Accounts[account.Id].Status = i18n.Text("client.status.verifying")
		progress <- verifyProgress

		// simulate deplay
		for _i := range 5 {
			time.Sleep(time.Millisecond * 100)
			if checkContextCanceled(account.Id, verifyProgress) {
				continue accountLoop
			}
			verifyProgress.Accounts[account.Id].Progress = float64(_i+1) / 10
			progress <- verifyProgress
		}

		ok := true
		if !ok {
			verifyProgress.Accounts[account.Id].Status = i18n.Text("client.status.error")
			verifyProgress.Accounts[account.Id].Progress = 1
			verifyProgress.Accounts[account.Id].Err = fmt.Errorf("some weird error on account with id %v", account.Id)
			progress <- verifyProgress
			continue
		}

		// simulate deplay
		for _i := range 5 {
			time.Sleep(time.Millisecond * 100)
			if checkContextCanceled(account.Id, verifyProgress) {
				continue accountLoop
			}
			verifyProgress.Accounts[account.Id].Progress = float64(_i+6) / 10
			progress <- verifyProgress
		}

		// simulate getting remoteState from remote
		remoteState, hasState := c.remoteStates[account.Id]

		if !hasState || c.accountDeployCache(account, deployDatas[i]) != remoteState {
			// update accounts deploy cache to reflect remotes state
			c.deployCaches[account.Id] = c.remoteStates[account.Id]

			verifyProgress.Accounts[account.Id].Status = i18n.Text("client.status.error")
			verifyProgress.Accounts[account.Id].Err = errors.New("account is out of sync")
		} else {
			verifyProgress.Accounts[account.Id].Status = i18n.Text("client.status.finished")
		}

		_ = c.writeAuditLog("account.verify", client.AuditLogDetails{{"account", fmt.Sprintf("%#v", account)}})

		verifyProgress.Accounts[account.Id].Progress = 1
		progress <- verifyProgress
	}
	progress <- verifyProgress

	return nil
}

// projectSingleAccount runs a batch operation for one account and projects the
// aggregate progress down onto that account's single-account channel, returning
// its terminal error joined with any operation-level error. The caller owns progress.
func projectSingleAccount(accountId client.AccountId, progress chan<- client.ProgressAccount, run func(chan<- client.ProgressAccounts) error) error {
	accountsProgress := make(chan client.ProgressAccounts)
	var opErr error
	go func() {
		defer close(accountsProgress)
		opErr = run(accountsProgress)
	}()

	var accountErr error
	for dp := range accountsProgress {
		if pa := dp.Accounts[accountId]; pa != nil {
			accountErr = pa.Err
			progress <- pa.ProgressAccount
		}
	}

	return errors.Join(opErr, accountErr)
}

// --- Other ---

func (c *Client) ListAuditLogs(ctx context.Context, offset int, limit int) ([]client.AuditLog, error) {
	// newest-first, mirroring the bunrewrite impl (ORDER BY timestamp DESC)
	logs := make([]client.AuditLog, len(c.auditLogs))
	for i, log := range c.auditLogs {
		logs[len(c.auditLogs)-1-i] = log
	}

	// skip offset entries from the front (newest), guarding bounds
	if offset > 0 {
		if offset >= len(logs) {
			return []client.AuditLog{}, nil
		}
		logs = logs[offset:]
	}

	// cap to limit
	if limit > 0 && limit < len(logs) {
		logs = logs[:limit]
	}

	return append([]client.AuditLog(nil), logs...), nil
}

func (c *Client) ListConnectorKeys(ctx context.Context) ([]string, error) {
	return connector.Keys(), nil
}

func (c *Client) ConnectorSecretFields(connectorKey string) ([]client.SecretField, error) {
	con, err := connector.Resolve(connectorKey)
	if err != nil {
		// An unchosen or unregistered connector has no fields, not an error.
		return nil, nil
	}

	return con.SecretFields(), nil
}

func (c *Client) OnboardHost(ctx context.Context, host string, port int /* , gateway string, plugin string */, accountUsername string, deploymentKey string) (chan client.OnboardHostProgress, error) {
	return nil, errors.New("client.OnboardHost not implemented")
}

func (c *Client) DecommisionAccount(ctx context.Context, id client.AccountId) (chan client.DecommisionAccountProgress, error) {
	return nil, errors.New("client.DecommisionAccount not implemented")
}
