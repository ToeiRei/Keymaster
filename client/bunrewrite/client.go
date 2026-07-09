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
	"strings"
	"time"

	"github.com/bobg/go-generics/v4/slices"
	"github.com/toeirei/keymaster/client"
	"github.com/toeirei/keymaster/config"
	"github.com/toeirei/keymaster/core"
	"github.com/toeirei/keymaster/core/db"
	"github.com/toeirei/keymaster/core/model"
	"github.com/toeirei/keymaster/core/sshkey"
	"github.com/toeirei/keymaster/tags"
	"github.com/toeirei/keymaster/tags/tagsbun"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/mysqldialect"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/dialect/sqlitedialect"

	// sql drivers
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

type BunClient struct {
	config config.Config
	log    *log.Logger
	bun    bun.IDB
}

// *[BunClient] implements [client.Client]
var _ client.Client = (*BunClient)(nil)

func NewBunClient(config config.Config, logger *log.Logger) (*BunClient, error) {
	// resolve db drive
	var dbDriver string
	switch config.Database.Type {
	case "sqlite":
		dbDriver = "sqlite"
	case "postgres":
		dbDriver = "pgx"
	case "mysql":
		dbDriver = "mysql"
	default:
		return nil, fmt.Errorf("unknown db type: %w", config.Database.Type)
	}

	// create connection
	dbConn, err := sql.Open(dbDriver, config.Database.Dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// create bun instance
	var dbBun *bun.DB
	switch dbDriver {
	case "sqlite":
		dbBun = bun.NewDB(dbConn, sqlitedialect.New())
	case "pgx":
		dbBun = bun.NewDB(dbConn, pgdialect.New())
	case "mysql":
		dbBun = bun.NewDB(dbConn, mysqldialect.New())
	}

	return &BunClient{config, logger, dbBun}, nil
}

func NewDefaultBunClient(logger *log.Logger) (*BunClient, error) {
	return NewBunClient(client.NewDefaultConfig(), logger)
}

func (c *BunClient) Close(ctx context.Context) error {
	return c.bun.(*bun.DB).Close()
}

func (c *BunClient) WithTransaction(ctx context.Context, fn func(ctx context.Context, c client.Client) error) error {
	return c.bun.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		return fn(ctx, &BunClient{c.config, c.log, tx})
	})
}

// --- Helper functions ---

// encodeHostPort encodes host and port into a single string for storage.
// Format: "hostname:port" (e.g., "example.com:22")
func encodeHostPort(host string, port int) string {
	return fmt.Sprintf("%s:%d", host, port)
}

// decodeHostPort decodes a host:port string into separate components.
// Returns host, port, and error if parsing fails.
func decodeHostPort(encoded string) (string, int, error) {
	parts := strings.SplitN(encoded, ":", 2)
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid host:port format: %s", encoded)
	}
	var port int
	_, err := fmt.Sscanf(parts[1], "%d", &port)
	if err != nil {
		return "", 0, fmt.Errorf("invalid port: %s", parts[1])
	}
	return parts[0], port, nil
}

// accountModelToClient converts a core.model.Account to a client.Account.
// Hostname is expected to be encoded as "host:port".
func (c *BunClient) accountModelToClient(m *model.Account) (client.Account, error) {
	host, port, err := decodeHostPort(m.Hostname)
	if err != nil {
		// Fallback: assume port 22 if decoding fails
		return client.Account{
			Id:           client.AccountId(m.ID),
			Username:     m.Username,
			Host:         m.Hostname,
			Port:         22,
			DeployMethod: "ssh",
			DeploySecret: "",
			DeployCache:  "",
		}, nil
	}
	return client.Account{
		Id:           client.AccountId(m.ID),
		Username:     m.Username,
		Host:         host,
		Port:         port,
		DeployMethod: "ssh",
		DeploySecret: "",
		DeployCache:  "",
	}, nil
}

// --- PublicKey Management ---

func modelToClientPublicKey(publicKeyModel db.PublicKeyModel) client.PublicKey {
	return client.PublicKey{
		Id:        client.PublicKeyId(publicKeyModel.ID),
		Algorithm: publicKeyModel.Algorithm,
		Data:      publicKeyModel.KeyData,
		Comment:   publicKeyModel.Comment,
		Tags: slices.Map(publicKeyModel.Tags, func(tagModel db.TagModel) tags.Tag {
			return tags.Tag(tagModel.Slug)
		}),
	}
}

func (c *BunClient) CreatePublicKey(ctx context.Context, key string, comment string, tags tags.Tags) (client.PublicKey, error) {
	// Parse the key to extract algorithm and key data.
	alg, data, _, err := sshkey.Parse(key)
	if err != nil {
		return client.PublicKey{}, err
	}

	// mock expiresAt
	expiresAt, _ := time.Parse(time.DateOnly, "9999-01-02")

	// create public key model
	publicKeyModel := db.PublicKeyModel{
		Algorithm: alg,
		KeyData:   data,
		Comment:   comment,
		ExpiresAt: sql.NullTime{expiresAt, true},
	}

	err = c.bun.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// insert public key
		_, err = tx.NewInsert().
			Model(&publicKeyModel).
			Exec(ctx)
		if err != nil {
			return err
		}

		if len(tags) > 0 {
			// upsert missing tags
			tagModels_Insert := slices.Map(tags.Slice(), func(tag string) *db.TagModel {
				return &db.TagModel{Slug: tag}
			})

			_, err = tx.NewInsert().
				Model(tagModels_Insert).
				Ignore().
				Exec(ctx)
			if err != nil {
				return err
			}

			// resolve tags
			var tagModels_Select []db.TagModel

			_, err = tx.NewSelect().
				Model(&tagModels_Select).
				Where("slug IN (?)", bun.List(tags.Slice())).
				Exec(ctx)
			if err != nil {
				return err
			}

			if len(tagModels_Select) != len(tags) {
				return errors.New("tags quantity missmatch after upsert")
			}

			publicKeyModel.Tags = tagModels_Select

			// connect tags to public key
			publicKeyToTagModels := slices.Map(tagModels_Select, func(tagModel db.TagModel) *db.PublicKeyToTagModel {
				return &db.PublicKeyToTagModel{
					PublicKeyId: publicKeyModel.ID,
					TagId:       tagModel.ID,
				}
			})

			_, err = tx.NewInsert().
				Model(publicKeyToTagModels).
				Exec(ctx)
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return client.PublicKey{}, err
	}

	return modelToClientPublicKey(publicKeyModel), nil
}

func (c *BunClient) GetPublicKey(ctx context.Context, id client.PublicKeyId) (client.PublicKey, error) {
	publicKeyModel := db.PublicKeyModel{ID: int(id)}
	_, err := c.bun.NewSelect().
		Model(&publicKeyModel).
		WherePK().
		Relation("Tags").
		Exec(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return client.PublicKey{}, fmt.Errorf("public key not found: %d", id)
	}
	if err != nil {
		return client.PublicKey{}, err
	}

	return modelToClientPublicKey(publicKeyModel), nil
}

func (c *BunClient) GetPublicKeys(ctx context.Context, ids ...client.PublicKeyId) ([]client.PublicKey, error) {
	var publicKeysModel []*db.PublicKeyModel
	_, err := c.bun.NewSelect().
		Model(&publicKeysModel).
		Where("id IN (?)", bun.List(ids)).
		Relation("Tags").
		Exec(ctx)
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

func (c *BunClient) ListPublicKeys(ctx context.Context, tagMatcher string) ([]client.PublicKey, error) {
	var publicKeysModel []*db.PublicKeyModel

	if len(tagMatcher) == 0 {
		_, err := c.bun.NewSelect().
			Model(&publicKeysModel).
			Relation("Tags").
			Exec(ctx)
		if err != nil {
			return nil, err
		}
	} else {
		expr, err := tags.ParseMatcher(tagMatcher)
		if err != nil {
			return nil, fmt.Errorf("failed to parse matcher %q: %v", tagMatcher, err)
		}

		_, err = c.bun.NewSelect().
			Model(&publicKeysModel).
			Relation("Tags").
			Apply(tagsbun.TagsExprToWhere(expr, tagsbun.TagsExprToSubqueryConfig{
				TaggedTable:    "public_keys",
				TaggedColumnId: "id",

				TaggedToTagTable:          "public_key_to_tags",
				TaggedToTagColumnTagId:    "tag_id",
				TaggedToTagColumnTaggedId: "public_key_id",

				TagTable:       "tags",
				TagColumnId:    "id",
				TagColumnValue: "slug",
			})).
			Exec(ctx)
		if err != nil {
			return nil, err
		}
	}

	return slices.Map(publicKeysModel, func(publicKeyModel *db.PublicKeyModel) client.PublicKey {
		return modelToClientPublicKey(*publicKeyModel)
	}), nil
}

func (c *BunClient) ListPublicKeysLinkedToAccount(ctx context.Context, accountId client.AccountId, expired bool) ([]client.PublicKey, error) {
	km := core.DefaultKeyManager()
	if km == nil {
		return nil, errors.New("no key manager available")
	}

	// Get keys assigned to this account.
	pks, err := km.GetKeysForAccount(int(accountId))
	if err != nil {
		return nil, fmt.Errorf("failed to get keys for account: %w", err)
	}

	// Also include global keys.
	global, err := km.GetGlobalPublicKeys()
	if err != nil {
		return nil, fmt.Errorf("failed to get global public keys: %w", err)
	}

	// Merge and deduplicate by ID.
	seen := make(map[int]bool)
	var result []client.PublicKey

	for _, pk := range global {
		seen[pk.ID] = true
		result = append(result, client.PublicKey{
			Id:        client.PublicKeyId(pk.ID),
			Algorithm: pk.Algorithm,
			Data:      pk.KeyData,
			Comment:   pk.Comment,
			Tags:      nil,
		})
	}

	for _, pk := range pks {
		if !seen[pk.ID] {
			seen[pk.ID] = true
			result = append(result, client.PublicKey{
				Id:        client.PublicKeyId(pk.ID),
				Algorithm: pk.Algorithm,
				Data:      pk.KeyData,
				Comment:   pk.Comment,
				Tags:      nil,
			})
		}
	}

	return result, nil
}

func (c *BunClient) UpdatePublicKey(ctx context.Context, id client.PublicKeyId, comment string, tags tags.Tags) (client.PublicKey, error) {
	// mock expiresAt
	expiresAt, _ := time.Parse(time.DateOnly, "9999-01-02")

	publicKeyModel := db.PublicKeyModel{
		ID:        int(id),
		Comment:   comment,
		ExpiresAt: sql.NullTime{expiresAt, true},
	}

	err := c.bun.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// update public key
		_, err := tx.NewUpdate().
			Model(&publicKeyModel).
			Column("comment", "expires_at").
			WherePK().
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = tx.NewSelect().
			Model(&publicKeyModel).
			WherePK().
			Relation("Tags").
			Exec(ctx)
		if err != nil {
			return err
		}

		existingTags := slices.Map(publicKeyModel.Tags, func(tagModel db.TagModel) string {
			return tagModel.Slug
		})
		newTags := slices.Filter(tags.Slice(), func(tag string) bool {
			return !slices.Contains(existingTags, tag)
		})
		oldTags := slices.Filter(existingTags, func(tag string) bool {
			return !slices.Contains(tags.Slice(), tag)
		})

		// upsert new tags
		tagModels_Insert := slices.Map(newTags, func(tag string) *db.TagModel {
			return &db.TagModel{Slug: tag}
		})

		_, err = tx.NewInsert().
			Model(tagModels_Insert).
			Ignore().
			Exec(ctx)
		if err != nil {
			return err
		}

		// resolve tags
		var newTagModels_Select, oldTagModels_Select []db.TagModel

		_, err = tx.NewSelect().
			Model(&newTagModels_Select).
			Where("slug IN (?)", bun.List(newTags)).
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = tx.NewSelect().
			Model(&oldTagModels_Select).
			Where("slug IN (?)", bun.List(oldTags)).
			Exec(ctx)
		if err != nil {
			return err
		}

		// update relations

		oldTagIds := slices.Map(oldTagModels_Select, func(tagModel db.TagModel) int {
			return tagModel.ID
		})

		_, err = tx.NewDelete().
			Model((*db.PublicKeyToTagModel)(nil)).
			Where("public_key_id = ?", publicKeyModel.ID).
			Where("tag_id IN (?)", bun.List(oldTagIds)).
			Exec(ctx)
		if err != nil {
			return err
		}

		newPublicKeyToTagModels := slices.Map(newTagModels_Select, func(tagModel db.TagModel) *db.PublicKeyToTagModel {
			return &db.PublicKeyToTagModel{
				PublicKeyId: publicKeyModel.ID,
				TagId:       tagModel.ID,
			}
		})
		_, err = tx.NewInsert().
			Model(newPublicKeyToTagModels).
			Exec(ctx)
		if err != nil {
			return err
		}

		// update [publicKeyModel.Tags] in-place to avoid another db query
		publicKeyModel.Tags = append(
			slices.Filter(publicKeyModel.Tags, func(tagModel db.TagModel) bool {
				return !slices.Contains(oldTags, tagModel.Slug)
			}),
			newTagModels_Select...,
		)

		return nil
	})
	if err != nil {
		return client.PublicKey{}, err
	}

	return modelToClientPublicKey(publicKeyModel), nil
}

func (c *BunClient) DeletePublicKeys(ctx context.Context, ids ...client.PublicKeyId) error {
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
			return fmt.Errorf("%d public key ids could not be found", unaffectedRows)
		}

		return nil
	})
}

// --- Account Management ---

func modelToClientAccount(accountModel db.AccountModel) client.Account {
	host, port, err := decodeHostPort(accountModel.Hostname)
	if err != nil {
		// Fallback: assume port 22 if decoding fails.
		host, port = accountModel.Hostname, 22
	}

	return client.Account{
		Id:           client.AccountId(accountModel.ID),
		Username:     accountModel.Username,
		Host:         host,
		Port:         port,
		DeployMethod: accountModel.DeployMethod,
		DeploySecret: accountModel.DeploySecret,
		DeployCache:  "",
	}
}

func (c *BunClient) CreateAccount(ctx context.Context, username string, host string, port int, deploymentMethod string, deploymentSecret string) (client.Account, error) {
	accountModel := db.AccountModel{
		Username:     username,
		Hostname:     encodeHostPort(host, port),
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

func (c *BunClient) GetAccount(ctx context.Context, id client.AccountId) (client.Account, error) {
	accountModel := db.AccountModel{ID: int(id)}
	_, err := c.bun.NewSelect().
		Model(&accountModel).
		WherePK().
		Exec(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return client.Account{}, fmt.Errorf("account not found: %d", id)
	}
	if err != nil {
		return client.Account{}, err
	}

	return modelToClientAccount(accountModel), nil
}

func (c *BunClient) GetAccounts(ctx context.Context, ids ...client.AccountId) ([]client.Account, error) {
	var accountModels []*db.AccountModel
	_, err := c.bun.NewSelect().
		Model(&accountModels).
		Where("id IN (?)", bun.List(ids)).
		Exec(ctx)
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

func (c *BunClient) ListAccounts(ctx context.Context) ([]client.Account, error) {
	var accountModels []*db.AccountModel
	_, err := c.bun.NewSelect().
		Model(&accountModels).
		Exec(ctx)
	if err != nil {
		return nil, err
	}

	return slices.Map(accountModels, func(accountModel *db.AccountModel) client.Account {
		return modelToClientAccount(*accountModel)
	}), nil
}

func (c *BunClient) ListAccountsDirty(ctx context.Context) ([]client.Account, error) {
	var accountModels []*db.AccountModel
	_, err := c.bun.NewSelect().
		Model(&accountModels).
		Where("is_dirty = ?", true).
		Exec(ctx)
	if err != nil {
		return nil, err
	}

	return slices.Map(accountModels, func(accountModel *db.AccountModel) client.Account {
		return modelToClientAccount(*accountModel)
	}), nil
}

func (c *BunClient) ListAccountsLinkedToPublicKey(ctx context.Context, publicKeyId client.PublicKeyId, expired bool) ([]client.Account, error) {
	km := core.DefaultKeyManager()
	if km == nil {
		return nil, errors.New("no key manager available")
	}

	// Get accounts that have this key assigned.
	accounts, err := km.GetAccountsForKey(int(publicKeyId))
	if err != nil {
		return nil, fmt.Errorf("failed to get accounts for key: %w", err)
	}

	var result []client.Account
	for _, acc := range accounts {
		clientAcc, err := c.accountModelToClient(&acc)
		if err != nil {
			clientAcc = client.Account{
				Id:           client.AccountId(acc.ID),
				Username:     acc.Username,
				Host:         acc.Hostname,
				Port:         22,
				DeployMethod: "ssh",
			}
		}
		result = append(result, clientAcc)
	}

	return result, nil
}

func (c *BunClient) UpdateAccount(ctx context.Context, id client.AccountId, username string, host string, port int, deploymentMethod string, deploymentSecret string) (client.Account, error) {
	accountModel := db.AccountModel{ID: int(id)}

	err := c.bun.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewUpdate().
			Model((*db.AccountModel)(nil)).
			Where("id = ?", int(id)).
			Set("username = ?", username).
			Set("hostname = ?", encodeHostPort(host, port)).
			Set("deploy_method = ?", deploymentMethod).
			Set("deploy_secret = ?", deploymentSecret).
			Exec(ctx)
		if err != nil {
			return err
		}

		_, err = tx.NewSelect().
			Model(&accountModel).
			WherePK().
			Exec(ctx)
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

func (c *BunClient) DeleteAccounts(ctx context.Context, ids ...client.AccountId) error {
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
			return fmt.Errorf("%d account ids could not be found", unaffectedRows)
		}

		return nil
	})
}

func (c *BunClient) IsAccountDirty(ctx context.Context, account client.Account) (bool, error) {
	accountModel := db.AccountModel{ID: int(account.Id)}
	_, err := c.bun.NewSelect().
		Model(&accountModel).
		Column("is_dirty").
		WherePK().
		Exec(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return false, fmt.Errorf("account not found: %d", account.Id)
	}
	if err != nil {
		return false, err
	}

	return accountModel.IsDirty, nil
}

// --- Link Management ---

func (c *BunClient) CreateLink(ctx context.Context, accountId client.AccountId, tagMatcher string, expiresAt time.Time) (client.Link, error) {
	// TODO: Link operations not yet fully implemented.
	// account_keys table doesn't have tagMatcher or expiresAt columns.
	// This is a stub implementation.
	return client.Link{
		Id:         client.LinkId(0), // TODO: Needs proper LinkId generation
		AccountId:  accountId,
		TagMatcher: tagMatcher,
		ExpiresAt:  expiresAt,
	}, errors.New("CreateLink: TODO - link operations not yet fully implemented")
}

func (c *BunClient) GetLink(ctx context.Context, id client.LinkId) (client.Link, error) {
	// TODO: Link operations not yet fully implemented.
	return client.Link{}, errors.New("GetLink: TODO - link operations not yet fully implemented")
}

func (c *BunClient) GetLinks(ctx context.Context, ids ...client.LinkId) ([]client.Link, error) {
	// TODO: Link operations not yet fully implemented.
	return nil, errors.New("GetLinks: TODO - link operations not yet fully implemented")
}

func (c *BunClient) ListLinksForAccount(ctx context.Context, accountId client.AccountId, expired bool) ([]client.Link, error) {
	km := core.DefaultKeyManager()
	if km == nil {
		return nil, errors.New("no key manager available")
	}

	// Get public keys assigned to this account.
	keys, err := km.GetKeysForAccount(int(accountId))
	if err != nil {
		return nil, fmt.Errorf("failed to get keys for account: %w", err)
	}

	// Convert to simplified Link objects (without tagMatcher/expiresAt).
	// TODO: Once account_keys table has tagMatcher/expiresAt columns, populate these fields.
	var result []client.Link
	for i := range keys {
		result = append(result, client.Link{
			Id:         client.LinkId(i + 1), // TODO: Use proper link IDs once schema supports them
			AccountId:  accountId,
			TagMatcher: "", // TODO: Not yet in schema
			ExpiresAt:  time.Time{},
		})
	}

	return result, nil
}

func (c *BunClient) ListLinksForPublicKey(ctx context.Context, publicKeyId client.PublicKeyId, expired bool) ([]client.Link, error) {
	km := core.DefaultKeyManager()
	if km == nil {
		return nil, errors.New("no key manager available")
	}

	// Get accounts that have this key assigned.
	accounts, err := km.GetAccountsForKey(int(publicKeyId))
	if err != nil {
		return nil, fmt.Errorf("failed to get accounts for key: %w", err)
	}

	// Convert to simplified Link objects.
	// TODO: Once account_keys table has tagMatcher/expiresAt columns, populate these fields.
	var result []client.Link
	for i, acc := range accounts {
		result = append(result, client.Link{
			Id:         client.LinkId(i + 1), // TODO: Use proper link IDs once schema supports them
			AccountId:  client.AccountId(acc.ID),
			TagMatcher: "", // TODO: Not yet in schema
			ExpiresAt:  time.Time{},
		})
	}

	return result, nil
}

func (c *BunClient) UpdateLink(ctx context.Context, id client.LinkId, accountId client.AccountId, tagMatcher string, expiresAt time.Time) (client.Link, error) {
	// TODO: Link operations not yet fully implemented.
	return client.Link{}, errors.New("UpdateLink: TODO - link operations not yet fully implemented")
}

func (c *BunClient) DeleteLinks(ctx context.Context, ids ...client.LinkId) error {
	// TODO: Link operations not yet fully implemented.
	return errors.New("DeleteLinks: TODO - link operations not yet fully implemented")
}

// --- Deploy & Verify ---

func (c *BunClient) DeployAccount(ctx context.Context, accountId client.AccountId) (chan client.DeployProgressAccount, error) {
	// TODO: Implement deployment streaming for single account.
	return nil, errors.New("DeployAccount: TODO - not yet implemented")
}

func (c *BunClient) DeployAccounts(ctx context.Context, accountIds ...client.AccountId) (chan client.DeployProgressAccounts, error) {
	// TODO: Implement deployment streaming for multiple accounts.
	return nil, errors.New("DeployAccounts: TODO - not yet implemented")
}

func (c *BunClient) VerifyAccount(ctx context.Context, accountId client.AccountId) (chan client.VerifyProgressAccount, error) {
	// TODO: Implement verification streaming for single account.
	return nil, errors.New("VerifyAccount: TODO - not yet implemented")
}

func (c *BunClient) VerifyAccounts(ctx context.Context, accountIds ...client.AccountId) (chan client.VerifyProgressAccounts, error) {
	// TODO: Implement verification streaming for multiple accounts.
	return nil, errors.New("VerifyAccounts: TODO - not yet implemented")
}

// --- Other Operations ---

func (c *BunClient) ListAuditLogs(ctx context.Context, limit int) ([]client.AuditLog, error) {
	// TODO: Implement audit log listing.
	return nil, errors.New("ListAuditLogs: TODO - not yet implemented")
}

func (c *BunClient) ListExistingTags(ctx context.Context) tags.Tags {
	// TODO: Implement tag listing from existing accounts/keys.
	return tags.Tags{}
}

func (c *BunClient) OnboardHost(ctx context.Context, host string, port int, accountUsername string, deploymentKey string) (chan client.OnboardHostProgress, error) {
	// TODO: Implement host onboarding with streaming progress.
	return nil, errors.New("OnboardHost: TODO - not yet implemented")
}

func (c *BunClient) DecommisionAccount(ctx context.Context, id client.AccountId) (chan client.DecommisionAccountProgress, error) {
	// TODO: Implement account decommissioning with streaming progress.
	return nil, errors.New("DecommisionAccount: TODO - not yet implemented")
}
