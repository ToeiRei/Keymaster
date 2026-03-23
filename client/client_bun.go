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
	"github.com/toeirei/keymaster/core/model"
	"github.com/toeirei/keymaster/core/security"
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
	// tag link management (in-memory)
	tagLinks map[ID]struct {
		accountID int
		filter    string
		expiresAt time.Time
	}
	nextTagLinkID ID
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
		tagLinks: make(map[ID]struct {
			accountID int
			filter    string
			expiresAt time.Time
		}),
		nextTagLinkID: 1,
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

func (c *BunClient) CreatePublicKey(ctx context.Context, key string, comment *string, tags []string) (PublicKey, error) {
	km := core.DefaultKeyManager()
	if km == nil {
		return PublicKey{}, errors.New("no key manager available")
	}
	// If the caller supplied a full authorized_keys line, parse it. Otherwise
	// treat the provided `key` string as the raw key data.
	alg, keyData, parsedComment, perr := sshkey.Parse(key)
	if perr != nil {
		// Not a full authorized_keys line; treat as raw key data.
		alg = ""
		keyData = key
	}

	_comment := ""
	if comment != nil {
		_comment = *comment
	} else if parsedComment != "" {
		_comment = parsedComment
	}

	pk, err := km.AddPublicKeyAndGetModel(alg, keyData, _comment, false, time.Time{})
	if err != nil {
		return PublicKey{}, err
	}
	return PublicKey{
		Id:        ID(pk.ID),
		Algorithm: alg,
		Data:      pk.KeyData,
		Comment:   &pk.Comment,
		Tags:      tags,
	}, nil
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
	for _, pk := range pks {
		if ID(pk.ID) == id {
			return PublicKey{ID(pk.ID), pk.Algorithm, pk.KeyData, &pk.Comment, nil}, // TODO where the tags at?
				nil
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
	for _, pk := range pks {
		for _, id := range ids {
			if ID(pk.ID) == id {
				out = append(out, PublicKey{ID(pk.ID), pk.Algorithm, pk.KeyData, &pk.Comment, nil}) // TODO where the tags at?
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
	for _, pk := range pks {
		// Tags are not modeled in core.PublicKey; return empty tags for now.
		out = append(out, PublicKey{ID(pk.ID), pk.Algorithm, pk.KeyData, &pk.Comment, nil}) // TODO where the tags at?
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
		if t.Port != port {
			t.Port = port
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
	if t.Host != target.Host {
		delete(c.hostToID, t.Host)
		c.hostToID[target.Host] = id
	}
	// Update stored target (including port)
	c.targetsByID[id] = Target{id, target.Host, target.Port}
	return nil
}

func (c *BunClient) DeleteTargets(ctx context.Context, ids ...ID) error {
	for _, id := range ids {
		if t, ok := c.targetsByID[id]; ok {
			delete(c.targetsByID, id)
			delete(c.hostToID, t.Host)
		}
	}
	return nil
}

// --- Account Management ---

func (c *BunClient) CreateAccount(ctx context.Context, targetID ID, name string, deploymentKey string) (Account, error) {
	// Resolve hostname from targetID
	var hostname string
	if t, ok := c.targetsByID[targetID]; ok {
		hostname = t.Host
	} else {
		return Account{}, errors.New("unknown target")
	}
	// Prefer package-level AccountManager when available so that created
	// accounts are visible to package-level helpers (key manager, deployer).
	if am := core.DefaultAccountManager(); am != nil {
		acctID, err := am.AddAccount(name, hostname, "", "")
		if err != nil {
			return Account{}, err
		}
		return Account{ID(acctID), targetID, name, deploymentKey, nil}, nil
	}
	if c.store == nil {
		return Account{}, errors.New("no store available")
	}
	acctID, err := c.store.AddAccount(name, hostname, "", "")
	if err != nil {
		return Account{}, err
	}
	return Account{ID(acctID), targetID, name, deploymentKey, nil}, nil
}

func (c *BunClient) GetAccount(ctx context.Context, id ID) (Account, error) {
	if c.store == nil {
		return Account{}, errors.New("no store available")
	}
	m, err := c.store.GetAccount(int(id))
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
		targetID = t.Id
	}
	return Account{ID(m.ID), targetID, m.Username, "", nil}, nil
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
	if c.store == nil {
		return nil, errors.New("no store available")
	}
	accounts, err := c.store.GetAccounts()
	if err != nil {
		return nil, err
	}
	var out []Account
	for _, m := range accounts {
		if m.Hostname == t.Host {
			// try to find targetID mapping (should match)
			out = append(out, Account{ID(m.ID), targetID, m.Username, "", nil})
		}
	}
	return out, nil
}

func (c *BunClient) DeleteAccounts(ctx context.Context, ids ...ID) error {
	for _, id := range ids {
		if c.store != nil {
			if err := c.store.DeleteAccount(int(id)); err != nil {
				return err
			}
		} else {
			return errors.New("no store available")
		}
	}
	return nil
}

func (c *BunClient) IsAccountDirty(ctx context.Context, a Account) (bool, error) {
	if c.store == nil {
		return false, errors.New("no store available")
	}
	account, err := c.store.GetAccount(int(a.Id))
	if err != nil {
		return false, err
	}
	return account.IsDirty, nil
}

func (c *BunClient) GetDirtyAccounts(ctx context.Context) ([]Account, error) {
	if c.store == nil {
		return nil, errors.New("no store available")
	}
	accounts, err := c.store.GetAccounts()
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
				targetID = t.Id
			}
			out = append(out, Account{ID(m.ID), targetID, m.Username, "", nil})
		}
	}
	return out, nil
}

// --- Tag to Account Management ---

// LinkTagAccount associates a tag filter (e.g. "device:mobile&company:telekom") with
// an `accountID` until `expiresAt`.
func (c *BunClient) LinkTagAccount(ctx context.Context, accountID ID, filter string, expiresAt time.Time) (ID, error) {
	id := c.nextTagLinkID
	c.nextTagLinkID++
	c.tagLinks[id] = struct {
		accountID int
		filter    string
		expiresAt time.Time
	}{accountID: int(accountID), filter: filter, expiresAt: expiresAt}
	return id, nil
}

// UnLinkTagAccount removes previously created tag-account links.
func (c *BunClient) UnLinkTagAccount(ctx context.Context, linkIDs ...ID) error {
	for _, id := range linkIDs {
		delete(c.tagLinks, id)
	}
	return nil
}

// ResolvePublicKeysForAccount returns public keys applicable to `accountID`.
func (c *BunClient) ResolvePublicKeysForAccount(ctx context.Context, accountID ID) ([]PublicKey, error) {
	km := core.DefaultKeyManager()
	if km == nil {
		return nil, errors.New("no key manager available")
	}
	// Keys explicitly assigned to account
	assigned, err := km.GetKeysForAccount(int(accountID))
	if err != nil {
		return nil, err
	}
	// Global keys
	global, err := km.GetGlobalPublicKeys()
	if err != nil {
		return nil, err
	}
	// Merge unique by ID
	seen := make(map[int]bool)
	var out []PublicKey
	for _, pk := range global {
		seen[pk.ID] = true
		out = append(out, PublicKey{ID(pk.ID), pk.Algorithm, pk.KeyData, &pk.Comment, nil}) // TODO where the tags at?
	}
	for _, pk := range assigned {
		if !seen[pk.ID] {
			seen[pk.ID] = true
			out = append(out, PublicKey{ID(pk.ID), pk.Algorithm, pk.KeyData, &pk.Comment, nil}) // TODO where the tags at?
		}
	}
	return out, nil
}

// ResolveAccountsForPublicKey returns accounts that a public key applies to.
func (c *BunClient) ResolveAccountsForPublicKey(ctx context.Context, publicKeyID ID) ([]Account, error) {
	km := core.DefaultKeyManager()
	if km == nil {
		return nil, errors.New("no key manager available")
	}
	accs, err := km.GetAccountsForKey(int(publicKeyID))
	if err != nil {
		return nil, err
	}
	// Fallback: if direct reverse lookup returned no accounts (some adapters
	// may not implement it), scan all accounts and check their assigned keys.
	if len(accs) == 0 {
		if c.store != nil {
			all, aerr := c.store.GetAccounts()
			if aerr == nil {
				for _, am := range all {
					keysFor, kerr := km.GetKeysForAccount(am.ID)
					if kerr != nil {
						continue
					}
					for _, k := range keysFor {
						if k.ID == int(publicKeyID) {
							accs = append(accs, am)
							break
						}
					}
				}
			}
		}
	}
	var out []Account
	for _, a := range accs {
		// ensure target exists
		var tid ID
		if id, ok := c.hostToID[a.Hostname]; ok {
			tid = id
		} else {
			t, terr := c.CreateTarget(ctx, a.Hostname, 22)
			if terr != nil {
				return nil, terr
			}
			tid = t.Id
		}
		out = append(out, Account{ID(a.ID), tid, a.Username, "", nil})
	}
	return out, nil
}

// --- Onboarding & Decommision ---

// OnboardHost starts onboarding of a host and returns a progress channel.
func (c *BunClient) OnboardHost(ctx context.Context, host string, port int /* , gateway string, plugin string */, accountName string, deploymentKey string) (chan OnboardHostProgress, error) {
	ch := make(chan OnboardHostProgress, 2)
	go func() {
		defer close(ch)
		ch <- OnboardHostProgress{Percent: 0}

		var pk security.Secret
		if deploymentKey != "" {
			pk = security.FromBytes([]byte(deploymentKey))
		}

		params := core.BootstrapParams{
			Username:       accountName,
			Hostname:       host,
			Label:          "",
			Tags:           "",
			SelectedKeyIDs: nil,
			TempPrivateKey: pk,
		}

		deps := core.BootstrapDeps{
			AddAccount: func(username, hostname, label, tags string) (int, error) {
				return c.store.AddAccount(username, hostname, label, tags)
			},
			DeleteAccount: func(accountID int) error {
				return c.store.DeleteAccount(accountID)
			},
			AssignKey: func(keyID, accountID int) error {
				return c.store.AssignKeyToAccount(keyID, accountID)
			},
			GenerateKeysContent: core.GenerateKeysContent,
			NewBootstrapDeployer: func(hostname, username string, privateKey interface{}, expectedHostKey string) (core.BootstrapDeployer, error) {
				var sec security.Secret
				if pk, ok := privateKey.(security.Secret); ok {
					sec = pk
				}
				d, err := core.NewBootstrapDeployer(hostname, username, sec, expectedHostKey)
				return d, err
			},
			GetActiveSystemKey: func() (*model.SystemKey, error) {
				return c.store.GetActiveSystemKey()
			},
			LogAudit: func(e core.BootstrapAuditEvent) error { return nil },
		}

		_, _ = core.PerformBootstrapDeployment(ctx, params, deps)
		ch <- OnboardHostProgress{Percent: 100}
	}()
	return ch, nil
}

// DecommisionTarget decommissions a deployment target and streams progress.
func (c *BunClient) DecommisionTarget(ctx context.Context, id ID) (chan DecommisionTargetProgress, error) {
	ch := make(chan DecommisionTargetProgress, 2)
	go func() {
		defer close(ch)
		ch <- DecommisionTargetProgress{Percent: 0}

		t, ok := c.targetsByID[id]
		if !ok {
			ch <- DecommisionTargetProgress{Percent: 100}
			return
		}

		accountsModel, err := c.store.GetAllActiveAccounts()
		if err != nil {
			ch <- DecommisionTargetProgress{Percent: 100}
			return
		}
		var targets []model.Account
		for _, a := range accountsModel {
			if a.Hostname == t.Host {
				targets = append(targets, a)
			}
		}

		_, _ = core.DecommissionAccounts(ctx, targets, core.DecommissionOptions{}, core.DefaultDeployerManager, c.store, nil)
		ch <- DecommisionTargetProgress{Percent: 100}
	}()
	return ch, nil
}

// DecommisionAccount decommissions an account and streams progress.
func (c *BunClient) DecommisionAccount(ctx context.Context, id ID) (chan DecommisionAccountProgress, error) {
	ch := make(chan DecommisionAccountProgress, 2)
	go func() {
		defer close(ch)
		ch <- DecommisionAccountProgress{Percent: 0}

		m, err := c.store.GetAccount(int(id))
		if err != nil || m == nil {
			ch <- DecommisionAccountProgress{Percent: 100}
			return
		}
		targets := []model.Account{*m}
		_, _ = core.DecommissionAccounts(ctx, targets, core.DecommissionOptions{}, core.DefaultDeployerManager, c.store, nil)
		ch <- DecommisionAccountProgress{Percent: 100}
	}()
	return ch, nil
}

// --- Deploy stuff ---

// DeployPublicKeys deploys public keys to their target accounts and reports
// progress on the returned channel.
func (c *BunClient) DeployPublicKeys(ctx context.Context, publicKeyID ...ID) (chan DeployProgress, error) {
	ch := make(chan DeployProgress, 10)
	go func() {
		defer close(ch)
		ch <- DeployProgress{Percent: 0}

		var allAccounts []model.Account
		for _, pid := range publicKeyID {
			accounts, err := c.ResolveAccountsForPublicKey(ctx, pid)
			if err != nil {
				continue
			}
			for _, a := range accounts {
				if m, err := c.store.GetAccount(int(a.Id)); err == nil && m != nil {
					allAccounts = append(allAccounts, *m)
				}
			}
		}
		total := len(allAccounts)
		for i, acc := range allAccounts {
			_ = core.DefaultDeployerManager.DeployForAccount(acc, false)
			percent := float32(i+1) / float32(total) * 100
			ch <- DeployProgress{Percent: percent}
		}
		if total == 0 {
			ch <- DeployProgress{Percent: 100}
		}
	}()
	return ch, nil
}

// DeployTargets deploys to the specified target ids and streams progress.
func (c *BunClient) DeployTargets(ctx context.Context, targetID ...ID) (chan DeployProgress, error) {
	ch := make(chan DeployProgress, 10)
	go func() {
		defer close(ch)
		ch <- DeployProgress{Percent: 0}

		var allAccounts []model.Account
		for _, tid := range targetID {
			accounts, err := c.ListAccountsByTarget(ctx, tid)
			if err != nil {
				continue
			}
			for _, a := range accounts {
				if m, err := c.store.GetAccount(int(a.Id)); err == nil && m != nil {
					allAccounts = append(allAccounts, *m)
				}
			}
		}
		total := len(allAccounts)
		for i, acc := range allAccounts {
			_ = core.DefaultDeployerManager.DeployForAccount(acc, false)
			percent := float32(i+1) / float32(total) * 100
			ch <- DeployProgress{Percent: percent}
		}
		if total == 0 {
			ch <- DeployProgress{Percent: 100}
		}
	}()
	return ch, nil
}

// DeployAccounts deploys to the specified account ids and streams progress.
func (c *BunClient) DeployAccounts(ctx context.Context, accountID ...ID) (chan DeployProgress, error) {
	ch := make(chan DeployProgress, 10)
	go func() {
		defer close(ch)
		ch <- DeployProgress{Percent: 0}

		var allAccounts []model.Account
		for _, aid := range accountID {
			if m, err := c.store.GetAccount(int(aid)); err == nil && m != nil {
				allAccounts = append(allAccounts, *m)
			}
		}
		total := len(allAccounts)
		for i, acc := range allAccounts {
			_ = core.DefaultDeployerManager.DeployForAccount(acc, false)
			percent := float32(i+1) / float32(total) * 100
			ch <- DeployProgress{Percent: percent}
		}
		if total == 0 {
			ch <- DeployProgress{Percent: 100}
		}
	}()
	return ch, nil
}

// DeployAll triggers deployment for all pending targets/accounts.
func (c *BunClient) DeployAll(ctx context.Context) (chan DeployProgress, error) {
	ch := make(chan DeployProgress, 10)
	go func() {
		defer close(ch)
		ch <- DeployProgress{Percent: 0}
		res, _ := core.DeployAccounts(ctx, c.store, core.DefaultDeployerManager, nil, nil)
		total := len(res)
		for i := range res {
			percent := float32(i+1) / float32(total) * 100
			ch <- DeployProgress{Percent: percent}
		}
		if total == 0 {
			ch <- DeployProgress{Percent: 100}
		}
	}()
	return ch, nil
}
