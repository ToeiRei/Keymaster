// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package mock

import (
	"context"
	"time"

	"github.com/toeirei/keymaster/client"
	"github.com/toeirei/keymaster/tags"
)

type Client struct {
	Overwrites ClientOverwrites
	BaseClient client.Client
	Pre        func(method string, args map[string]any)
}

type ClientOverwrites struct {
	// --- Lifecycle & Initialization ---
	Close           func(ctx context.Context) error
	WithTransaction func(ctx context.Context, fn func(c client.Client) error) error

	// --- PublicKey Management ---
	CreatePublicKey               func(ctx context.Context, key string, comment string, tags tags.Tags) (client.PublicKey, error)
	GetPublicKey                  func(ctx context.Context, id client.PublicKeyId) (client.PublicKey, error)
	GetPublicKeys                 func(ctx context.Context, ids ...client.PublicKeyId) ([]client.PublicKey, error)
	ListPublicKeys                func(ctx context.Context, tagMatcher string) ([]client.PublicKey, error)
	ListPublicKeysLinkedToAccount func(ctx context.Context, accountId client.AccountId, expired bool) ([]client.PublicKey, error)
	UpdatePublicKey               func(ctx context.Context, id client.PublicKeyId, comment string, tags tags.Tags) error
	DeletePublicKeys              func(ctx context.Context, ids ...client.PublicKeyId) error

	// --- Account Management ---
	CreateAccount                 func(ctx context.Context, username string, host string, port int, deploymentMethod string, deploymentSecret string) (client.Account, error)
	GetAccount                    func(ctx context.Context, id client.AccountId) (client.Account, error)
	GetAccounts                   func(ctx context.Context, ids ...client.AccountId) ([]client.Account, error)
	ListAccounts                  func(ctx context.Context) ([]client.Account, error)
	ListAccountsDirty             func(ctx context.Context) ([]client.Account, error)
	ListAccountsLinkedToPublicKey func(ctx context.Context, publicKeyId client.PublicKeyId, expired bool) ([]client.Account, error)
	UpdateAccount                 func(ctx context.Context, id client.AccountId, username string, host string, port int, deploymentMethod string, deploymentSecret string) error
	DeleteAccounts                func(ctx context.Context, ids ...client.AccountId) error
	IsAccountDirty                func(ctx context.Context, account client.Account) (bool, error)

	// --- Link Management ---
	CreateLink            func(ctx context.Context, accountId client.AccountId, tagMatcher string, expiresAt time.Time) (client.Link, error)
	GetLink               func(ctx context.Context, id client.LinkId) (client.Link, error)
	GetLinks              func(ctx context.Context, ids ...client.LinkId) ([]client.Link, error)
	ListLinksForAccount   func(ctx context.Context, accountId client.AccountId, expired bool) ([]client.Link, error)
	ListLinksForPublicKey func(ctx context.Context, publicKeyId client.PublicKeyId, expired bool) ([]client.Link, error)
	UpdateLink            func(ctx context.Context, id client.LinkId, accountId client.AccountId, tagMatcher string, expiresAt time.Time) error
	DeleteLinks           func(ctx context.Context, ids ...client.LinkId) error

	// --- Other ---
	ListExistingTags   func(ctx context.Context) tags.Tags
	OnboardHost        func(ctx context.Context, host string, port int /* , gateway string, plugin string */, accountUsername string, deploymentKey string) (chan client.OnboardHostProgress, error)
	DecommisionAccount func(ctx context.Context, id client.AccountId) (chan client.DecommisionAccountProgress, error)
	DeployAccount      func(ctx context.Context, accountId client.AccountId) (chan client.DeployProgressAccount, error)
	DeployAccounts     func(ctx context.Context, accountIds ...client.AccountId) (chan client.DeployProgressAccounts, error)
}

// *[Client] implements [client.Client]
var _ client.Client = (*Client)(nil)

type MockOption func(*Client)

func NewClient(opts ...MockOption) *Client {
	c := &Client{}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func WitchBaseClient(base client.Client) MockOption {
	return func(mc *Client) { mc.BaseClient = base }
}

func WitchOverwrites(overwrites ClientOverwrites) MockOption {
	return func(mc *Client) { mc.Overwrites = overwrites }
}

func WitchPre(fn func(method string, args map[string]any)) MockOption {
	return func(mc *Client) { mc.Pre = fn }
}

// --- Client implementation template ---

// func (m *Client) <MethodUsername>(ctx context.Context, <Args>) <ReturnValues> {
//     if m.Pre != nil {
//         m.Pre("<MethodUsername>", map[string]any{"<ArgUsername>": <ArgValue>, ...})
//     }
//     if m.Overwrites.<MethodUsername> != nil {
//         return m.Overwrites.<MethodUsername>(ctx, <Args>)
//     }
//     else if m.BaseClient != nil {
//         return m.BaseClient.<MethodUsername>(ctx, <Args>)
//     }
//     panic("Client.<MethodUsername> not implemented")
// }

// --- Lifecycle & Initialization ---

func (m *Client) Close(ctx context.Context) error {
	if m.Pre != nil {
		m.Pre("Close", map[string]any{"ctx": ctx})
	}
	if m.Overwrites.Close != nil {
		return m.Overwrites.Close(ctx)
	} else if m.BaseClient != nil {
		return m.BaseClient.Close(ctx)
	}
	panic("Client.Close not implemented")
}

func (m *Client) WithTransaction(ctx context.Context, fn func(c client.Client) error) error {
	if m.Pre != nil {
		m.Pre("WithTransaction", map[string]any{"ctx": ctx, "fn": fn})
	}
	if m.Overwrites.WithTransaction != nil {
		return m.Overwrites.WithTransaction(ctx, fn)
	} else if m.BaseClient != nil {
		return m.BaseClient.WithTransaction(ctx, fn)
	}
	panic("Client.WithTransaction not implemented")
}

// --- PublicKey Management ---

func (m *Client) CreatePublicKey(ctx context.Context, key string, comment string, tags tags.Tags) (client.PublicKey, error) {
	if m.Pre != nil {
		m.Pre("CreatePublicKey", map[string]any{"ctx": ctx, "key": key, "comment": comment, "tags": tags})
	}
	if m.Overwrites.CreatePublicKey != nil {
		return m.Overwrites.CreatePublicKey(ctx, key, comment, tags)
	} else if m.BaseClient != nil {
		return m.BaseClient.CreatePublicKey(ctx, key, comment, tags)
	}
	panic("Client.CreatePublicKey not implemented")
}

func (m *Client) GetPublicKey(ctx context.Context, id client.PublicKeyId) (client.PublicKey, error) {
	if m.Pre != nil {
		m.Pre("GetPublicKey", map[string]any{"ctx": ctx, "id": id})
	}
	if m.Overwrites.GetPublicKey != nil {
		return m.Overwrites.GetPublicKey(ctx, id)
	} else if m.BaseClient != nil {
		return m.BaseClient.GetPublicKey(ctx, id)
	}
	panic("Client.GetPublicKey not implemented")
}

func (m *Client) GetPublicKeys(ctx context.Context, ids ...client.PublicKeyId) ([]client.PublicKey, error) {
	if m.Pre != nil {
		m.Pre("GetPublicKeys", map[string]any{"ctx": ctx, "ids": ids})
	}
	if m.Overwrites.GetPublicKeys != nil {
		return m.Overwrites.GetPublicKeys(ctx, ids...)
	} else if m.BaseClient != nil {
		return m.BaseClient.GetPublicKeys(ctx, ids...)
	}
	panic("Client.GetPublicKeys not implemented")
}

func (m *Client) ListPublicKeys(ctx context.Context, tagMatcher string) ([]client.PublicKey, error) {
	if m.Pre != nil {
		m.Pre("ListPublicKeys", map[string]any{"ctx": ctx, "tagMatcher": tagMatcher})
	}
	if m.Overwrites.ListPublicKeys != nil {
		return m.Overwrites.ListPublicKeys(ctx, tagMatcher)
	} else if m.BaseClient != nil {
		return m.BaseClient.ListPublicKeys(ctx, tagMatcher)
	}
	panic("Client.ListPublicKeys not implemented")
}

func (m *Client) ListPublicKeysLinkedToAccount(ctx context.Context, accountId client.AccountId, expired bool) ([]client.PublicKey, error) {
	if m.Pre != nil {
		m.Pre("ListPublicKeysLinkedToAccount", map[string]any{"ctx": ctx, "accountId": accountId, "expired": expired})
	}
	if m.Overwrites.ListPublicKeysLinkedToAccount != nil {
		return m.Overwrites.ListPublicKeysLinkedToAccount(ctx, accountId, expired)
	} else if m.BaseClient != nil {
		return m.BaseClient.ListPublicKeysLinkedToAccount(ctx, accountId, expired)
	}
	panic("Client.ListPublicKeysLinkedToAccount not implemented")
}

func (m *Client) UpdatePublicKey(ctx context.Context, id client.PublicKeyId, comment string, tags tags.Tags) error {
	if m.Pre != nil {
		m.Pre("UpdatePublicKey", map[string]any{"ctx": ctx, "id": id, "comment": comment, "tags": tags})
	}
	if m.Overwrites.UpdatePublicKey != nil {
		return m.Overwrites.UpdatePublicKey(ctx, id, comment, tags)
	} else if m.BaseClient != nil {
		return m.BaseClient.UpdatePublicKey(ctx, id, comment, tags)
	}
	panic("Client.UpdatePublicKey not implemented")
}

func (m *Client) DeletePublicKeys(ctx context.Context, ids ...client.PublicKeyId) error {
	if m.Pre != nil {
		m.Pre("DeletePublicKeys", map[string]any{"ctx": ctx, "ids": ids})
	}
	if m.Overwrites.DeletePublicKeys != nil {
		return m.Overwrites.DeletePublicKeys(ctx, ids...)
	} else if m.BaseClient != nil {
		return m.BaseClient.DeletePublicKeys(ctx, ids...)
	}
	panic("Client.DeletePublicKeys not implemented")
}

// --- Account Management ---

func (m *Client) CreateAccount(ctx context.Context, username string, host string, port int, deploymentMethod string, deploymentSecret string) (client.Account, error) {
	if m.Pre != nil {
		m.Pre("CreateAccount", map[string]any{
			"ctx": ctx, "username": username, "host": host, "port": port,
			"deploymentMethod": deploymentMethod, "deploymentSecret": deploymentSecret,
		})
	}
	if m.Overwrites.CreateAccount != nil {
		return m.Overwrites.CreateAccount(ctx, username, host, port, deploymentMethod, deploymentSecret)
	} else if m.BaseClient != nil {
		return m.BaseClient.CreateAccount(ctx, username, host, port, deploymentMethod, deploymentSecret)
	}
	panic("Client.CreateAccount not implemented")
}

func (m *Client) GetAccount(ctx context.Context, id client.AccountId) (client.Account, error) {
	if m.Pre != nil {
		m.Pre("GetAccount", map[string]any{"ctx": ctx, "id": id})
	}
	if m.Overwrites.GetAccount != nil {
		return m.Overwrites.GetAccount(ctx, id)
	} else if m.BaseClient != nil {
		return m.BaseClient.GetAccount(ctx, id)
	}
	panic("Client.GetAccount not implemented")
}

func (m *Client) GetAccounts(ctx context.Context, ids ...client.AccountId) ([]client.Account, error) {
	if m.Pre != nil {
		m.Pre("GetAccounts", map[string]any{"ctx": ctx, "ids": ids})
	}
	if m.Overwrites.GetAccounts != nil {
		return m.Overwrites.GetAccounts(ctx, ids...)
	} else if m.BaseClient != nil {
		return m.BaseClient.GetAccounts(ctx, ids...)
	}
	panic("Client.GetAccounts not implemented")
}

func (m *Client) ListAccounts(ctx context.Context) ([]client.Account, error) {
	if m.Pre != nil {
		m.Pre("ListAccounts", map[string]any{"ctx": ctx})
	}
	if m.Overwrites.ListAccounts != nil {
		return m.Overwrites.ListAccounts(ctx)
	} else if m.BaseClient != nil {
		return m.BaseClient.ListAccounts(ctx)
	}
	panic("Client.ListAccounts not implemented")
}

func (m *Client) ListAccountsDirty(ctx context.Context) ([]client.Account, error) {
	if m.Pre != nil {
		m.Pre("ListAccountsDirty", map[string]any{"ctx": ctx})
	}
	if m.Overwrites.ListAccountsDirty != nil {
		return m.Overwrites.ListAccountsDirty(ctx)
	} else if m.BaseClient != nil {
		return m.BaseClient.ListAccountsDirty(ctx)
	}
	panic("Client.ListAccountsDirty not implemented")
}

func (m *Client) ListAccountsLinkedToPublicKey(ctx context.Context, publicKeyId client.PublicKeyId, expired bool) ([]client.Account, error) {
	if m.Pre != nil {
		m.Pre("ListAccountsLinkedToPublicKey", map[string]any{"ctx": ctx, "publicKeyId": publicKeyId, "expired": expired})
	}
	if m.Overwrites.ListAccountsLinkedToPublicKey != nil {
		return m.Overwrites.ListAccountsLinkedToPublicKey(ctx, publicKeyId, expired)
	} else if m.BaseClient != nil {
		return m.BaseClient.ListAccountsLinkedToPublicKey(ctx, publicKeyId, expired)
	}
	panic("Client.ListAccountsLinkedToPublicKey not implemented")
}

func (m *Client) UpdateAccount(ctx context.Context, id client.AccountId, username string, host string, port int, deploymentMethod string, deploymentSecret string) error {
	if m.Pre != nil {
		m.Pre("UpdateAccount", map[string]any{
			"ctx": ctx, "id": id, "username": username, "host": host, "port": port,
			"deploymentMethod": deploymentMethod, "deploymentSecret": deploymentSecret,
		})
	}
	if m.Overwrites.UpdateAccount != nil {
		return m.Overwrites.UpdateAccount(ctx, id, username, host, port, deploymentMethod, deploymentSecret)
	} else if m.BaseClient != nil {
		return m.BaseClient.UpdateAccount(ctx, id, username, host, port, deploymentMethod, deploymentSecret)
	}
	panic("Client.UpdateAccount not implemented")
}

func (m *Client) DeleteAccounts(ctx context.Context, ids ...client.AccountId) error {
	if m.Pre != nil {
		m.Pre("DeleteAccounts", map[string]any{"ctx": ctx, "ids": ids})
	}
	if m.Overwrites.DeleteAccounts != nil {
		return m.Overwrites.DeleteAccounts(ctx, ids...)
	} else if m.BaseClient != nil {
		return m.BaseClient.DeleteAccounts(ctx, ids...)
	}
	panic("Client.DeleteAccounts not implemented")
}

func (m *Client) IsAccountDirty(ctx context.Context, account client.Account) (bool, error) {
	if m.Pre != nil {
		m.Pre("IsAccountDirty", map[string]any{"ctx": ctx, "account": account})
	}
	if m.Overwrites.IsAccountDirty != nil {
		return m.Overwrites.IsAccountDirty(ctx, account)
	} else if m.BaseClient != nil {
		return m.BaseClient.IsAccountDirty(ctx, account)
	}
	panic("Client.IsAccountDirty not implemented")
}

// --- Link Management ---

func (m *Client) CreateLink(ctx context.Context, accountId client.AccountId, tagMatcher string, expiresAt time.Time) (client.Link, error) {
	if m.Pre != nil {
		m.Pre("CreateLink", map[string]any{"ctx": ctx, "accountId": accountId, "tagMatcher": tagMatcher, "expiresAt": expiresAt})
	}
	if m.Overwrites.CreateLink != nil {
		return m.Overwrites.CreateLink(ctx, accountId, tagMatcher, expiresAt)
	} else if m.BaseClient != nil {
		return m.BaseClient.CreateLink(ctx, accountId, tagMatcher, expiresAt)
	}
	panic("Client.CreateLink not implemented")
}

func (m *Client) GetLink(ctx context.Context, id client.LinkId) (client.Link, error) {
	if m.Pre != nil {
		m.Pre("GetLink", map[string]any{"ctx": ctx, "id": id})
	}
	if m.Overwrites.GetLink != nil {
		return m.Overwrites.GetLink(ctx, id)
	} else if m.BaseClient != nil {
		return m.BaseClient.GetLink(ctx, id)
	}
	panic("Client.GetLink not implemented")
}

func (m *Client) GetLinks(ctx context.Context, ids ...client.LinkId) ([]client.Link, error) {
	if m.Pre != nil {
		m.Pre("GetLinks", map[string]any{"ctx": ctx, "ids": ids})
	}
	if m.Overwrites.GetLinks != nil {
		return m.Overwrites.GetLinks(ctx, ids...)
	} else if m.BaseClient != nil {
		return m.BaseClient.GetLinks(ctx, ids...)
	}
	panic("Client.GetLinks not implemented")
}

func (m *Client) ListLinksForAccount(ctx context.Context, accountId client.AccountId, expired bool) ([]client.Link, error) {
	if m.Pre != nil {
		m.Pre("ListLinksForAccount", map[string]any{"ctx": ctx, "accountId": accountId, "expired": expired})
	}
	if m.Overwrites.ListLinksForAccount != nil {
		return m.Overwrites.ListLinksForAccount(ctx, accountId, expired)
	} else if m.BaseClient != nil {
		return m.BaseClient.ListLinksForAccount(ctx, accountId, expired)
	}
	panic("Client.ListLinksForAccount not implemented")
}

func (m *Client) ListLinksForPublicKey(ctx context.Context, publicKeyId client.PublicKeyId, expired bool) ([]client.Link, error) {
	if m.Pre != nil {
		m.Pre("ListLinksForPublicKey", map[string]any{"ctx": ctx, "publicKeyId": publicKeyId, "expired": expired})
	}
	if m.Overwrites.ListLinksForPublicKey != nil {
		return m.Overwrites.ListLinksForPublicKey(ctx, publicKeyId, expired)
	} else if m.BaseClient != nil {
		return m.BaseClient.ListLinksForPublicKey(ctx, publicKeyId, expired)
	}
	panic("Client.ListLinksForPublicKey not implemented")
}

func (m *Client) UpdateLink(ctx context.Context, id client.LinkId, accountId client.AccountId, tagMatcher string, expiresAt time.Time) error {
	if m.Pre != nil {
		m.Pre("UpdateLink", map[string]any{"ctx": ctx, "id": id, "accountId": accountId, "tagMatcher": tagMatcher, "expiresAt": expiresAt})
	}
	if m.Overwrites.UpdateLink != nil {
		return m.Overwrites.UpdateLink(ctx, id, accountId, tagMatcher, expiresAt)
	} else if m.BaseClient != nil {
		return m.BaseClient.UpdateLink(ctx, id, accountId, tagMatcher, expiresAt)
	}
	panic("Client.UpdateLink not implemented")
}

func (m *Client) DeleteLinks(ctx context.Context, ids ...client.LinkId) error {
	if m.Pre != nil {
		m.Pre("DeleteLinks", map[string]any{"ctx": ctx, "ids": ids})
	}
	if m.Overwrites.DeleteLinks != nil {
		return m.Overwrites.DeleteLinks(ctx, ids...)
	} else if m.BaseClient != nil {
		return m.BaseClient.DeleteLinks(ctx, ids...)
	}
	panic("Client.DeleteLinks not implemented")
}

// --- Other ---

func (m *Client) ListExistingTags(ctx context.Context) tags.Tags {
	if m.Pre != nil {
		m.Pre("ListExistingTags", map[string]any{"ctx": ctx})
	}
	if m.Overwrites.ListExistingTags != nil {
		return m.Overwrites.ListExistingTags(ctx)
	} else if m.BaseClient != nil {
		return m.BaseClient.ListExistingTags(ctx)
	}
	panic("Client.ListExistingTags not implemented")
}

func (m *Client) OnboardHost(ctx context.Context, host string, port int, accountUsername string, deploymentKey string) (chan client.OnboardHostProgress, error) {
	if m.Pre != nil {
		m.Pre("OnboardHost", map[string]any{"ctx": ctx, "host": host, "port": port, "accountUsername": accountUsername, "deploymentKey": deploymentKey})
	}
	if m.Overwrites.OnboardHost != nil {
		return m.Overwrites.OnboardHost(ctx, host, port, accountUsername, deploymentKey)
	} else if m.BaseClient != nil {
		return m.BaseClient.OnboardHost(ctx, host, port, accountUsername, deploymentKey)
	}
	panic("Client.OnboardHost not implemented")
}

func (m *Client) DecommisionAccount(ctx context.Context, id client.AccountId) (chan client.DecommisionAccountProgress, error) {
	if m.Pre != nil {
		m.Pre("DecommisionAccount", map[string]any{"ctx": ctx, "id": id})
	}
	if m.Overwrites.DecommisionAccount != nil {
		return m.Overwrites.DecommisionAccount(ctx, id)
	} else if m.BaseClient != nil {
		return m.BaseClient.DecommisionAccount(ctx, id)
	}
	panic("Client.DecommisionAccount not implemented")
}

func (m *Client) DeployAccount(ctx context.Context, accountId client.AccountId) (chan client.DeployProgressAccount, error) {
	if m.Pre != nil {
		m.Pre("DeployAccount", map[string]any{"ctx": ctx, "accountId": accountId})
	}
	if m.Overwrites.DeployAccount != nil {
		return m.Overwrites.DeployAccount(ctx, accountId)
	} else if m.BaseClient != nil {
		return m.BaseClient.DeployAccount(ctx, accountId)
	}
	panic("Client.DeployAccount not implemented")
}

func (m *Client) DeployAccounts(ctx context.Context, accountIds ...client.AccountId) (chan client.DeployProgressAccounts, error) {
	if m.Pre != nil {
		m.Pre("DeployAccounts", map[string]any{"ctx": ctx, "accountIds": accountIds})
	}
	if m.Overwrites.DeployAccounts != nil {
		return m.Overwrites.DeployAccounts(ctx, accountIds...)
	} else if m.BaseClient != nil {
		return m.BaseClient.DeployAccounts(ctx, accountIds...)
	}
	panic("Client.DeployAccounts not implemented")
}
