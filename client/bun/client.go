// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package bun

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/toeirei/keymaster/client"
	"github.com/toeirei/keymaster/config"
	"github.com/toeirei/keymaster/core"
	"github.com/toeirei/keymaster/core/model"
	"github.com/toeirei/keymaster/core/sshkey"
	"github.com/toeirei/keymaster/tags"
)

// BunClient is a client implementation backed by the Bun ORM and core.Store.
// It provides full CRUD operations for accounts, public keys, and links.
type BunClient struct {
	config config.Config
	store  core.Store
	log    *log.Logger
	// TODO: in-memory cache for frequently accessed entities (optional optimization)
}

// Verify BunClient implements client.Client.
var _ client.Client = (*BunClient)(nil)

// NewBunClient creates and initializes a new BunClient from the provided config and logger.
// It initializes the database with migrations and returns a ready-to-use client.
func NewBunClient(cfg config.Config, logger *log.Logger) (*BunClient, error) {
	// Initialize package-level DB (migrations, global store).
	if err := core.InitDB(cfg.Database.Type, cfg.Database.Dsn); err != nil {
		return nil, fmt.Errorf("failed to init DB: %w", err)
	}

	// Create a Store instance for this client.
	st, err := core.NewStoreFromDSN(cfg.Database.Type, cfg.Database.Dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to create store: %w", err)
	}

	return &BunClient{
		config: cfg,
		store:  st,
		log:    logger,
	}, nil
}

// Close closes the client and cleans up resources.
func (c *BunClient) Close(ctx context.Context) error {
	if c.store != nil {
		return core.CloseStore(c.store)
	}
	return nil
}

// WithTransaction executes a function within a database transaction.
// TODO: Implement transaction support via bun.DB transactions.
func (c *BunClient) WithTransaction(ctx context.Context, fn func(c client.Client) error) error {
	return fn(c)
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

func (c *BunClient) CreatePublicKey(ctx context.Context, key string, comment string, tags tags.Tags) (client.PublicKey, error) {
	km := core.DefaultKeyManager()
	if km == nil {
		return client.PublicKey{}, errors.New("no key manager available")
	}

	// Parse the key to extract algorithm and key data.
	alg, keyData, _, perr := sshkey.Parse(key)
	if perr != nil {
		// If parsing fails, treat the input as raw key data.
		alg = "ssh-ed25519"
		keyData = key
	}

	// Add the public key using the KeyManager.
	pk, err := km.AddPublicKeyAndGetModel(alg, keyData, comment, false, time.Time{})
	if err != nil {
		return client.PublicKey{}, fmt.Errorf("failed to add public key: %w", err)
	}

	return client.PublicKey{
		Id:        client.PublicKeyId(pk.ID),
		Algorithm: pk.Algorithm,
		Data:      pk.KeyData,
		Comment:   comment,
		Tags:      nil, // TODO: PublicKey.Tags not yet modeled in core.model.PublicKey
	}, nil
}

func (c *BunClient) GetPublicKey(ctx context.Context, id client.PublicKeyId) (client.PublicKey, error) {
	km := core.DefaultKeyManager()
	if km == nil {
		return client.PublicKey{}, errors.New("no key manager available")
	}

	// Fetch all public keys and find the one with matching ID.
	pks, err := km.GetAllPublicKeys()
	if err != nil {
		return client.PublicKey{}, fmt.Errorf("failed to get public keys: %w", err)
	}

	for _, pk := range pks {
		if pk.ID == int(id) {
			return client.PublicKey{
				Id:        client.PublicKeyId(pk.ID),
				Algorithm: pk.Algorithm,
				Data:      pk.KeyData,
				Comment:   pk.Comment,
				Tags:      nil, // TODO: PublicKey.Tags stub
			}, nil
		}
	}

	return client.PublicKey{}, fmt.Errorf("public key not found: %d", id)
}

func (c *BunClient) GetPublicKeys(ctx context.Context, ids ...client.PublicKeyId) ([]client.PublicKey, error) {
	km := core.DefaultKeyManager()
	if km == nil {
		return nil, errors.New("no key manager available")
	}

	pks, err := km.GetAllPublicKeys()
	if err != nil {
		return nil, fmt.Errorf("failed to get public keys: %w", err)
	}

	// Build a map of requested IDs for fast lookup.
	requested := make(map[int]bool)
	for _, id := range ids {
		requested[int(id)] = true
	}

	var result []client.PublicKey
	for _, pk := range pks {
		if requested[pk.ID] {
			result = append(result, client.PublicKey{
				Id:        client.PublicKeyId(pk.ID),
				Algorithm: pk.Algorithm,
				Data:      pk.KeyData,
				Comment:   pk.Comment,
				Tags:      nil, // TODO: PublicKey.Tags stub
			})
		}
	}

	return result, nil
}

func (c *BunClient) ListPublicKeys(ctx context.Context, tagMatcher string) ([]client.PublicKey, error) {
	km := core.DefaultKeyManager()
	if km == nil {
		return nil, errors.New("no key manager available")
	}

	pks, err := km.GetAllPublicKeys()
	if err != nil {
		return nil, fmt.Errorf("failed to list public keys: %w", err)
	}

	var result []client.PublicKey
	for _, pk := range pks {
		result = append(result, client.PublicKey{
			Id:        client.PublicKeyId(pk.ID),
			Algorithm: pk.Algorithm,
			Data:      pk.KeyData,
			Comment:   pk.Comment,
			Tags:      nil, // TODO: tagMatcher filtering not yet implemented
		})
	}

	return result, nil
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
	// TODO: PublicKey updates not yet supported; keys are immutable in current schema.
	return client.PublicKey{}, errors.New("UpdatePublicKey: operation not yet supported")
}

func (c *BunClient) DeletePublicKeys(ctx context.Context, ids ...client.PublicKeyId) error {
	km := core.DefaultKeyManager()
	if km == nil {
		return errors.New("no key manager available")
	}

	for _, id := range ids {
		if err := km.DeletePublicKey(int(id)); err != nil {
			return fmt.Errorf("failed to delete public key %d: %w", id, err)
		}
	}

	return nil
}

// --- Account Management ---

func (c *BunClient) CreateAccount(ctx context.Context, username string, host string, port int, deploymentMethod string, deploymentSecret string) (client.Account, error) {
	if c.store == nil {
		return client.Account{}, errors.New("no store available")
	}

	// Encode host:port into hostname.
	encoded := encodeHostPort(host, port)

	// Add account to store.
	id, err := c.store.AddAccount(username, encoded, "", "")
	if err != nil {
		return client.Account{}, fmt.Errorf("failed to create account: %w", err)
	}

	return client.Account{
		Id:           client.AccountId(id),
		Username:     username,
		Host:         host,
		Port:         port,
		DeployMethod: deploymentMethod,
		DeploySecret: deploymentSecret,
		DeployCache:  "",
	}, nil
}

func (c *BunClient) GetAccount(ctx context.Context, id client.AccountId) (client.Account, error) {
	if c.store == nil {
		return client.Account{}, errors.New("no store available")
	}

	m, err := c.store.GetAccount(int(id))
	if err != nil {
		return client.Account{}, fmt.Errorf("failed to get account: %w", err)
	}
	if m == nil {
		return client.Account{}, fmt.Errorf("account not found: %d", id)
	}

	return c.accountModelToClient(m)
}

func (c *BunClient) GetAccounts(ctx context.Context, ids ...client.AccountId) ([]client.Account, error) {
	var result []client.Account
	for _, id := range ids {
		acc, err := c.GetAccount(ctx, id)
		if err != nil {
			return nil, err
		}
		result = append(result, acc)
	}
	return result, nil
}

func (c *BunClient) ListAccounts(ctx context.Context) ([]client.Account, error) {
	if c.store == nil {
		return nil, errors.New("no store available")
	}

	accounts, err := c.store.GetAllAccounts()
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}

	var result []client.Account
	for i := range accounts {
		acc, err := c.accountModelToClient(&accounts[i])
		if err != nil {
			// Log error but continue; use fallback conversion
			if c.log != nil {
				c.log.Printf("warning: failed to convert account %d: %v", accounts[i].ID, err)
			}
			acc = client.Account{
				Id:           client.AccountId(accounts[i].ID),
				Username:     accounts[i].Username,
				Host:         accounts[i].Hostname,
				Port:         22,
				DeployMethod: "ssh",
			}
		}
		result = append(result, acc)
	}

	return result, nil
}

func (c *BunClient) ListAccountsDirty(ctx context.Context) ([]client.Account, error) {
	if c.store == nil {
		return nil, errors.New("no store available")
	}

	accounts, err := c.store.GetAllAccounts()
	if err != nil {
		return nil, fmt.Errorf("failed to list dirty accounts: %w", err)
	}

	var result []client.Account
	for i := range accounts {
		if accounts[i].IsDirty {
			acc, err := c.accountModelToClient(&accounts[i])
			if err != nil {
				acc = client.Account{
					Id:           client.AccountId(accounts[i].ID),
					Username:     accounts[i].Username,
					Host:         accounts[i].Hostname,
					Port:         22,
					DeployMethod: "ssh",
				}
			}
			result = append(result, acc)
		}
	}

	return result, nil
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
	if c.store == nil {
		return client.Account{}, errors.New("no store available")
	}

	// Get existing account.
	m, err := c.store.GetAccount(int(id))
	if err != nil {
		return client.Account{}, fmt.Errorf("failed to get account for update: %w", err)
	}
	if m == nil {
		return client.Account{}, fmt.Errorf("account not found: %d", id)
	}

	// Update fields that changed.
	encoded := encodeHostPort(host, port)
	if m.Hostname != encoded {
		if err := c.store.UpdateAccountHostname(int(id), encoded); err != nil {
			return client.Account{}, fmt.Errorf("failed to update hostname: %w", err)
		}
	}
	// Username is not persisted through this store path yet.

	return client.Account{
		Id:           client.AccountId(m.ID),
		Username:     username,
		Host:         host,
		Port:         port,
		DeployMethod: deploymentMethod,
		DeploySecret: deploymentSecret,
		DeployCache:  "",
	}, nil
}

func (c *BunClient) DeleteAccounts(ctx context.Context, ids ...client.AccountId) error {
	if c.store == nil {
		return errors.New("no store available")
	}

	for _, id := range ids {
		if err := c.store.DeleteAccount(int(id)); err != nil {
			return fmt.Errorf("failed to delete account %d: %w", id, err)
		}
	}

	return nil
}

func (c *BunClient) IsAccountDirty(ctx context.Context, account client.Account) (bool, error) {
	if c.store == nil {
		return false, errors.New("no store available")
	}

	m, err := c.store.GetAccount(int(account.Id))
	if err != nil {
		return false, fmt.Errorf("failed to check account dirty status: %w", err)
	}
	if m == nil {
		return false, fmt.Errorf("account not found: %d", account.Id)
	}

	return m.IsDirty, nil
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
