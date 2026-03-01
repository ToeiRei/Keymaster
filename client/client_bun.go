// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package client

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/toeirei/keymaster/config"
	"github.com/toeirei/keymaster/core"
	"github.com/toeirei/keymaster/core/sshkey"
)

type BunClient struct {
	//lint:ignore U1000 Placeholder for future configuration wiring.
	config config.Config
	//lint:ignore U1000 Placeholder for future store wiring.
	store core.Store
	// NOTE:
	// log != audit_log
	// log is not meant for cli out
	//lint:ignore U1000 Placeholder for future logging wiring.
	log *log.Logger
	// in-memory mapping for UI-level Targets
	hostToID     map[string]ID
	targetsByID  map[ID]Target
	nextTargetID ID
}

// *BunClient implements Client
var _ Client = (*BunClient)(nil)

// --- Lifecycle & Initialization ---

// New creates and initializes a new `BunClient` from the provided `Config` and
// `logger`. The implementation should connect to the backing store, run any
// migrations and return a ready-to-use client. Currently unimplemented.
func NewBunClient(config config.Config, logger *log.Logger) (*BunClient, error) {
	// Initialize package-level DB (migrations, global store) so core/deploy
	// wiring that relies on package-level adapters works the same as the CLI.
	if err := core.InitDB(config.Database.Type, config.Database.Dsn); err != nil {
		return nil, err
	}

	// Also create a wrapped core.Store instance we can use without relying on
	// package globals. NewStoreFromDSN returns a core.Store wrapper around
	// the underlying DB implementation.
	st, err := core.NewStoreFromDSN(config.Database.Type, config.Database.Dsn)
	if err != nil {
		return nil, err
	}

	return &BunClient{
		config:       config,
		log:          logger,
		store:        st,
		hostToID:     make(map[string]ID),
		targetsByID:  make(map[ID]Target),
		nextTargetID: 1,
	}, nil
}

// Close cleans up resources held by the client and closes any open
// connections. Calls should pass a context for cancellation/timeouts.
func (c *BunClient) Close(ctx context.Context) error {
	// Attempt to close any store resources created via core.NewStoreFromDSN.
	if c.store != nil {
		return core.CloseStore(c.store)
	}
	return nil
}

// --- PublicKey Management ---

func (c *BunClient) CreatePublicKey(ctx context.Context, identity string, tags []string) (PublicKey, error) {
	km := core.DefaultKeyManager()
	if km == nil {
		return PublicKey{}, errors.New("no key manager available")
	}

	// Generate a new keypair and store the public key via the KeyManager.
	pubLine, _, err := core.DefaultKeyGenerator().GenerateAndMarshalEd25519Key(identity, "")
	if err != nil {
		return PublicKey{}, err
	}
	alg, keyData, comment, perr := sshkey.Parse(pubLine)
	if perr != nil {
		return PublicKey{}, perr
	}
	pk, err := km.AddPublicKeyAndGetModel(alg, keyData, comment, false, time.Time{})
	if err != nil {
		return PublicKey{}, err
	}
	return PublicKey{ID(pk.ID), pk.Comment, tags}, nil
}

func (c *BunClient) GetPublicKey(ctx context.Context, id ID) (PublicKey, error) {
	km := core.DefaultKeyManager()
	if km == nil {
		return PublicKey{}, errors.New("no key manager available")
	}
	pks, err := km.GetAllPublicKeys()
	if err != nil {
		return PublicKey{}, err
	}
	for _, p := range pks {
		if ID(p.ID) == id {
			return PublicKey{ID(p.ID), p.Comment, nil}, nil
		}
	}
	return PublicKey{}, errors.New("public key not found")
}

func (c *BunClient) GetPublicKeys(ctx context.Context, ids ...ID) ([]PublicKey, error) {
	km := core.DefaultKeyManager()
	if km == nil {
		return nil, errors.New("no key manager available")
	}
	pks, err := km.GetAllPublicKeys()
	if err != nil {
		return nil, err
	}
	var out []PublicKey
	for _, p := range pks {
		for _, id := range ids {
			if ID(p.ID) == id {
				out = append(out, PublicKey{ID(p.ID), p.Comment, nil})
			}
		}
	}
	return out, nil
}

func (c *BunClient) ListPublicKeys(ctx context.Context, tagFilter string) ([]PublicKey, error) {
	km := core.DefaultKeyManager()
	if km == nil {
		return nil, errors.New("no key manager available")
	}
	pks, err := km.GetAllPublicKeys()
	if err != nil {
		return nil, err
	}
	var out []PublicKey
	for _, p := range pks {
		// Tags are not modeled in core.PublicKey; return empty tags for now.
		out = append(out, PublicKey{ID(p.ID), p.Comment, nil})
	}
	return out, nil
}

func (c *BunClient) UpdatePublicKeyTags(ctx context.Context, id ID, tags []string) error {
	// Tags are a client/UI-level concept for now; no-op.
	return nil
}

func (c *BunClient) DeletePublicKeys(ctx context.Context, ids ...ID) error {
	km := core.DefaultKeyManager()
	if km == nil {
		return errors.New("no key manager available")
	}
	for _, id := range ids {
		if err := km.DeletePublicKey(int(id)); err != nil {
			return err
		}
	}
	return nil
}

// --- Target Management ---

func (c *BunClient) CreateTarget(ctx context.Context, host string, port int /* , gateway string, plugin string */) (Target, error) {
	if id, ok := c.hostToID[host]; ok {
		t := c.targetsByID[id]
		// update port if changed
		if t.port != port {
			t.port = port
			c.targetsByID[id] = t
		}
		return t, nil
	}
	id := c.nextTargetID
	c.nextTargetID++
	t := Target{id, host, port}
	c.hostToID[host] = id
	c.targetsByID[id] = t
	return t, nil
}

func (c *BunClient) GetTarget(ctx context.Context, id ID) (Target, error) {
	if t, ok := c.targetsByID[id]; ok {
		return t, nil
	}
	return Target{}, errors.New("target not found")
}

func (c *BunClient) GetTargets(ctx context.Context, ids ...ID) ([]Target, error) {
	var out []Target
	for _, id := range ids {
		if t, ok := c.targetsByID[id]; ok {
			out = append(out, t)
		}
	}
	return out, nil
}

func (c *BunClient) ListTargets(ctx context.Context) ([]Target, error) {
	// Seed targets from existing accounts if none known yet.
	if len(c.targetsByID) == 0 {
		accounts, err := core.GetAllAccounts()
		if err == nil {
			for _, a := range accounts {
				if _, ok := c.hostToID[a.Hostname]; !ok {
					id := c.nextTargetID
					c.nextTargetID++
					t := Target{id, a.Hostname, 22}
					c.hostToID[a.Hostname] = id
					c.targetsByID[id] = t
				}
			}
		}
	}
	out := make([]Target, 0, len(c.targetsByID))
	for _, t := range c.targetsByID {
		out = append(out, t)
	}
	return out, nil
}

func (c *BunClient) UpdateTarget(ctx context.Context, id ID, target Target) error {
	t, ok := c.targetsByID[id]
	if !ok {
		return errors.New("target not found")
	}
	// If host changed, update host->id mapping
	if t.host != target.host {
		delete(c.hostToID, t.host)
		c.hostToID[target.host] = id
	}
	// Update stored target (including port)
	c.targetsByID[id] = Target{id, target.host, target.port}
	return nil
}

func (c *BunClient) DeleteTargets(ctx context.Context, ids ...ID) error {
	for _, id := range ids {
		if t, ok := c.targetsByID[id]; ok {
			delete(c.targetsByID, id)
			delete(c.hostToID, t.host)
		}
	}
	return nil
}

// --- Account Management ---

func (c *BunClient) CreateAccount(ctx context.Context, targetID ID, name string, deploymentKey string) (Account, error) {
	// Resolve hostname from targetID
	var hostname string
	if t, ok := c.targetsByID[targetID]; ok {
		hostname = t.host
	} else {
		return Account{}, errors.New("unknown target")
	}
	am := core.DefaultAccountManager()
	if am == nil {
		return Account{}, errors.New("no account manager available")
	}
	// label and tags are UI-level; pass empty
	acctID, err := am.AddAccount(name, hostname, "", "")
	if err != nil {
		return Account{}, err
	}
	return Account{ID(acctID), targetID, name, deploymentKey}, nil
}

func (c *BunClient) GetAccount(ctx context.Context, id ID) (Account, error) {
	m, err := core.GetAccount(int(id))
	if err != nil {
		return Account{}, err
	}
	if m == nil {
		return Account{}, errors.New("account not found")
	}
	// Ensure target exists in memory
	var targetID ID
	if idt, ok := c.hostToID[m.Hostname]; ok {
		targetID = idt
	} else {
		// create a new target entry with default port 22
		t, terr := c.CreateTarget(ctx, m.Hostname, 22)
		if terr != nil {
			return Account{}, terr
		}
		targetID = t.id
	}
	return Account{ID(m.ID), targetID, m.Username, ""}, nil
}

func (c *BunClient) GetAccounts(ctx context.Context, ids ...ID) ([]Account, error) {
	var out []Account
	for _, id := range ids {
		a, err := c.GetAccount(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}

func (c *BunClient) ListAccountsByTarget(ctx context.Context, targetID ID) ([]Account, error) {
	t, ok := c.targetsByID[targetID]
	if !ok {
		return nil, errors.New("target not found")
	}
	accounts, err := core.GetAccounts()
	if err != nil {
		return nil, err
	}
	var out []Account
	for _, m := range accounts {
		if m.Hostname == t.host {
			// try to find targetID mapping (should match)
			out = append(out, Account{ID(m.ID), targetID, m.Username, ""})
		}
	}
	return out, nil
}

func (c *BunClient) DeleteAccounts(ctx context.Context, ids ...ID) error {
	for _, id := range ids {
		if err := core.DeleteAccount(int(id)); err != nil {
			return err
		}
	}
	return nil
}

func (c *BunClient) GetDirtyAccounts(ctx context.Context) ([]Account, error) {
	accounts, err := core.GetAccounts()
	if err != nil {
		return nil, err
	}
	var out []Account
	for _, m := range accounts {
		if m.IsDirty {
			// ensure target mapping
			var targetID ID
			if idt, ok := c.hostToID[m.Hostname]; ok {
				targetID = idt
			} else {
				t, terr := c.CreateTarget(ctx, m.Hostname, 22)
				if terr != nil {
					return nil, terr
				}
				targetID = t.id
			}
			out = append(out, Account{ID(m.ID), targetID, m.Username, ""})
		}
	}
	return out, nil
}

// --- Tag to Account Management ---

// LinkTagAccount associates a tag filter (e.g. "device:mobile&company:telekom") with
// an `accountID` until `expiresAt`.
func (c *BunClient) LinkTagAccount(ctx context.Context, accountID ID, filter string, expiresAt time.Time) (ID, error) {
	return 0, errors.New("client.LinkTagAccount not implemented")
}

// UnLinkTagAccount removes previously created tag-account links.
func (c *BunClient) UnLinkTagAccount(ctx context.Context, linkIDs ...ID) error {
	return errors.New("client.UnLinkTagAccount not implemented")
}

// ResolvePublicKeysForAccount returns public keys applicable to `accountID`.
func (c *BunClient) ResolvePublicKeysForAccount(ctx context.Context, accountID ID) ([]PublicKey, error) {
	return nil, errors.New("client.ResolvePublicKeysForAccount not implemented")
}

// ResolveAccountsForPublicKey returns accounts that a public key applies to.
func (c *BunClient) ResolveAccountsForPublicKey(ctx context.Context, publicKeyID ID) ([]Account, error) {
	return nil, errors.New("client.ResolveAccountsForPublicKey not implemented")
}

// --- Onboarding & Decommision ---

// OnboardHost starts onboarding of a host and returns a progress channel.
func (c *BunClient) OnboardHost(ctx context.Context, host string, port int /* , gateway string, plugin string */, accountName string, deploymentKey string) (chan OnboardHostProgress, error) {
	return nil, errors.New("client.OnboardHost not implemented")
}

// DecommisionTarget decommissions a deployment target and streams progress.
func (c *BunClient) DecommisionTarget(ctx context.Context, id ID) (chan DecommisionTargetProgress, error) {
	return nil, errors.New("client.DecommisionTarget not implemented")
}

// DecommisionAccount decommissions an account and streams progress.
func (c *BunClient) DecommisionAccount(ctx context.Context, id ID) (chan DecommisionAccountProgress, error) {
	return nil, errors.New("client.DecommisionAccount not implemented")
}

// --- Deploy stuff ---

// DeployPublicKeys deploys public keys to their target accounts and reports
// progress on the returned channel.
func (c *BunClient) DeployPublicKeys(ctx context.Context, publicKeyID ...ID) (chan DeployProgress, error) {
	return nil, errors.New("client.DeployPublicKeys not implemented")
}

// DeployTargets deploys to the specified target ids and streams progress.
func (c *BunClient) DeployTargets(ctx context.Context, targetID ...ID) (chan DeployProgress, error) {
	return nil, errors.New("client.DeployTargets not implemented")
}

// DeployAccounts deploys to the specified account ids and streams progress.
func (c *BunClient) DeployAccounts(ctx context.Context, accountID ...ID) (chan DeployProgress, error) {
	return nil, errors.New("client.DeployAccounts not implemented")
}

// DeployAll triggers deployment for all pending targets/accounts.
func (c *BunClient) DeployAll(ctx context.Context) (chan DeployProgress, error) {
	return nil, errors.New("client.DeployAll not implemented")
}
