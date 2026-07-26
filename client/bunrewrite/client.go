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
	"os"
	"os/user"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bobg/go-generics/v4/slices"
	"github.com/toeirei/keymaster/client"
	"github.com/toeirei/keymaster/client/bunrewrite/db"
	"github.com/toeirei/keymaster/config"
	"github.com/toeirei/keymaster/connector"
	"github.com/toeirei/keymaster/core/sshkey"
	"github.com/toeirei/keymaster/ui/i18n"
	"github.com/toeirei/keymaster/util/slicest"
	"github.com/uptrace/bun"

	// for now, directly import/activate connectors here
	_ "github.com/toeirei/keymaster/connector/mock"
	_ "github.com/toeirei/keymaster/connector/ssh"
)

type Client struct {
	config config.Config
	log    *log.Logger
	bun    bun.IDB
	// referer names the frontend implementation;
	// it is stamped onto every audit log entry this client writes.
	referer string
}

// *[Client] implements [client.Client]
var _ client.Client = (*Client)(nil)

func NewBunClient(config config.Config, logger *log.Logger, referer string) (*Client, error) {
	dbBun, err := db.Open(config.Database.Type, config.Database.Dsn)
	if err != nil {
		return nil, err
	}

	return &Client{config, logger, dbBun, referer}, nil
}

func NewDefaultBunClient(logger *log.Logger, referer string) (*Client, error) {
	return NewBunClient(client.NewDefaultConfig(), logger, referer)
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
		return fn(ctx, &Client{c.config, c.log, tx, c.referer})
	})
}

// --- Helper functions ---

// nullTime converts a time.Time into a sql.NullTime, treating the zero value as
// NULL (i.e. "never expires").
func nullTime(t time.Time) sql.NullTime {
	return sql.NullTime{Time: t, Valid: !t.IsZero()}
}

// nullString converts a string into a sql.NullString, treating the empty string
// as NULL.
func nullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

// --- Audit logging ---

// auditTime renders an expiry for audit details, mapping the zero value (never
// expires) to "never" instead of a noisy zero timestamp.
func auditTime(t time.Time) string {
	if t.IsZero() {
		return "never"
	}
	return t.Format(time.RFC3339)
}

// auditSecret redacts a deploy secret for audit details: it records only whether
// any secret material was provided, never the values themselves (mirroring
// Account.String()). It reads the per-field values rather than the serialized
// JSON, which is never empty even for an unset secret.
func auditSecret(values map[string]string) string {
	for _, value := range values {
		if value != "" {
			return "<redacted>"
		}
	}
	return ""
}

// auditIds renders a list of numeric ids as a comma-separated string.
func auditIds[T ~int](ids []T) string {
	strs := make([]string, len(ids))
	for i, id := range ids {
		strs[i] = strconv.Itoa(int(id))
	}
	return strings.Join(strs, ", ")
}

// accountOpAuditDetails builds the audit details describing the outcome of a
// deploy/verify operation against a single account. opErr is nil on success.
func accountOpAuditDetails(account client.Account, keyCount int, opErr error) client.AuditLogDetails {
	details := client.AuditLogDetails{
		{"accountId", strconv.Itoa(int(account.Id))},
		{"connection", fmt.Sprintf("%s@%s:%d", account.Username, account.Host, account.Port)},
		{"deployMethod", account.DeployMethod},
	}
	if opErr != nil {
		return append(details,
			client.AuditLogDetail{"result", "error"},
			client.AuditLogDetail{"error", opErr.Error()},
		)
	}
	return append(details,
		client.AuditLogDetail{"result", "success"},
		client.AuditLogDetail{"keyCount", strconv.Itoa(keyCount)},
	)
}

// writeAuditLog records a single audit log entry through the given handle.
// Callers pass the transaction (tx) from their surrounding RunInTx so the audit
// entry is committed — or rolled back — atomically with the change it documents.
func (c *Client) writeAuditLog(ctx context.Context, idb bun.IDB, action string, details client.AuditLogDetails) error {
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}

	osUser, err := user.Current()
	if err != nil {
		return err
	}

	username := osUser.Username
	if runtime.GOOS == "windows" {
		usernameParts := strings.Split(username, "\\")
		username = usernameParts[len(usernameParts)-1]
	}

	auditLogModel := db.AuditLogModel{
		Timestamp: time.Now(),
		Username:  username,
		Hostname:  nullString(hostname),
		Referrer:  nullString(c.referer),
		Action:    action,
		Details:   details,
	}

	_, err = idb.NewInsert().
		Model(&auditLogModel).
		Exec(ctx)
	return err
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

	err = c.bun.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewInsert().
			Model(&publicKeyModel).
			Exec(ctx); err != nil {
			return err
		}
		return c.writeAuditLog(ctx, tx, "public_key.create", client.AuditLogDetails{
			{"id", strconv.Itoa(publicKeyModel.ID)},
			{"algorithm", alg},
			{"data", data},
			{"comment", comment},
			{"isGlobal", strconv.FormatBool(isGlobal)},
			{"expiresAt", auditTime(expiresAt)},
		})
	})
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
		return client.PublicKey{}, i18n.NewError("errors.client.public_key_not_found", id)
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
		Where("id IN (?)", bun.List(ids)).
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
		return nil, i18n.NewError("errors.client.public_keys_not_found", strings.Join(missingIds, ", "))
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

	var updated client.PublicKey
	err := c.bun.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		res, err := tx.NewUpdate().
			Model(&publicKeyModel).
			Column("comment", "is_global", "expires_at").
			WherePK().
			Exec(ctx)
		if err != nil {
			return err
		}

		rowsAffected, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if rowsAffected == 0 {
			return i18n.NewError("errors.client.public_key_not_found", id)
		}

		// re-read within the transaction to return the full, current row
		updatedModel := db.PublicKeyModel{ID: int(id)}
		if err := tx.NewSelect().
			Model(&updatedModel).
			WherePK().
			Scan(ctx); err != nil {
			return err
		}
		updated = modelToClientPublicKey(updatedModel)

		return c.writeAuditLog(ctx, tx, "public_key.update", client.AuditLogDetails{
			{"id", strconv.Itoa(int(id))},
			{"comment", comment},
			{"isGlobal", strconv.FormatBool(isGlobal)},
			{"expiresAt", auditTime(expiresAt)},
		})
	})
	if err != nil {
		return client.PublicKey{}, err
	}

	return updated, nil
}

func (c *Client) DeletePublicKeys(ctx context.Context, ids ...client.PublicKeyId) error {
	return c.bun.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		res, err := tx.NewDelete().
			Model((*db.PublicKeyModel)(nil)).
			Where("id IN (?)", bun.List(ids)).
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
			return i18n.NewError("errors.client.public_keys_delete_missing", unaffectedRows)
		}

		return c.writeAuditLog(ctx, tx, "public_key.delete", client.AuditLogDetails{
			{"ids", auditIds(ids)},
		})
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
		Serial:       accountModel.Serial,
		DeployMethod: accountModel.DeployMethod,
		DeploySecret: accountModel.DeploySecret,
		DeployCache:  accountModel.DeployCache,
	}
}

// serializeSecret turns the per-field values a UI collected into the JSON the
// deploy_secret column stores, using the connector that owns its shape.
func serializeSecret(deploymentMethod string, values map[string]string) (string, error) {
	con, err := connector.Resolve(deploymentMethod)
	if err != nil {
		return "", err
	}
	secret, err := con.NewSecretFromValues(values)
	if err != nil {
		return "", err
	}
	return secret.Serialize()
}

func (c *Client) CreateAccount(ctx context.Context, username string, host string, port int, deploymentMethod string, deploymentSecret map[string]string) (client.Account, error) {
	serializedSecret, err := serializeSecret(deploymentMethod, deploymentSecret)
	if err != nil {
		return client.Account{}, err
	}

	accountModel := db.AccountModel{
		Username:     username,
		Host:         host,
		Port:         strconv.Itoa(port),
		IsActive:     true,
		IsDirty:      true,
		DeployMethod: deploymentMethod,
		DeploySecret: serializedSecret,
	}

	err = c.bun.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewInsert().
			Model(&accountModel).
			Exec(ctx); err != nil {
			return err
		}
		return c.writeAuditLog(ctx, tx, "account.create", client.AuditLogDetails{
			{"id", strconv.Itoa(accountModel.ID)},
			{"username", username},
			{"host", host},
			{"port", strconv.Itoa(port)},
			{"deployMethod", deploymentMethod},
			{"deploySecret", auditSecret(deploymentSecret)},
		})
	})
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
		return client.Account{}, i18n.NewError("errors.client.account_not_found", id)
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
		Where("id IN (?)", bun.List(ids)).
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
		return nil, i18n.NewError("errors.client.accounts_not_found", strings.Join(missingIds, ", "))
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

func (c *Client) UpdateAccount(ctx context.Context, id client.AccountId, username string, host string, port int, deploymentMethod string, deploymentSecret map[string]string) (client.Account, error) {
	serializedSecret, err := serializeSecret(deploymentMethod, deploymentSecret)
	if err != nil {
		return client.Account{}, err
	}

	accountModel := db.AccountModel{
		ID:           int(id),
		Username:     username,
		Host:         host,
		Port:         strconv.Itoa(port),
		DeployMethod: deploymentMethod,
		DeploySecret: serializedSecret,
	}

	err = c.bun.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
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

		return c.writeAuditLog(ctx, tx, "account.update", client.AuditLogDetails{
			{"id", strconv.Itoa(int(id))},
			{"username", username},
			{"host", host},
			{"port", strconv.Itoa(port)},
			{"deployMethod", deploymentMethod},
			{"deploySecret", auditSecret(deploymentSecret)},
		})
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
			Where("id IN (?)", bun.List(ids)).
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
			return i18n.NewError("errors.client.accounts_delete_missing", unaffectedRows)
		}

		return c.writeAuditLog(ctx, tx, "account.delete", client.AuditLogDetails{
			{"ids", auditIds(ids)},
		})
	})
}

func (c *Client) IsAccountDirty(ctx context.Context, account client.Account) (bool, error) {
	con, err := connector.Resolve(account.DeployMethod)
	if err != nil {
		return true, err
	}

	deployData, err := c.accountDeployData(ctx, con, account)
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

	err := c.bun.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewInsert().
			Model(&linkModel).
			Exec(ctx); err != nil {
			return err
		}
		return c.writeAuditLog(ctx, tx, "link.create", client.AuditLogDetails{
			{"accountId", strconv.Itoa(int(accountId))},
			{"publicKeyId", strconv.Itoa(int(publicKeyId))},
			{"expiresAt", auditTime(expiresAt)},
		})
	})
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
		return client.Link{}, i18n.NewError("errors.client.link_not_found", accountId, publicKeyId)
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

	err := c.bun.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// account_id and public_key_id form the primary key, so only expiry is mutable.
		res, err := tx.NewUpdate().
			Model(&linkModel).
			Column("expires_at").
			WherePK().
			Exec(ctx)
		if err != nil {
			return err
		}

		rowsAffected, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if rowsAffected == 0 {
			return i18n.NewError("errors.client.link_not_found", accountId, publicKeyId)
		}

		return c.writeAuditLog(ctx, tx, "link.update", client.AuditLogDetails{
			{"accountId", strconv.Itoa(int(accountId))},
			{"publicKeyId", strconv.Itoa(int(publicKeyId))},
			{"expiresAt", auditTime(expiresAt)},
		})
	})
	if err != nil {
		return client.Link{}, err
	}

	return modelToClientLink(linkModel), nil
}

func (c *Client) DeleteLink(ctx context.Context, accountId client.AccountId, publicKeyId client.PublicKeyId) error {
	return c.bun.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		res, err := tx.NewDelete().
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
			return i18n.NewError("errors.client.link_not_found", accountId, publicKeyId)
		}

		return c.writeAuditLog(ctx, tx, "link.delete", client.AuditLogDetails{
			{"accountId", strconv.Itoa(int(accountId))},
			{"publicKeyId", strconv.Itoa(int(publicKeyId))},
		})
	})
}

// --- Deploy & Verify ---

func (c *Client) accountDeployData(ctx context.Context, con connector.Connector, account client.Account) (connector.DeployData, error) {
	secret, err := con.NewSecret(account.DeploySecret)
	if err != nil {
		return connector.DeployData{}, err
	}

	cache, err := con.NewCache(account.DeployCache)
	if err != nil {
		return connector.DeployData{}, err
	}

	now := time.Now()

	var linkModels []db.LinkModel
	query := c.bun.NewSelect().
		Model(&linkModels).
		Relation("PublicKey").
		// WHERE links
		Where("link_model.account_id = ?", int(account.Id)).
		Where("(link_model.expires_at IS NULL OR link_model.expires_at > ?)", now).
		// WHERE public_keys
		Where("public_key.id IS NOT NULL"). // converts relation to inner join
		Where("(public_key.expires_at IS NULL OR public_key.expires_at > ?)", now).
		Where("public_key.is_global = ?", false)

	queryStr := query.String()
	_ = queryStr

	err = query.Scan(ctx)
	if err != nil {
		return connector.DeployData{}, err
	}

	var globalPublicKeyModels []db.PublicKeyModel
	err = c.bun.NewSelect().
		Model(&globalPublicKeyModels).
		Where("(expires_at IS NULL OR expires_at > ?)", now).
		Where("is_global = ?", true).
		Scan(ctx)
	if err != nil {
		return connector.DeployData{}, err
	}

	globalRecords := slicest.Map(globalPublicKeyModels, func(publicKeyModel db.PublicKeyModel) connector.DeployRecord {
		var expiresAt time.Time
		if publicKeyModel.ExpiresAt.Valid {
			expiresAt = publicKeyModel.ExpiresAt.Time
		}
		return connector.DeployRecord{
			publicKeyModel.Algorithm,
			publicKeyModel.Data,
			publicKeyModel.Comment,
			publicKeyModel.IsGlobal,
			expiresAt,
		}
	})

	localRecords := slicest.Map(linkModels, func(linkModel db.LinkModel) connector.DeployRecord {
		var expiresAt time.Time
		// if linkModel.ExpiresAt.Valid && linkModel.PublicKey.ExpiresAt.Valid {
		// 	if linkModel.ExpiresAt.Time.Before(linkModel.PublicKey.ExpiresAt.Time) {
		// 		expiresAt = linkModel.ExpiresAt.Time
		// 	} else {
		// 		expiresAt = linkModel.PublicKey.ExpiresAt.Time
		// 	}
		// } else if linkModel.ExpiresAt.Valid {
		// 	expiresAt = linkModel.ExpiresAt.Time
		// } else if linkModel.PublicKey.ExpiresAt.Valid {
		// 	expiresAt = linkModel.PublicKey.ExpiresAt.Time
		// }
		for _, t := range []sql.NullTime{linkModel.ExpiresAt, linkModel.PublicKey.ExpiresAt} {
			if t.Valid && (expiresAt.IsZero() || t.Time.Before(expiresAt)) {
				expiresAt = t.Time
			}
		}
		return connector.DeployRecord{
			linkModel.PublicKey.Algorithm,
			linkModel.PublicKey.Data,
			linkModel.PublicKey.Comment,
			linkModel.PublicKey.IsGlobal,
			expiresAt,
		}
	})

	return connector.DeployData{
		append(globalRecords, localRecords...),
		secret,
		cache,
		account.Serial,
	}, nil
}

func accountConnectionData(account client.Account) connector.ConnectionData {
	return connector.ConnectionData{
		account.Username,
		account.Host,
		account.Port,
	}
}

type connectorOperation func(ctx context.Context, deployData connector.DeployData, connectionData connector.ConnectionData, userRequester connector.UserRequester, progress chan<- connector.Progress) (ok bool, newCache connector.Cache, err error)

type accountProgressUpdate struct {
	accountId client.AccountId
	progress  client.ProgressAccountWithError
}

// runAccounts resolves each account's connector, runs the given operation
// (deploy or verify), and forwards every connector progress update into a
// shared aggregate snapshot sent on the caller-provided progress channel. It
// blocks until all accounts have been processed; per-account failures are
// carried in the snapshot, while the returned error reports operation-level
// failures (account lookup or the initial "requested" audit write). The caller
// owns progress and must close it. action is the audit action base (e.g.
// "account.deploy") recorded for the request and each account's outcome.
//
// The per-account operations run concurrently and each writes its own audit
// entry through c.bun, so this must be called on a pooled *bun.DB client, never
// on a transaction-bound one (a bun.Tx is not safe for concurrent use).
func (c *Client) runAccounts(ctx context.Context, selectOp func(connector.Connector) connectorOperation, action string, userRequester connector.UserRequester, progress chan<- client.ProgressAccounts, concurrent int, accountIds ...client.AccountId) error {
	accounts, err := c.GetAccounts(ctx, accountIds...)
	if err != nil {
		return err
	}

	// Record the request up front. If we cannot even record that the operation was requested, refuse to run it
	if err := c.writeAuditLog(ctx, c.bun, action+".requested", client.AuditLogDetails{
		{"accountIds", auditIds(accountIds)},
	}); err != nil {
		return i18n.WrapError(err, "errors.client.audit_write_failed")
	}

	if concurrent <= 0 {
		concurrent = len(accountIds)
	}
	semaphore := make(chan struct{}, concurrent)

	progressMap := slicest.ToMap(accounts, func(account client.Account) (client.AccountId, *client.ProgressAccountWithError) {
		return account.Id, &client.ProgressAccountWithError{ProgressAccount: client.ProgressAccount{Progress: 0, Status: i18n.Text("client.status.not_started")}}
	})

	accountProgressChan := make(chan accountProgressUpdate, concurrent)

	var wg sync.WaitGroup
	for _, account := range accounts {
		wg.Add(1)
		semaphore <- struct{}{}

		go func(account client.Account) {
			defer wg.Done()
			defer func() { <-semaphore }()

			c.runAccount(ctx, account, selectOp, action, accountProgressChan, userRequester)
		}(account)
	}

	go func() {
		wg.Wait()
		close(accountProgressChan)
	}()

	for accountProgress := range accountProgressChan {
		*progressMap[accountProgress.accountId] = accountProgress.progress
		progress <- client.ProgressAccounts{Accounts: progressMap}
	}

	return nil
}

// runAccount performs the connector operation for a single account, streaming
// its progress into the shared aggregate. Any failure to reach the connector is
// reported as an error status on that account rather than aborting the batch.
// Once the operation resolves, its outcome is recorded under action. The remote
// side effect has already happened by then, so a failed audit write cannot be
// rolled back — it is surfaced as an error on this account's progress instead.
func (c *Client) runAccount(ctx context.Context, account client.Account, selectOp func(connector.Connector) connectorOperation, action string, progressChanAccount chan accountProgressUpdate, userRequester connector.UserRequester) {
	sendProgress := func(progress client.ProgressAccountWithError) {
		progressChanAccount <- accountProgressUpdate{
			account.Id,
			progress,
		}
	}

	fail := func(err error) {
		sendProgress(client.ProgressAccountWithError{
			client.ProgressAccount{Progress: 1, Status: i18n.Text("client.status.error")},
			err,
		})
	}

	// runOp funnels every terminal path (connector unreachable, deploy-data
	// failure, drift, or normal completion) into a single return so the outcome
	// is audited exactly once. It returns the number of keys involved and the
	// operation error, if any.
	runOp := func() (int, error) {
		con, err := connector.Resolve(account.DeployMethod)
		if err != nil {
			fail(err)
			return 0, err
		}

		deployData, err := c.accountDeployData(ctx, con, account)
		if err != nil {
			fail(err)
			return 0, err
		}

		connectorProgress := make(chan connector.Progress)
		var ok bool
		var cache connector.Cache
		go func() {
			defer close(connectorProgress)
			ok, cache, err = selectOp(con)(ctx, deployData, accountConnectionData(account), userRequester, connectorProgress)
		}()
		for cp := range connectorProgress {
			sendProgress(client.ProgressAccountWithError{ProgressAccount: cp})
		}
		if err != nil {
			fail(err)
			return 0, err
		}

		serializedCache, err := cache.Serialize()
		if err != nil {
			fail(err)
			return 0, err
		}

		_, err = c.bun.NewUpdate().
			Model(&db.AccountModel{ID: int(account.Id), DeployCache: serializedCache}).
			Column("deploy_cache").
			WherePK().
			Exec(ctx)
		if err != nil {
			fail(err)
			return 0, err
		}

		if !ok {
			// A verify mismatch is a per-account failure, not a connector error.
			err := i18n.NewError("errors.client.out_of_sync")
			fail(err)
			return len(deployData.Records), err
		}

		return len(deployData.Records), nil
	}

	keyCount, opErr := runOp()

	if auditErr := c.writeAuditLog(ctx, c.bun, action, accountOpAuditDetails(account, keyCount, opErr)); auditErr != nil {
		sendProgress(client.ProgressAccountWithError{
			client.ProgressAccount{Progress: 1, Status: i18n.Text("client.status.error")},
			errors.Join(opErr, i18n.WrapError(auditErr, "errors.client.audit_write_failed")),
		})
	}
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

func (c *Client) DeployAccount(ctx context.Context, userRequester client.UserRequester, progress chan<- client.DeployProgressAccount, accountId client.AccountId) error {
	return projectSingleAccount(accountId, progress, func(p chan<- client.ProgressAccounts) error {
		return c.DeployAccounts(ctx, userRequester, p, accountId)
	})
}

func (c *Client) DeployAccounts(ctx context.Context, userRequester client.UserRequester, progress chan<- client.DeployProgressAccounts, accountIds ...client.AccountId) error {
	return c.runAccounts(ctx, func(con connector.Connector) connectorOperation {
		return func(ctx context.Context, deployData connector.DeployData, connectionData connector.ConnectionData, userRequester connector.UserRequester, progress chan<- connector.Progress) (bool, connector.Cache, error) {
			cache, err := con.Deploy(ctx, deployData, connectionData, userRequester, progress)
			return true, cache, err
		}
	}, "account.deploy", userRequester, progress, runtime.GOMAXPROCS(0), accountIds...)
}

func (c *Client) VerifyAccount(ctx context.Context, userRequester client.UserRequester, progress chan<- client.VerifyProgressAccount, accountId client.AccountId) error {
	return projectSingleAccount(accountId, progress, func(p chan<- client.ProgressAccounts) error {
		return c.VerifyAccounts(ctx, userRequester, p, accountId)
	})
}

func (c *Client) VerifyAccounts(ctx context.Context, userRequester client.UserRequester, progress chan<- client.VerifyProgressAccounts, accountIds ...client.AccountId) error {
	return c.runAccounts(ctx, func(con connector.Connector) connectorOperation {
		return con.Verify
	}, "account.verify", userRequester, progress, runtime.GOMAXPROCS(0), accountIds...)
}

// --- Other Operations ---

func modelToClientAuditLog(auditLogModel db.AuditLogModel) client.AuditLog {
	return client.AuditLog{
		client.AuditLogId(auditLogModel.ID),
		auditLogModel.Timestamp,
		auditLogModel.Action,
		auditLogModel.Details,
		client.AuditLogMetadata{
			Hostname: auditLogModel.Hostname.String,
			Hostuser: auditLogModel.Username,
			Referer:  auditLogModel.Referrer.String,
		},
	}
}

func (c *Client) ListAuditLogs(ctx context.Context, offset int, limit int) ([]client.AuditLog, error) {
	var auditLogModels []*db.AuditLogModel
	query := c.bun.NewSelect().
		Model(&auditLogModels).
		OrderExpr("timestamp DESC")
	if offset > 0 {
		query = query.Offset(offset)
	}
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

func (c *Client) ListConnectorKeys(ctx context.Context) ([]string, error) {
	return connector.Keys(), nil
}

// secretFields resolves the connector and describes the secret held in raw. An
// unregistered connector has no fields rather than being an error, so a UI can
// ask about a not-yet-chosen deploy method.
func secretFields(connectorKey string, raw string) ([]client.SecretField, error) {
	con, err := connector.Resolve(connectorKey)
	if err != nil {
		return nil, nil
	}

	secret, err := con.NewSecret(raw)
	if err != nil {
		return nil, err
	}

	return secret.Fields(), nil
}

func (c *Client) ConnectorSecretFields(ctx context.Context, connectorKey string) ([]client.SecretField, error) {
	return secretFields(connectorKey, "")
}

func (c *Client) AccountSecretFields(ctx context.Context, account client.Account) ([]client.SecretField, error) {
	return secretFields(account.DeployMethod, account.DeploySecret)
}

func (c *Client) OnboardHost(ctx context.Context, host string, port int, accountUsername string, deploymentKey string) (chan client.OnboardHostProgress, error) {
	panic("not planned")
}

func (c *Client) DecommisionAccount(ctx context.Context, id client.AccountId) (chan client.DecommisionAccountProgress, error) {
	panic("not planned")
}
