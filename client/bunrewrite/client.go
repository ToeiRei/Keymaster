// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package bunrewrite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/bobg/go-generics/v4/slices"
	"github.com/toeirei/keymaster/client"
	"github.com/toeirei/keymaster/client/bunrewrite/db"
	"github.com/toeirei/keymaster/config"
	"github.com/toeirei/keymaster/connector"
	"github.com/toeirei/keymaster/core/sshkey"
	"github.com/toeirei/keymaster/util/slicest"
	"github.com/uptrace/bun"

	// for now, directly import/activate connectors here
	_ "github.com/toeirei/keymaster/connector/ssh"
)

type Client struct {
	config config.Config
	log    *log.Logger
	bun    bun.IDB
}

// *[Client] implements [client.Client]
var _ client.Client = (*Client)(nil)

func NewBunClient(config config.Config, logger *log.Logger) (*Client, error) {
	dbBun, err := db.Open(config.Database.Type, config.Database.Dsn)
	if err != nil {
		return nil, err
	}

	return &Client{config, logger, dbBun}, nil
}

func NewDefaultBunClient(logger *log.Logger) (*Client, error) {
	return NewBunClient(client.NewDefaultConfig(), logger)
}

func (c *Client) Close(ctx context.Context) error {
	if dbBun, ok := c.bun.(*bun.DB); ok {
		return dbBun.Close()
	}
	// transaction-scoped client: nothing to close.
	return nil
}

func (c *Client) WithTransaction(ctx context.Context, fn func(ctx context.Context, c client.Client) error) error {
	return c.bun.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		return fn(ctx, &Client{c.config, c.log, tx})
	})
}

// --- Helper functions ---

// nullTime converts a time.Time into a sql.NullTime, treating the zero value as
// NULL (i.e. "never expires").
func nullTime(t time.Time) sql.NullTime {
	return sql.NullTime{Time: t, Valid: !t.IsZero()}
}

// isExpired reports whether an expiry is still active at time now. A NULL/zero
// expiry never expires.
func isExpired(expiresAt sql.NullTime, now time.Time) bool {
	return expiresAt.Valid && now.After(expiresAt.Time)
}

// --- PublicKey Management ---

func modelToClientPublicKey(publicKeyModel db.PublicKeyModel) client.PublicKey {
	publicKey := client.PublicKey{
		Id:        client.PublicKeyId(publicKeyModel.ID),
		Algorithm: publicKeyModel.Algorithm,
		Data:      publicKeyModel.Data,
		Comment:   publicKeyModel.Comment,
		IsGlobal:  publicKeyModel.IsGlobal,
	}
	if publicKeyModel.ExpiresAt.Valid {
		publicKey.ExpiresAt = publicKeyModel.ExpiresAt.Time
	}
	return publicKey
}

func (c *Client) CreatePublicKey(ctx context.Context, key string, comment string, isGlobal bool, expiresAt time.Time) (client.PublicKey, error) {
	// Parse the key to extract algorithm and key data.
	alg, data, _, err := sshkey.Parse(key)
	if err != nil {
		return client.PublicKey{}, err
	}

	publicKeyModel := db.PublicKeyModel{
		Algorithm: alg,
		Data:      data,
		Comment:   comment,
		IsGlobal:  isGlobal,
		ExpiresAt: nullTime(expiresAt),
	}

	_, err = c.bun.NewInsert().
		Model(&publicKeyModel).
		Exec(ctx)
	if err != nil {
		return client.PublicKey{}, err
	}

	return modelToClientPublicKey(publicKeyModel), nil
}

func (c *Client) GetPublicKey(ctx context.Context, id client.PublicKeyId) (client.PublicKey, error) {
	publicKeyModel := db.PublicKeyModel{ID: int(id)}
	err := c.bun.NewSelect().
		Model(&publicKeyModel).
		WherePK().
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return client.PublicKey{}, fmt.Errorf("public key not found: %d", id)
	}
	if err != nil {
		return client.PublicKey{}, err
	}

	return modelToClientPublicKey(publicKeyModel), nil
}

func (c *Client) GetPublicKeys(ctx context.Context, ids ...client.PublicKeyId) ([]client.PublicKey, error) {
	var publicKeysModel []*db.PublicKeyModel
	err := c.bun.NewSelect().
		Model(&publicKeysModel).
		Where("id IN (?)", bun.In(ids)).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	publicKeys := slices.Map(publicKeysModel, func(publicKeyModel *db.PublicKeyModel) client.PublicKey {
		return modelToClientPublicKey(*publicKeyModel)
	})

	if len(publicKeys) != len(ids) {
		publicKeyIds := slices.Map(publicKeys, func(publicKey client.PublicKey) client.PublicKeyId { return publicKey.Id })
		missingIds := slices.Map(slices.Filter(ids, func(id client.PublicKeyId) bool {
			return !slices.Contains(publicKeyIds, id)
		}), func(id client.PublicKeyId) string { return fmt.Sprint(id) })
		return nil, fmt.Errorf("public keys with ids could not be found: %s", strings.Join(missingIds, ", "))
	}

	return publicKeys, nil
}

func (c *Client) ListPublicKeys(ctx context.Context) ([]client.PublicKey, error) {
	var publicKeysModel []*db.PublicKeyModel
	err := c.bun.NewSelect().
		Model(&publicKeysModel).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return slices.Map(publicKeysModel, func(publicKeyModel *db.PublicKeyModel) client.PublicKey {
		return modelToClientPublicKey(*publicKeyModel)
	}), nil
}

func (c *Client) ListPublicKeysLinkedToAccount(ctx context.Context, accountId client.AccountId, expired bool) ([]client.PublicKey, error) {
	now := time.Now()

	// Resolve the account's links. Unless expired links were requested, a link
	// only counts while active (a NULL expires_at never expires).
	var linkModels []db.LinkModel
	linkQuery := c.bun.NewSelect().
		Model(&linkModels).
		Where("account_id = ?", int(accountId))
	if !expired {
		linkQuery = linkQuery.Where("(expires_at IS NULL OR expires_at > ?)", now)
	}
	if err := linkQuery.Scan(ctx); err != nil {
		return nil, err
	}

	publicKeyIds := slices.Map(linkModels, func(link db.LinkModel) int { return link.PublicKeyId })

	// Fetch the linked keys plus all global keys. A global key still respects
	// its own expiry (a NULL expires_at never expires); linked keys are included
	// regardless of their own expiry because the link's expiry already gates them.
	var publicKeysModel []*db.PublicKeyModel
	query := c.bun.NewSelect().Model(&publicKeysModel).
		WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			q = q.Where("is_global = ?", true)
			if !expired {
				q = q.Where("(expires_at IS NULL OR expires_at > ?)", now)
			}
			return q
		})

	if len(publicKeyIds) > 0 {
		query = query.WhereOr("id IN (?)", bun.List(publicKeyIds))
	}

	if err := query.Scan(ctx); err != nil {
		return nil, err
	}

	return slices.Map(publicKeysModel, func(publicKeyModel *db.PublicKeyModel) client.PublicKey {
		return modelToClientPublicKey(*publicKeyModel)
	}), nil
}

func (c *Client) UpdatePublicKey(ctx context.Context, id client.PublicKeyId, comment string, isGlobal bool, expiresAt time.Time) (client.PublicKey, error) {
	publicKeyModel := db.PublicKeyModel{
		ID:        int(id),
		Comment:   comment,
		IsGlobal:  isGlobal,
		ExpiresAt: nullTime(expiresAt),
	}

	res, err := c.bun.NewUpdate().
		Model(&publicKeyModel).
		Column("comment", "is_global", "expires_at").
		WherePK().
		Exec(ctx)
	if err != nil {
		return client.PublicKey{}, err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return client.PublicKey{}, err
	}
	if rowsAffected == 0 {
		return client.PublicKey{}, fmt.Errorf("public key not found: %d", id)
	}

	// re-read to return the full, current row
	return c.GetPublicKey(ctx, id)
}

func (c *Client) DeletePublicKeys(ctx context.Context, ids ...client.PublicKeyId) error {
	return c.bun.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		res, err := tx.NewDelete().
			Model((*db.PublicKeyModel)(nil)).
			Where("id IN (?)", bun.In(ids)).
			Exec(ctx)
		if err != nil {
			return err
		}

		rowsAffected, err := res.RowsAffected()
		if err != nil {
			return err
		}

		unaffectedRows := int64(len(ids)) - rowsAffected
		if unaffectedRows > 0 {
			return fmt.Errorf("%d public key ids could not be found", unaffectedRows)
		}

		return nil
	})
}

// --- Account Management ---

func modelToClientAccount(accountModel db.AccountModel) client.Account {
	port, _ := strconv.Atoi(accountModel.Port)
	return client.Account{
		Id:           client.AccountId(accountModel.ID),
		Username:     accountModel.Username,
		Host:         accountModel.Host,
		Port:         port,
		DeployMethod: accountModel.DeployMethod,
		DeploySecret: accountModel.DeploySecret,
		DeployCache:  "",
	}
}

func (c *Client) CreateAccount(ctx context.Context, username string, host string, port int, deploymentMethod string, deploymentSecret string) (client.Account, error) {
	accountModel := db.AccountModel{
		Username:     username,
		Host:         host,
		Port:         strconv.Itoa(port),
		IsActive:     true,
		IsDirty:      true,
		DeployMethod: deploymentMethod,
		DeploySecret: deploymentSecret,
	}

	_, err := c.bun.NewInsert().
		Model(&accountModel).
		Exec(ctx)
	if err != nil {
		return client.Account{}, err
	}

	return modelToClientAccount(accountModel), nil
}

func (c *Client) GetAccount(ctx context.Context, id client.AccountId) (client.Account, error) {
	accountModel := db.AccountModel{ID: int(id)}
	err := c.bun.NewSelect().
		Model(&accountModel).
		WherePK().
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return client.Account{}, fmt.Errorf("account not found: %d", id)
	}
	if err != nil {
		return client.Account{}, err
	}

	return modelToClientAccount(accountModel), nil
}

func (c *Client) GetAccounts(ctx context.Context, ids ...client.AccountId) ([]client.Account, error) {
	var accountModels []*db.AccountModel
	err := c.bun.NewSelect().
		Model(&accountModels).
		Where("id IN (?)", bun.In(ids)).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	accounts := slices.Map(accountModels, func(accountModel *db.AccountModel) client.Account {
		return modelToClientAccount(*accountModel)
	})

	if len(accounts) != len(ids) {
		accountIds := slices.Map(accounts, func(account client.Account) client.AccountId { return account.Id })
		missingIds := slices.Map(slices.Filter(ids, func(id client.AccountId) bool {
			return !slices.Contains(accountIds, id)
		}), func(id client.AccountId) string { return fmt.Sprint(id) })
		return nil, fmt.Errorf("accounts with ids could not be found: %s", strings.Join(missingIds, ", "))
	}

	return accounts, nil
}

func (c *Client) ListAccounts(ctx context.Context) ([]client.Account, error) {
	var accountModels []*db.AccountModel
	err := c.bun.NewSelect().
		Model(&accountModels).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return slices.Map(accountModels, func(accountModel *db.AccountModel) client.Account {
		return modelToClientAccount(*accountModel)
	}), nil
}

func (c *Client) ListAccountsDirty(ctx context.Context) ([]client.Account, error) {
	// var accountModels []*db.AccountModel
	// err := c.bun.NewSelect().
	// 	Model(&accountModels).
	// 	Where("is_dirty = ?", true).
	// 	Scan(ctx)
	// if err != nil {
	// 	return nil, err
	// }

	accounts, err := c.ListAccounts(ctx)
	if err != nil {
		return nil, err
	}

	return slices.Filter(accounts, func(account client.Account) bool {
		dirty, err := c.IsAccountDirty(ctx, account)
		return dirty || err != nil
	}), nil
}

func (c *Client) ListAccountsLinkedToPublicKey(ctx context.Context, publicKeyId client.PublicKeyId, expired bool) ([]client.Account, error) {
	// A global key applies to every account, as long as the key has not expired.
	publicKey, err := c.GetPublicKey(ctx, publicKeyId)
	if err != nil {
		return nil, err
	}
	if publicKey.IsGlobal {
		if expired || !isExpired(nullTime(publicKey.ExpiresAt), time.Now()) {
			return c.ListAccounts(ctx)
		}
		return []client.Account{}, nil
	}

	var linkModels []db.LinkModel
	err = c.bun.NewSelect().
		Model(&linkModels).
		Where("public_key_id = ?", int(publicKeyId)).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	accountIds := slices.Map(
		slices.Filter(linkModels, func(link db.LinkModel) bool {
			return expired || !isExpired(link.ExpiresAt, now)
		}),
		func(link db.LinkModel) client.AccountId { return client.AccountId(link.AccountId) },
	)

	if len(accountIds) == 0 {
		return []client.Account{}, nil
	}

	return c.GetAccounts(ctx, accountIds...)
}

func (c *Client) UpdateAccount(ctx context.Context, id client.AccountId, username string, host string, port int, deploymentMethod string, deploymentSecret string) (client.Account, error) {
	accountModel := db.AccountModel{
		ID:           int(id),
		Username:     username,
		Host:         host,
		Port:         strconv.Itoa(port),
		DeployMethod: deploymentMethod,
		DeploySecret: deploymentSecret,
	}

	err := c.bun.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// update account
		_, err := tx.NewUpdate().
			Model(&accountModel).
			Column("username", "host", "port", "deploy_method", "deploy_secret").
			WherePK().
			Exec(ctx)
		if err != nil {
			return err
		}

		err = tx.NewSelect().
			Model(&accountModel).
			WherePK().
			Scan(ctx)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return client.Account{}, err
	}

	return modelToClientAccount(accountModel), nil
}

func (c *Client) DeleteAccounts(ctx context.Context, ids ...client.AccountId) error {
	return c.bun.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		res, err := tx.NewDelete().
			Model((*db.AccountModel)(nil)).
			Where("id IN (?)", bun.In(ids)).
			Exec(ctx)
		if err != nil {
			return err
		}

		rowsAffected, err := res.RowsAffected()
		if err != nil {
			return err
		}

		unaffectedRows := int64(len(ids)) - rowsAffected
		if unaffectedRows > 0 {
			return fmt.Errorf("%d account ids could not be found", unaffectedRows)
		}

		return nil
	})
}

func (c *Client) IsAccountDirty(ctx context.Context, account client.Account) (bool, error) {
	con, err := connector.Resolve(account.DeployMethod)
	if err != nil {
		return true, err
	}

	deployData, err := c.accountDeployData(ctx, account)
	if err != nil {
		return true, err
	}

	valid, err := con.VerifyOffline(ctx, deployData)
	if err != nil {
		return true, err
	}

	return !valid, nil
}

// --- Link Management ---

func modelToClientLink(linkModel db.LinkModel) client.Link {
	link := client.Link{
		AccountId:   client.AccountId(linkModel.AccountId),
		PublicKeyId: client.PublicKeyId(linkModel.PublicKeyId),
	}
	if linkModel.ExpiresAt.Valid {
		link.ExpiresAt = linkModel.ExpiresAt.Time
	}
	return link
}

func (c *Client) CreateLink(ctx context.Context, accountId client.AccountId, publicKeyId client.PublicKeyId, expiresAt time.Time) (client.Link, error) {
	linkModel := db.LinkModel{
		AccountId:   int(accountId),
		PublicKeyId: int(publicKeyId),
		ExpiresAt:   nullTime(expiresAt),
	}

	_, err := c.bun.NewInsert().
		Model(&linkModel).
		Exec(ctx)
	if err != nil {
		return client.Link{}, err
	}

	return modelToClientLink(linkModel), nil
}

func (c *Client) GetLink(ctx context.Context, accountId client.AccountId, publicKeyId client.PublicKeyId) (client.Link, error) {
	linkModel := db.LinkModel{AccountId: int(accountId), PublicKeyId: int(publicKeyId)}
	err := c.bun.NewSelect().
		Model(&linkModel).
		WherePK().
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return client.Link{}, fmt.Errorf("link not found: account %d, public key %d", accountId, publicKeyId)
	}
	if err != nil {
		return client.Link{}, err
	}

	return modelToClientLink(linkModel), nil
}

func (c *Client) ListLinksForAccount(ctx context.Context, accountId client.AccountId, expired bool) ([]client.Link, error) {
	var linkModels []*db.LinkModel
	err := c.bun.NewSelect().
		Model(&linkModels).
		Where("account_id = ?", int(accountId)).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	return slices.Map(
		slices.Filter(linkModels, func(link *db.LinkModel) bool {
			return expired || !isExpired(link.ExpiresAt, now)
		}),
		func(linkModel *db.LinkModel) client.Link {
			return modelToClientLink(*linkModel)
		},
	), nil
}

func (c *Client) ListLinksForPublicKey(ctx context.Context, publicKeyId client.PublicKeyId, expired bool) ([]client.Link, error) {
	var linkModels []*db.LinkModel
	err := c.bun.NewSelect().
		Model(&linkModels).
		Where("public_key_id = ?", int(publicKeyId)).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	return slices.Map(
		slices.Filter(linkModels, func(link *db.LinkModel) bool {
			return expired || !isExpired(link.ExpiresAt, now)
		}),
		func(linkModel *db.LinkModel) client.Link {
			return modelToClientLink(*linkModel)
		},
	), nil
}

func (c *Client) UpdateLink(ctx context.Context, accountId client.AccountId, publicKeyId client.PublicKeyId, expiresAt time.Time) (client.Link, error) {
	linkModel := db.LinkModel{
		AccountId:   int(accountId),
		PublicKeyId: int(publicKeyId),
		ExpiresAt:   nullTime(expiresAt),
	}

	// account_id and public_key_id form the primary key, so only expiry is mutable.
	res, err := c.bun.NewUpdate().
		Model(&linkModel).
		Column("expires_at").
		WherePK().
		Exec(ctx)
	if err != nil {
		return client.Link{}, err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return client.Link{}, err
	}
	if rowsAffected == 0 {
		return client.Link{}, fmt.Errorf("link not found: account %d, public key %d", accountId, publicKeyId)
	}

	return modelToClientLink(linkModel), nil
}

func (c *Client) DeleteLink(ctx context.Context, accountId client.AccountId, publicKeyId client.PublicKeyId) error {
	res, err := c.bun.NewDelete().
		Model((*db.LinkModel)(nil)).
		Where("account_id = ?", int(accountId)).
		Where("public_key_id = ?", int(publicKeyId)).
		Exec(ctx)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("link not found: account %d, public key %d", accountId, publicKeyId)
	}

	return nil
}

// --- Deploy & Verify ---

// accountDeployData assembles the connector deploy data for an account: its
// currently active linked public keys plus the account's deploy secret and
// cache. A connector uses this to render (and fingerprint) the account's
// authorized_keys.
func (c *Client) accountDeployData(ctx context.Context, account client.Account) (connector.DeployData, error) {
	publicKeys, err := c.ListPublicKeysLinkedToAccount(ctx, account.Id, false)
	if err != nil {
		return connector.DeployData{}, err
	}

	return connector.DeployData{
		Records: slicest.Map(publicKeys, func(publicKey client.PublicKey) connector.DeployRecord {
			return connector.DeployRecord{
				Algorithm: publicKey.Algorithm,
				Data:      publicKey.Data,
				Comment:   publicKey.Comment,
				IsGlobal:  publicKey.IsGlobal,
				ExpiresAt: publicKey.ExpiresAt,
			}
		}),
		Secret: account.DeploySecret,
		Cache:  account.DeployCache,
	}, nil
}

// accountConnectionData describes how to reach the account's host.
func accountConnectionData(account client.Account) connector.ConnectionData {
	return connector.ConnectionData{
		Username: account.Username,
		Host:     account.Host,
		Port:     account.Port,
	}
}

// connectorOperation is the shared shape of Connector.Deploy and
// Connector.Verify, letting DeployAccounts and VerifyAccounts reuse the same
// per-account streaming logic.
type connectorOperation func(ctx context.Context, deployData connector.DeployData, connectionData connector.ConnectionData) (chan connector.Progress, error)

// runAccounts resolves each account's connector, runs the given operation
// (deploy or verify) sequentially, and forwards every connector progress update
// into a shared aggregate snapshot sent on the returned channel. The channel is
// closed once all accounts have been processed.
func (c *Client) runAccounts(ctx context.Context, selectOp func(connector.Connector) connectorOperation, accountIds ...client.AccountId) (chan client.DeployProgressAccounts, error) {
	accounts, err := c.GetAccounts(ctx, accountIds...)
	if err != nil {
		return nil, err
	}

	progress := client.DeployProgressAccounts{
		Accounts: slicest.ToMap(accounts, func(account client.Account) (client.AccountId, *client.DeployProgressAccount) {
			return account.Id, &client.DeployProgressAccount{Progress: 0, Status: "not started", Err: nil}
		}),
	}

	progressChan := make(chan client.DeployProgressAccounts)
	go func() {
		defer close(progressChan)

		for _, account := range accounts {
			c.runAccount(ctx, account, selectOp, progress, progressChan)
		}

		// final snapshot so consumers always observe the terminal state
		progressChan <- progress
	}()

	return progressChan, nil
}

// runAccount performs the connector operation for a single account, streaming
// its progress into the shared aggregate. Any failure to reach the connector is
// reported as an error status on that account rather than aborting the batch.
func (c *Client) runAccount(ctx context.Context, account client.Account, selectOp func(connector.Connector) connectorOperation, progress client.DeployProgressAccounts, progressChan chan client.DeployProgressAccounts) {
	accountProgress := progress.Accounts[account.Id]

	fail := func(err error) {
		accountProgress.Status = "error"
		accountProgress.Progress = 1
		accountProgress.Err = err
		progressChan <- progress
	}

	con, err := connector.Resolve(account.DeployMethod)
	if err != nil {
		fail(err)
		return
	}

	deployData, err := c.accountDeployData(ctx, account)
	if err != nil {
		fail(err)
		return
	}

	connectorProgress, err := selectOp(con)(ctx, deployData, accountConnectionData(account))
	if err != nil {
		fail(err)
		return
	}

	for cp := range connectorProgress {
		accountProgress.Progress = cp.Progress
		accountProgress.Status = cp.Status
		accountProgress.Err = cp.Err
		progressChan <- progress
	}
}

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
	return c.runAccounts(ctx, func(con connector.Connector) connectorOperation {
		return con.Deploy
	}, accountIds...)
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
	return c.runAccounts(ctx, func(con connector.Connector) connectorOperation {
		return con.Verify
	}, accountIds...)
}

// --- Other Operations ---

func modelToClientAuditLog(auditLogModel db.AuditLogModel) client.AuditLog {
	return client.AuditLog{
		Id:        client.AuditLogId(auditLogModel.ID),
		Timestamp: auditLogModel.Timestamp,
		Metadata: client.AuditLogMetadata{
			Hostname: auditLogModel.Hostname.String,
			Hostuser: auditLogModel.Username,
			Referer:  auditLogModel.Referrer.String,
		},
		Action:  auditLogModel.Action,
		Details: auditLogModel.Details,
	}
}

func (c *Client) ListAuditLogs(ctx context.Context, limit int) ([]client.AuditLog, error) {
	var auditLogModels []*db.AuditLogModel
	query := c.bun.NewSelect().
		Model(&auditLogModels).
		OrderExpr("timestamp DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Scan(ctx); err != nil {
		return nil, err
	}

	return slices.Map(auditLogModels, func(auditLogModel *db.AuditLogModel) client.AuditLog {
		return modelToClientAuditLog(*auditLogModel)
	}), nil
}

func (c *Client) OnboardHost(ctx context.Context, host string, port int, accountUsername string, deploymentKey string) (chan client.OnboardHostProgress, error) {
	panic("not planned")
}

func (c *Client) DecommisionAccount(ctx context.Context, id client.AccountId) (chan client.DecommisionAccountProgress, error) {
	panic("not planned")
}
