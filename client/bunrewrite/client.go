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
	_ "github.com/toeirei/keymaster/connector/ssh2"
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

// auditSecret redacts a connector secret for audit details.
func auditSecret(values map[string]string) string {
	fields := make([]string, 0, len(values))
	for key := range values {
		fields = append(fields, key+":"+"<redacted>")
	}
	return strings.Join(fields, ",")
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
		{"connector", account.Connector},
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
// entry is committed (or rolled back) atomically with the change it documents.
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

func modelToClientAccount(accountModel db.AccountModel) (client.Account, error) {
	conn, err := connector.Resolve(accountModel.Connector)
	if err != nil {
		return client.Account{}, err
	}

	secret, err := conn.ParseSecret(accountModel.ConnectorSecret)
	if err != nil {
		return client.Account{}, err
	}

	return client.Account{
		client.AccountId(accountModel.ID),
		accountModel.Username,
		accountModel.Host,
		accountModel.Port,
		accountModel.Connector,
		secret,
	}, nil
}

// serializeSecret turns the per-field values a UI collected into the JSON the
// connector_secret column stores, using the connector that owns its shape.
func serializeSecret(connectorKey string, values map[string]string) (string, error) {
	con, err := connector.Resolve(connectorKey)
	if err != nil {
		return "", err
	}
	secret, err := con.ParseSecretFromFields(values)
	if err != nil {
		return "", err
	}
	return secret.Serialize()
}

func (c *Client) CreateAccount(ctx context.Context, username string, host string, port int, connectorKey string, connectorSecret map[string]string) (client.Account, error) {
	serializedSecret, err := serializeSecret(connectorKey, connectorSecret)
	if err != nil {
		return client.Account{}, err
	}

	accountModel := db.AccountModel{
		Username:        username,
		Host:            host,
		Port:            port,
		IsActive:        true,
		IsDirty:         true,
		Connector:       connectorKey,
		ConnectorSecret: serializedSecret,
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
			{"connector", connectorKey},
			{"connectorSecret", auditSecret(connectorSecret)},
		})
	})
	if err != nil {
		return client.Account{}, err
	}

	return modelToClientAccount(accountModel)
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

	return modelToClientAccount(accountModel)
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

	accounts, err := slicest.MapX(accountModels, func(accountModel *db.AccountModel) (client.Account, error) {
		return modelToClientAccount(*accountModel)
	})
	if err != nil {
		return nil, err
	}

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

	return slicest.MapX(accountModels, func(accountModel *db.AccountModel) (client.Account, error) {
		return modelToClientAccount(*accountModel)
	})
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

// UpdateAccount edits where an account points, never how it authenticates.
// Overwriting connector_secret offline could discard the credential the target
// actually holds.
func (c *Client) UpdateAccount(ctx context.Context, id client.AccountId, username string, host string, port int) (client.Account, error) {
	accountModel := db.AccountModel{
		ID:       int(id),
		Username: username,
		Host:     host,
		Port:     port,
	}

	err := c.bun.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// update account
		_, err := tx.NewUpdate().
			Model(&accountModel).
			Column("username", "host", "port").
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
		})
	})
	if err != nil {
		return client.Account{}, err
	}

	return modelToClientAccount(accountModel)
}

// UpdateAccountConnectorForce is the escape hatch for a target whose credentials
// were changed outside Keymaster: without it the stored secret no longer
// authenticates and the sanctioned path has nothing to start from. It abandons
// any outstanding secret update, so it doubles as the manual abort for one that cannot
// converge.
func (c *Client) UpdateAccountConnectorForce(ctx context.Context, id client.AccountId, connectorKey string, connectorSecret map[string]string) (client.Account, error) {
	serializedSecret, err := serializeSecret(connectorKey, connectorSecret)
	if err != nil {
		return client.Account{}, err
	}

	accountModel := db.AccountModel{ID: int(id)}
	err = c.bun.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if err := tx.NewSelect().
			Model(&accountModel).
			WherePK().
			Scan(ctx); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return i18n.NewError("errors.client.account_not_found", id)
			}
			return err
		}

		connectorChanged := accountModel.Connector != connectorKey

		accountModel.Connector = connectorKey
		accountModel.ConnectorSecret = serializedSecret
		accountModel.ConnectorSecretRollback = sql.NullString{}
		columns := []string{"connector", "connector_secret", "connector_secret_rollback"}

		// A cache written by another connector is a foreign type its ParseCache
		// rejects, which would break every read of this account. Within one
		// connector the cache is kept: it still holds a valid known host, and the
		// hash simply stops matching, which is what marks the account dirty.
		if connectorChanged {
			accountModel.ConnectorCache = ""
			columns = append(columns, "connector_cache")
		}

		if _, err := tx.NewUpdate().
			Model(&accountModel).
			Column(columns...).
			WherePK().
			Exec(ctx); err != nil {
			return err
		}

		return c.writeAuditLog(ctx, tx, "account.update.connector_force", client.AuditLogDetails{
			{"id", strconv.Itoa(int(id))},
			{"connector", connectorKey},
			{"connectorSecret", auditSecret(connectorSecret)},
			{"connectorChanged", strconv.FormatBool(connectorChanged)},
		})
	})
	if err != nil {
		return client.Account{}, err
	}

	return modelToClientAccount(accountModel)
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
	con, err := connector.Resolve(account.Connector)
	if err != nil {
		return true, err
	}

	deployment, cache, err := c.accountDeployment(ctx, con, account)
	if err != nil {
		return true, err
	}

	valid, err := con.VerifyOffline(ctx, cache, deployment)
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

// --- Deploy & Verify & Secret Update ---

// accountConnectorCache reads an account's serialized connector cache, which
// deploy and verify own and [client.Account] therefore does not carry.
func (c *Client) accountConnectorCache(ctx context.Context, id client.AccountId) (string, error) {
	var rawCache string
	err := c.bun.NewSelect().
		Model((*db.AccountModel)(nil)).
		Column("connector_cache").
		Where("id = ?", int(id)).
		Scan(ctx, &rawCache)
	if errors.Is(err, sql.ErrNoRows) {
		return "", i18n.NewError("errors.client.account_not_found", id)
	}
	return rawCache, err
}

// persistConnectorCache stores what a connection observed. A connector hands
// back a cache on every path, so this runs after a failed operation too.
func (c *Client) persistConnectorCache(ctx context.Context, id client.AccountId, cache connector.Cache) error {
	serialized, err := cache.Serialize()
	if err != nil {
		return err
	}

	_, err = c.bun.NewUpdate().
		Model(&db.AccountModel{ID: int(id), ConnectorCache: serialized}).
		Column("connector_cache").
		WherePK().
		Exec(ctx)
	return err
}

// accountDeployment builds the state an account's target should be in, together
// with the connector cache describing what was last seen there. The cache is a
// second return rather than part of the deployment: a deployment is what a target
// should hold, a cache is what it did hold.
func (c *Client) accountDeployment(ctx context.Context, con connector.Connector, account client.Account) (connector.Deployment, connector.Cache, error) {
	rawCache, err := c.accountConnectorCache(ctx, account.Id)
	if err != nil {
		return connector.Deployment{}, nil, err
	}

	cache, err := con.ParseCache(rawCache)
	if err != nil {
		return connector.Deployment{}, nil, err
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
		return connector.Deployment{}, nil, err
	}

	var globalPublicKeyModels []db.PublicKeyModel
	err = c.bun.NewSelect().
		Model(&globalPublicKeyModels).
		Where("(expires_at IS NULL OR expires_at > ?)", now).
		Where("is_global = ?", true).
		Scan(ctx)
	if err != nil {
		return connector.Deployment{}, nil, err
	}

	globalRecords := slicest.Map(globalPublicKeyModels, func(publicKeyModel db.PublicKeyModel) connector.DeployRecord {
		var expiresAt time.Time
		if publicKeyModel.ExpiresAt.Valid {
			expiresAt = publicKeyModel.ExpiresAt.Time
		}
		return connector.DeployRecord{
			publicKeyModel.Algorithm,
			publicKeyModel.Data,
			expiresAt,
			publicKeyModel.Comment,
			publicKeyModel.IsGlobal,
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
			expiresAt,
			linkModel.PublicKey.Comment,
			linkModel.PublicKey.IsGlobal,
		}
	})

	return connector.Deployment{
		account.ConnectorSecret,
		append(globalRecords, localRecords...),
	}, cache, nil
}

// openAccountConnection dials the account with each candidate secret in turn and
// reports which one authenticated. The dial is the only way to learn what a target
// actually holds, so every caller that cannot assume the answer goes through here.
// Ordinary operations pass a single candidate; the ones that have to cope with an
// interrupted secret update pass both.
func (c *Client) openAccountConnection(
	ctx context.Context,
	con connector.Connector,
	account client.Account,
	cache connector.Cache,
	candidates []connector.Secret,
	userRequester connector.UserRequester,
) (connector.Connection, connector.Secret, error) {
	var lastErr error
	for _, secret := range candidates {
		conn, err := con.OpenConnection(ctx, secret, cache, account.Username, account.Host, account.Port, userRequester)
		if err != nil {
			lastErr = err
			continue
		}
		return conn, secret, nil
	}
	return nil, nil, lastErr
}

// connectorOperation is what a client-level operation does with an already-open
// connection. It returns no cache: the connection accumulates that itself and
// hands it over on Close, so the caller can persist it whether the operation
// succeeded or not.
type connectorOperation func(ctx context.Context, conn connector.Connection, deployment connector.Deployment, progress chan<- connector.Progress) (ok bool, err error)

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
// resolvePending lets an operation that writes settle an interrupted secret update
// first; a read-only one leaves it alone and copes by trying both secrets.
//
// The per-account operations run concurrently and each writes its own audit
// entry through c.bun, so this must be called on a pooled *bun.DB client, never
// on a transaction-bound one (a bun.Tx is not safe for concurrent use).
func (c *Client) runAccounts(ctx context.Context, selectOp func(connector.Connector) connectorOperation, action string, resolvePending bool, userRequester connector.UserRequester, progress chan<- client.ProgressAccounts, concurrent int, accountIds ...client.AccountId) error {
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
		return account.Id, &client.ProgressAccountWithError{ProgressAccount: client.ProgressAccount{0, i18n.Text("client.status.not_started")}}
	})

	accountProgressChan := make(chan accountProgressUpdate, concurrent)

	var wg sync.WaitGroup
	for _, account := range accounts {
		wg.Add(1)
		semaphore <- struct{}{}

		go func(account client.Account) {
			defer wg.Done()
			defer func() { <-semaphore }()

			c.runAccount(ctx, account, selectOp, action, resolvePending, accountProgressChan, userRequester)
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
// rolled back: it is surfaced as an error on this account's progress instead.
func (c *Client) runAccount(ctx context.Context, account client.Account, selectOp func(connector.Connector) connectorOperation, action string, resolvePending bool, progressChanAccount chan accountProgressUpdate, userRequester connector.UserRequester) {
	sendProgress := func(progress client.ProgressAccountWithError) {
		progressChanAccount <- accountProgressUpdate{
			account.Id,
			progress,
		}
	}

	send := func(progress client.ProgressAccount) {
		sendProgress(client.ProgressAccountWithError{ProgressAccount: progress})
	}

	fail := func(err error) {
		sendProgress(client.ProgressAccountWithError{
			client.ProgressAccount{1, i18n.Text("client.status.error")},
			err,
		})
	}

	// runOp funnels every terminal path (connector unreachable, deployment
	// failure, drift, or normal completion) into a single return so the outcome
	// is audited exactly once. It returns the number of keys involved and the
	// operation error, if any.
	runOp := func() (int, error) {
		con, err := connector.Resolve(account.Connector)
		if err != nil {
			fail(err)
			return 0, err
		}

		rollback, err := c.accountSecretRollback(ctx, account.Id)
		if err != nil {
			fail(err)
			return 0, err
		}

		// An interrupted secret update leaves it unclear which secret the target holds,
		// which an operation that writes cannot work around: it has to settle the
		// question first. A read-only one can simply try both, below.
		if rollback.Valid && resolvePending {
			account, err = c.resolvePendingSecretUpdate(ctx, con, account, rollback, userRequester, send)
			if err != nil {
				fail(err)
				return 0, err
			}
			rollback = sql.NullString{}
		}

		deployment, cache, err := c.accountDeployment(ctx, con, account)
		if err != nil {
			fail(err)
			return 0, err
		}

		// With a secret update still outstanding the target may hold either secret, so both
		// are offered. The recorded one goes first: it is the state we want to be in.
		candidates := []connector.Secret{deployment.Secret}
		if rollback.Valid {
			rollbackSecret, err := con.ParseSecret(rollback.String)
			if err != nil {
				fail(err)
				return 0, err
			}
			candidates = append(candidates, rollbackSecret)
		}

		// Opening carries no progress channel of its own: it is where the host key
		// handshake may stop to ask the user something, so the status has to be on
		// screen before the call rather than reported by it.
		send(client.ProgressAccount{0.1, i18n.Text("client.status.connecting")})
		conn, _, err := c.openAccountConnection(ctx, con, account, cache, candidates, userRequester)
		if err != nil {
			fail(err)
			return 0, err
		}
		// Close is idempotent, so this only guarantees the socket goes away on every
		// path. The call that matters is the one below, whose return value is the
		// cache worth keeping.
		defer conn.Close()

		connectorProgress := make(chan connector.Progress)
		var ok bool
		go func() {
			defer close(connectorProgress)
			ok, err = selectOp(con)(ctx, conn, deployment, connectorProgress)
		}()
		for cp := range connectorProgress {
			sendProgress(client.ProgressAccountWithError{ProgressAccount: cp})
		}
		opErr := err

		// Persisted whether the operation succeeded or not: the connection hands
		// back what it actually observed, including a host key the user has just
		// trusted, and discarding that would only mean asking them again.
		if err := c.persistConnectorCache(ctx, account.Id, conn.Close()); err != nil {
			fail(errors.Join(opErr, err))
			return 0, errors.Join(opErr, err)
		}
		if opErr != nil {
			fail(opErr)
			return 0, opErr
		}

		if !ok {
			// A verify mismatch is a per-account failure, not a connector error.
			err := i18n.NewError("errors.client.out_of_sync")
			fail(err)
			return len(deployment.Records), err
		}

		return len(deployment.Records), nil
	}

	keyCount, opErr := runOp()

	if auditErr := c.writeAuditLog(ctx, c.bun, action, accountOpAuditDetails(account, keyCount, opErr)); auditErr != nil {
		sendProgress(client.ProgressAccountWithError{
			client.ProgressAccount{1, i18n.Text("client.status.error")},
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

// accountSecretRollback reads the secret an outstanding secret update can fall back
// to. Invalid means no secret update is in flight.
func (c *Client) accountSecretRollback(ctx context.Context, id client.AccountId) (sql.NullString, error) {
	var rollback sql.NullString
	err := c.bun.NewSelect().
		Model((*db.AccountModel)(nil)).
		Column("connector_secret_rollback").
		Where("id = ?", int(id)).
		Scan(ctx, &rollback)
	if errors.Is(err, sql.ErrNoRows) {
		return rollback, i18n.NewError("errors.client.account_not_found", id)
	}
	return rollback, err
}

// beginSecretUpdate opens a secret update: the current secret is copied aside and the
// new one becomes current, before anything is attempted on the target, so a crash
// at any later point still leaves both candidates on record.
//
// The copy happens in SQL to preserve the stored bytes exactly, and the IS NULL
// guard is the whole state machine: it permits one secret update at a time and makes
// two concurrent callers race-safe, since only one can affect a row.
func (c *Client) beginSecretUpdate(ctx context.Context, id client.AccountId, serializedNew string) error {
	res, err := c.bun.NewUpdate().
		Model((*db.AccountModel)(nil)).
		Set("connector_secret_rollback = connector_secret").
		Set("connector_secret = ?", serializedNew).
		Where("id = ?", int(id)).
		Where("connector_secret_rollback IS NULL").
		Exec(ctx)
	if err != nil {
		return err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return i18n.NewError("errors.client.secret_update_pending", id)
	}
	return nil
}

// completeSecretUpdate closes a secret update, once the target has been confirmed
// reachable with the new secret.
func (c *Client) completeSecretUpdate(ctx context.Context, id client.AccountId) error {
	_, err := c.bun.NewUpdate().
		Model((*db.AccountModel)(nil)).
		Set("connector_secret_rollback = NULL").
		Where("id = ?", int(id)).
		Exec(ctx)
	return err
}

// The choices offered when a deploy meets an interrupted secret update whose target
// still holds the previous secret, in the order they are presented.
const (
	pendingSecretUpdateComplete = iota
	pendingSecretUpdateDiscard
)

// resolvePendingSecretUpdate brings an account out of an interrupted secret update before a
// deploy writes to it, and returns the account to deploy: the discard choice moves
// the previous secret back to current, so the caller has to work from what this
// hands back rather than what it passed in.
//
// Only one case is decided here without asking. If the recorded secret
// authenticates, the target demonstrably holds it, and clearing the rollback just
// records something already true. Any other case is a real fork -- finish the
// secret update, or throw it away -- and both answers lose something, so the operator
// picks.
func (c *Client) resolvePendingSecretUpdate(
	ctx context.Context,
	con connector.Connector,
	account client.Account,
	rollback sql.NullString,
	userRequester client.UserRequester,
	send func(client.ProgressAccount),
) (client.Account, error) {
	rawCache, err := c.accountConnectorCache(ctx, account.Id)
	if err != nil {
		return account, err
	}
	cache, err := con.ParseCache(rawCache)
	if err != nil {
		return account, err
	}

	send(client.ProgressAccount{0.05, i18n.Text("client.status.connecting")})
	conn, currentErr := con.OpenConnection(ctx, account.ConnectorSecret, cache, account.Username, account.Host, account.Port, userRequester)
	if currentErr == nil {
		if err := c.persistConnectorCache(ctx, account.Id, conn.Close()); err != nil {
			return account, err
		}
		if err := c.completeSecretUpdate(ctx, account.Id); err != nil {
			return account, err
		}
		return account, c.writeAuditLog(ctx, c.bun, "account.update.secret", accountOpAuditDetails(account, 0, nil))
	}

	rollbackSecret, err := con.ParseSecret(rollback.String)
	if err != nil {
		return account, err
	}

	probe, rollbackErr := con.OpenConnection(ctx, rollbackSecret, cache, account.Username, account.Host, account.Port, userRequester)
	if rollbackErr != nil {
		// Neither secret authenticates, so there is no choice worth offering: nothing
		// can be done to this target remotely at all.
		return account, i18n.WrapError(errors.Join(currentErr, rollbackErr), "errors.client.secret_update_unreachable", account.Id)
	}
	if err := c.persistConnectorCache(ctx, account.Id, probe.Close()); err != nil {
		return account, err
	}

	// With no one to ask, refuse rather than guess which loss is acceptable.
	if userRequester == nil {
		return account, i18n.NewError("errors.client.secret_update_pending", account.Id)
	}

	switch userRequester.RequestChoice([]fmt.Stringer{
		i18n.Text("client.pending_secret_update.complete", account),
		i18n.Text("client.pending_secret_update.discard", account),
	}) {
	case pendingSecretUpdateComplete:
		return account, c.convergeSecretUpdate(ctx, userRequester, send, account, con)

	case pendingSecretUpdateDiscard:
		if err := c.abortSecretUpdate(ctx, account.Id); err != nil {
			return account, err
		}
		if err := c.writeAuditLog(ctx, c.bun, "account.update.secret.discarded", accountOpAuditDetails(account, 0, nil)); err != nil {
			return account, err
		}
		// The current secret has changed, so the caller must not keep using the copy
		// it already has.
		return c.GetAccount(ctx, account.Id)

	default:
		return account, i18n.NewError("errors.client.secret_update_pending", account.Id)
	}
}

// convergeSecretUpdate resumes the interrupted secret update, forwarding its progress
// into the surrounding account operation.
func (c *Client) convergeSecretUpdate(
	ctx context.Context,
	userRequester client.UserRequester,
	send func(client.ProgressAccount),
	account client.Account,
	con connector.Connector,
) error {
	serialized, err := account.ConnectorSecret.Serialize()
	if err != nil {
		return err
	}

	progress := make(chan client.UpdateSecretProgressAccount)
	var opErr error
	go func() {
		defer close(progress)
		opErr = c.runSecretUpdate(ctx, userRequester, progress, account, con, account.ConnectorSecret, serialized)
	}()
	for p := range progress {
		send(p)
	}

	auditErr := c.writeAuditLog(ctx, c.bun, "account.update.secret", accountOpAuditDetails(account, 0, opErr))
	if auditErr != nil {
		return errors.Join(opErr, i18n.WrapError(auditErr, "errors.client.audit_write_failed"))
	}
	return opErr
}

// abortSecretUpdate unwinds a secret update, for the paths where the target is known
// to still hold the previous secret: nothing was written, or what was written has
// been restored over the connection that authenticated with it.
//
// Leaving the account pending instead would be safe but needlessly strict -- the
// operator could not simply correct a bad secret and try again, since a different
// target is refused while a secret update is outstanding.
func (c *Client) abortSecretUpdate(ctx context.Context, id client.AccountId) error {
	_, err := c.bun.NewUpdate().
		Model((*db.AccountModel)(nil)).
		Set("connector_secret = connector_secret_rollback").
		Set("connector_secret_rollback = NULL").
		Where("id = ?", int(id)).
		Where("connector_secret_rollback IS NOT NULL").
		Exec(ctx)
	return err
}

func (c *Client) UpdateAccountSecret(ctx context.Context, userRequester client.UserRequester, progress chan<- client.UpdateSecretProgressAccount, accountId client.AccountId, connectorSecret map[string]string) error {
	account, err := c.GetAccount(ctx, accountId)
	if err != nil {
		return err
	}

	con, err := connector.Resolve(account.Connector)
	if err != nil {
		return err
	}

	newSecret, err := con.ParseSecretFromFields(connectorSecret)
	if err != nil {
		return err
	}
	serializedNew, err := newSecret.Serialize()
	if err != nil {
		return err
	}

	// Record the request up front, the way runAccounts does. A secret update can strand a
	// target, so if we cannot even record that it was asked for, refuse to run it.
	if err := c.writeAuditLog(ctx, c.bun, "account.update.secret.requested", client.AuditLogDetails{
		{"accountId", strconv.Itoa(int(accountId))},
		{"connectorSecret", auditSecret(connectorSecret)},
	}); err != nil {
		return i18n.WrapError(err, "errors.client.audit_write_failed")
	}

	opErr := c.runSecretUpdate(ctx, userRequester, progress, account, con, newSecret, serializedNew)

	auditErr := c.writeAuditLog(ctx, c.bun, "account.update.secret", append(
		accountOpAuditDetails(account, 0, opErr),
		client.AuditLogDetail{"connectorSecret", auditSecret(connectorSecret)},
	))
	if auditErr != nil {
		return errors.Join(opErr, i18n.WrapError(auditErr, "errors.client.audit_write_failed"))
	}
	return opErr
}

// runSecretUpdate performs the secret update. It is a deploy keyed to the new secret,
// written over whichever credential still authenticates, then confirmed over a
// second connection opened with the new one.
//
// Which secret the target holds is never assumed: after an interrupted secret update
// it could be either, and the dial itself is what settles it. The connection that
// authenticated stays open as the way back, because once a new-secret-only state
// is written the old credential no longer authenticates and a database-only
// rollback could not recover.
func (c *Client) runSecretUpdate(
	ctx context.Context,
	userRequester client.UserRequester,
	progress chan<- client.UpdateSecretProgressAccount,
	account client.Account,
	con connector.Connector,
	newSecret connector.Secret,
	serializedNew string,
) error {
	rollback, err := c.accountSecretRollback(ctx, account.Id)
	if err != nil {
		return err
	}

	currentSerialized, err := account.ConnectorSecret.Serialize()
	if err != nil {
		return err
	}

	// Candidates in the order the target is most likely to hold them. The rollback
	// secret is the known-good one, so it goes first on a fresh secret update; on a
	// resume the new secret may already be installed, which the second attempt
	// finds.
	var candidates []connector.Secret
	switch {
	case !rollback.Valid:
		if err := c.beginSecretUpdate(ctx, account.Id, serializedNew); err != nil {
			return err
		}
		candidates = []connector.Secret{account.ConnectorSecret, newSecret}

	case serializedNew == currentSerialized:
		// Resuming the secret update already in flight. No write: the columns already
		// say what they need to.
		rollbackSecret, err := con.ParseSecret(rollback.String)
		if err != nil {
			return err
		}
		candidates = []connector.Secret{rollbackSecret, newSecret}

	default:
		// Retargeting would drop the secret the target might be holding and lock it
		// out for good.
		return i18n.NewError("errors.client.secret_update_pending", account.Id)
	}

	oldDeployment, cache, err := c.accountDeployment(ctx, con, account)
	if err != nil {
		return err
	}
	// accountDeployment keys the deployment to the account's stored secret, which
	// on a resume is already the new target. Both states are needed: the new one to
	// install, the old one to restore.
	oldDeployment.Secret = candidates[0]
	newDeployment := connector.Deployment{newSecret, oldDeployment.Records}

	send := func(fraction float64, status string) {
		progress <- client.UpdateSecretProgressAccount{fraction, i18n.Text(status)}
	}

	// forward rescales a connector's own 0..1 into the slice of the overall
	// operation it occupies, since one secret update spans several connector calls.
	forward := func(from, to float64, op func(progress chan<- connector.Progress) error) error {
		connectorProgress := make(chan connector.Progress)
		var opErr error
		go func() {
			defer close(connectorProgress)
			opErr = op(connectorProgress)
		}()
		for cp := range connectorProgress {
			progress <- client.UpdateSecretProgressAccount{from + cp.Progress*(to-from), cp.Status}
		}
		return opErr
	}

	var lastErr error
	for i, auth := range candidates {
		send(0.1, "client.status.connecting")
		escape, err := con.OpenConnection(ctx, auth, cache, account.Username, account.Host, account.Port, userRequester)
		if err != nil {
			lastErr = err
			continue
		}
		// Idempotent, so finishSecretUpdate can still close it for its cache; this
		// only makes sure the socket cannot outlive the operation.
		defer escape.Close()

		// candidates[0] is always the old credential and candidates[1] the new one,
		// so only the first can restore: reaching the second means the target already
		// holds the new secret and there is nothing to go back to.
		canRestore := i == 0

		return c.finishSecretUpdate(ctx, account, con, escape, cache, oldDeployment, newDeployment, canRestore, userRequester, send, forward)
	}

	// Nothing authenticated, so nothing was written. On a fresh secret update that means
	// the target still holds the secret it held before, so the swap can be unwound
	// rather than left outstanding. A resume cannot claim that: it was already
	// unclear which secret the target held, and it still is.
	err = i18n.WrapError(lastErr, "errors.client.secret_update_unreachable", account.Id)
	if !rollback.Valid {
		if abortErr := c.abortSecretUpdate(ctx, account.Id); abortErr != nil {
			return errors.Join(err, abortErr)
		}
	}
	return err
}

// finishSecretUpdate owns the part of the secret update that can leave a target in a
// half-changed state: the write, the confirmation, and the way back.
func (c *Client) finishSecretUpdate(
	ctx context.Context,
	account client.Account,
	con connector.Connector,
	escape connector.Connection,
	cache connector.Cache,
	oldDeployment connector.Deployment,
	newDeployment connector.Deployment,
	canRestore bool,
	userRequester client.UserRequester,
	send func(float64, string),
	forward func(float64, float64, func(chan<- connector.Progress) error) error,
) error {
	// The escape connection's cache is only worth persisting if nothing better
	// comes along, so hold it until the end.
	persist := func(cache connector.Cache, err error) error {
		if cacheErr := c.persistConnectorCache(ctx, account.Id, cache); cacheErr != nil {
			return errors.Join(err, cacheErr)
		}
		return err
	}

	// The write is the point of no return. Before it the target still holds what it
	// held, so a failure can unwind the swap entirely.
	if err := forward(0.15, 0.5, func(p chan<- connector.Progress) error {
		return escape.Deploy(ctx, newDeployment, p)
	}); err != nil {
		err = persist(escape.Close(), err)
		if canRestore {
			if abortErr := c.abortSecretUpdate(ctx, account.Id); abortErr != nil {
				return errors.Join(err, abortErr)
			}
		}
		return err
	}

	send(0.55, "client.status.confirming_secret")
	confirm, confirmErr := con.OpenConnection(ctx, newDeployment.Secret, cache, account.Username, account.Host, account.Port, userRequester)

	var ok bool
	if confirmErr == nil {
		defer confirm.Close()
		confirmErr = forward(0.6, 0.95, func(p chan<- connector.Progress) error {
			var err error
			ok, err = confirm.Verify(ctx, newDeployment, p)
			return err
		})
		// The confirming connection read the target, so its cache is the truthful
		// one whether or not it matched.
		cache = confirm.Close()
	}

	if confirmErr == nil && ok {
		if err := persist(cache, nil); err != nil {
			escape.Close()
			return err
		}
		escape.Close()
		if err := c.completeSecretUpdate(ctx, account.Id); err != nil {
			return err
		}
		send(1, "client.status.finished")
		return nil
	}

	if confirmErr == nil {
		confirmErr = i18n.NewError("errors.client.secret_not_confirmed", account.Id)
	}

	if !canRestore {
		// The target holds the new secret and the confirmation failed for some other
		// reason. Leaving it alone keeps the secret update resumable.
		return persist(escape.Close(), confirmErr)
	}

	send(0.97, "client.status.restoring_secret")
	if restoreErr := forward(0.97, 1, func(p chan<- connector.Progress) error {
		return escape.Deploy(ctx, oldDeployment, p)
	}); restoreErr != nil {
		return persist(escape.Close(), errors.Join(confirmErr, i18n.WrapError(restoreErr, "errors.connector.secret_restore_failed", account.Id)))
	}

	// Restored over the credential that authenticated with it, so the target
	// verifiably holds the previous secret again and the swap can be unwound. That
	// leaves no outstanding secret update, so a corrected secret can simply be submitted
	// again.
	err := persist(escape.Close(), confirmErr)
	if abortErr := c.abortSecretUpdate(ctx, account.Id); abortErr != nil {
		return errors.Join(err, abortErr)
	}
	return err
}

func (c *Client) DeployAccount(ctx context.Context, userRequester client.UserRequester, progress chan<- client.DeployProgressAccount, accountId client.AccountId) error {
	return projectSingleAccount(accountId, progress, func(p chan<- client.ProgressAccounts) error {
		return c.DeployAccounts(ctx, userRequester, p, accountId)
	})
}

func (c *Client) DeployAccounts(ctx context.Context, userRequester client.UserRequester, progress chan<- client.DeployProgressAccounts, accountIds ...client.AccountId) error {
	return c.runAccounts(ctx, func(con connector.Connector) connectorOperation {
		return func(ctx context.Context, conn connector.Connection, deployment connector.Deployment, progress chan<- connector.Progress) (bool, error) {
			return true, conn.Deploy(ctx, deployment, progress)
		}
	}, "account.deploy", true, userRequester, progress, runtime.GOMAXPROCS(0), accountIds...)
}

func (c *Client) VerifyAccount(ctx context.Context, userRequester client.UserRequester, progress chan<- client.VerifyProgressAccount, accountId client.AccountId) error {
	return projectSingleAccount(accountId, progress, func(p chan<- client.ProgressAccounts) error {
		return c.VerifyAccounts(ctx, userRequester, p, accountId)
	})
}

func (c *Client) VerifyAccounts(ctx context.Context, userRequester client.UserRequester, progress chan<- client.VerifyProgressAccounts, accountIds ...client.AccountId) error {
	return c.runAccounts(ctx, func(con connector.Connector) connectorOperation {
		return func(ctx context.Context, conn connector.Connection, deployment connector.Deployment, progress chan<- connector.Progress) (bool, error) {
			return conn.Verify(ctx, deployment, progress)
		}
	}, "account.verify", false, userRequester, progress, runtime.GOMAXPROCS(0), accountIds...)
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

func (c *Client) ConnectorSecretFields(connectorKey string) ([]client.SecretField, error) {
	con, err := connector.Resolve(connectorKey)
	if err != nil {
		return nil, err
	}

	return con.SecretFields(), nil
}

func (c *Client) OnboardHost(ctx context.Context, host string, port int, accountUsername string, deploymentKey string) (chan client.OnboardHostProgress, error) {
	panic("not planned")
}

func (c *Client) DecommisionAccount(ctx context.Context, id client.AccountId) (chan client.DecommisionAccountProgress, error) {
	panic("not planned")
}
