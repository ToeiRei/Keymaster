// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package client

import (
	"context"
	"time"
)

type MockClient struct {
	BaseClient Client
	Overwrites MockClientOverwrites
}

type MockClientOverwrites struct {
	// --- Lifecycle & Initialization ---
	Close func(ctx context.Context) error
	// --- PublicKey Management ---
	CreatePublicKey  func(ctx context.Context, key string, comment string, tags []string) (PublicKey, error)
	GetPublicKey     func(ctx context.Context, id ID) (PublicKey, error)
	GetPublicKeys    func(ctx context.Context, ids ...ID) ([]PublicKey, error)
	ListPublicKeys   func(ctx context.Context, tagFilter string) ([]PublicKey, error)
	UpdatePublicKey  func(ctx context.Context, id ID, comment string, tags []string) error
	DeletePublicKeys func(ctx context.Context, ids ...ID) error
	// --- Account Management ---
	CreateAccount    func(ctx context.Context, name string, host string, port int, deploymentMethod string, deploymentSecret string) (Account, error)
	GetAccount       func(ctx context.Context, id ID) (Account, error)
	GetAccounts      func(ctx context.Context, ids ...ID) ([]Account, error)
	ListAccounts     func(ctx context.Context) ([]Account, error)
	UpdateAccount    func(ctx context.Context, id ID, name string, host string, port int, deploymentMethod string, deploymentSecret string) error
	DeleteAccounts   func(ctx context.Context, ids ...ID) error
	IsAccountDirty   func(ctx context.Context, account Account) (bool, error)
	GetDirtyAccounts func(ctx context.Context) ([]Account, error)
	// --- Tag & Account-PublicKey relation Management ---
	ListExistingTags            func(ctx context.Context) []string
	LinkTagAccount              func(ctx context.Context, accountID ID, filter string, expiresAt time.Time) (ID, error)
	UnLinkTagAccount            func(ctx context.Context, linkIDs ...ID) error
	ResolvePublicKeyLinks       func(ctx context.Context, accountID ID) ([]Link, error)
	ResolveAccountLinks         func(ctx context.Context, publicKeyID ID) ([]Link, error)
	ResolvePublicKeysForAccount func(ctx context.Context, accountID ID) ([]PublicKey, error)
	ResolveAccountsForPublicKey func(ctx context.Context, publicKeyID ID) ([]Account, error)
	// --- Onboarding & Decommision ---
	OnboardHost        func(ctx context.Context, host string, port int, accountName string, deploymentKey string) (chan OnboardHostProgress, error)
	DecommisionAccount func(ctx context.Context, id ID) (chan DecommisionAccountProgress, error)
	// --- Deploy stuff ---
	DeployPublicKeys func(ctx context.Context, publicKeyID ...ID) (chan DeployProgress, error)
	DeployAccounts   func(ctx context.Context, accountID ...ID) (chan DeployProgress, error)
	DeployAll        func(ctx context.Context) (chan DeployProgress, error)
}

var _ Client = (*MockClient)(nil)

// client := NewMockClient(nil, MockClientOverwrites{ /* overwrite Client methods here... */ })
func NewMockClient(base Client, overwrites MockClientOverwrites) *MockClient {
	return &MockClient{
		BaseClient: base,
		Overwrites: overwrites,
	}
}

// --- Client implementation ---

// func (m *MockClient) <MethodName>(ctx context.Context, <Args>) <ReturnValues> {
//     if m.Overwrites.<MethodName> != nil {
//         return m.Overwrites.<MethodName>(ctx, <Args>)
//     }
//     else if m.BaseClient != nil {
//         return m.BaseClient.<MethodName>(ctx, <Args>)
//     }
//     panic("MockClient.<MethodName> not implemented")
// }

// --- Lifecycle & Initialization ---

func (m *MockClient) Close(ctx context.Context) error {
	if m.Overwrites.Close != nil {
		return m.Overwrites.Close(ctx)
	} else if m.BaseClient != nil {
		return m.BaseClient.Close(ctx)
	}
	panic("MockClient.Close not implemented")
}

// --- PublicKey Management ---

func (m *MockClient) CreatePublicKey(ctx context.Context, key string, comment string, tags []string) (PublicKey, error) {
	if m.Overwrites.CreatePublicKey != nil {
		return m.Overwrites.CreatePublicKey(ctx, key, comment, tags)
	} else if m.BaseClient != nil {
		return m.BaseClient.CreatePublicKey(ctx, key, comment, tags)
	}
	panic("MockClient.CreatePublicKey not implemented")
}

func (m *MockClient) GetPublicKey(ctx context.Context, id ID) (PublicKey, error) {
	if m.Overwrites.GetPublicKey != nil {
		return m.Overwrites.GetPublicKey(ctx, id)
	} else if m.BaseClient != nil {
		return m.BaseClient.GetPublicKey(ctx, id)
	}
	panic("MockClient.GetPublicKey not implemented")
}

func (m *MockClient) GetPublicKeys(ctx context.Context, ids ...ID) ([]PublicKey, error) {
	if m.Overwrites.GetPublicKeys != nil {
		return m.Overwrites.GetPublicKeys(ctx, ids...)
	} else if m.BaseClient != nil {
		return m.BaseClient.GetPublicKeys(ctx, ids...)
	}
	panic("MockClient.GetPublicKeys not implemented")
}

func (m *MockClient) ListPublicKeys(ctx context.Context, tagFilter string) ([]PublicKey, error) {
	if m.Overwrites.ListPublicKeys != nil {
		return m.Overwrites.ListPublicKeys(ctx, tagFilter)
	} else if m.BaseClient != nil {
		return m.BaseClient.ListPublicKeys(ctx, tagFilter)
	}
	panic("MockClient.ListPublicKeys not implemented")
}

func (m *MockClient) UpdatePublicKey(ctx context.Context, id ID, comment string, tags []string) error {
	if m.Overwrites.UpdatePublicKey != nil {
		return m.Overwrites.UpdatePublicKey(ctx, id, comment, tags)
	} else if m.BaseClient != nil {
		return m.BaseClient.UpdatePublicKey(ctx, id, comment, tags)
	}
	panic("MockClient.UpdatePublicKey not implemented")
}

func (m *MockClient) DeletePublicKeys(ctx context.Context, ids ...ID) error {
	if m.Overwrites.DeletePublicKeys != nil {
		return m.Overwrites.DeletePublicKeys(ctx, ids...)
	} else if m.BaseClient != nil {
		return m.BaseClient.DeletePublicKeys(ctx, ids...)
	}
	panic("MockClient.DeletePublicKeys not implemented")
}

// --- Account Management ---

func (m *MockClient) CreateAccount(ctx context.Context, name string, host string, port int, deploymentMethod string, deploymentSecret string) (Account, error) {
	if m.Overwrites.CreateAccount != nil {
		return m.Overwrites.CreateAccount(ctx, name, host, port, deploymentMethod, deploymentSecret)
	} else if m.BaseClient != nil {
		return m.BaseClient.CreateAccount(ctx, name, host, port, deploymentMethod, deploymentSecret)
	}
	panic("MockClient.CreateAccount not implemented")
}

func (m *MockClient) GetAccount(ctx context.Context, id ID) (Account, error) {
	if m.Overwrites.GetAccount != nil {
		return m.Overwrites.GetAccount(ctx, id)
	} else if m.BaseClient != nil {
		return m.BaseClient.GetAccount(ctx, id)
	}
	panic("MockClient.GetAccount not implemented")
}

func (m *MockClient) GetAccounts(ctx context.Context, ids ...ID) ([]Account, error) {
	if m.Overwrites.GetAccounts != nil {
		return m.Overwrites.GetAccounts(ctx, ids...)
	} else if m.BaseClient != nil {
		return m.BaseClient.GetAccounts(ctx, ids...)
	}
	panic("MockClient.GetAccounts not implemented")
}

func (m *MockClient) ListAccounts(ctx context.Context) ([]Account, error) {
	if m.Overwrites.ListAccounts != nil {
		return m.Overwrites.ListAccounts(ctx)
	} else if m.BaseClient != nil {
		return m.BaseClient.ListAccounts(ctx)
	}
	panic("MockClient.ListAccounts not implemented")
}

func (m *MockClient) UpdateAccount(ctx context.Context, id ID, name string, host string, port int, deploymentMethod string, deploymentSecret string) error {
	if m.Overwrites.UpdateAccount != nil {
		return m.Overwrites.UpdateAccount(ctx, id, name, host, port, deploymentMethod, deploymentSecret)
	} else if m.BaseClient != nil {
		return m.BaseClient.UpdateAccount(ctx, id, name, host, port, deploymentMethod, deploymentSecret)
	}
	panic("MockClient.UpdateAccount not implemented")
}

func (m *MockClient) DeleteAccounts(ctx context.Context, ids ...ID) error {
	if m.Overwrites.DeleteAccounts != nil {
		return m.Overwrites.DeleteAccounts(ctx, ids...)
	} else if m.BaseClient != nil {
		return m.BaseClient.DeleteAccounts(ctx, ids...)
	}
	panic("MockClient.DeleteAccounts not implemented")
}

func (m *MockClient) IsAccountDirty(ctx context.Context, account Account) (bool, error) {
	if m.Overwrites.IsAccountDirty != nil {
		return m.Overwrites.IsAccountDirty(ctx, account)
	} else if m.BaseClient != nil {
		return m.BaseClient.IsAccountDirty(ctx, account)
	}
	panic("MockClient.IsAccountDirty not implemented")
}

func (m *MockClient) GetDirtyAccounts(ctx context.Context) ([]Account, error) {
	if m.Overwrites.GetDirtyAccounts != nil {
		return m.Overwrites.GetDirtyAccounts(ctx)
	} else if m.BaseClient != nil {
		return m.BaseClient.GetDirtyAccounts(ctx)
	}
	panic("MockClient.GetDirtyAccounts not implemented")
}

// --- Tag & Account-PublicKey relation Management ---

func (m *MockClient) ListExistingTags(ctx context.Context) []string {
	if m.Overwrites.ListExistingTags != nil {
		return m.Overwrites.ListExistingTags(ctx)
	} else if m.BaseClient != nil {
		return m.BaseClient.ListExistingTags(ctx)
	}
	panic("MockClient.ListExistingTags not implemented")
}

func (m *MockClient) LinkTagAccount(ctx context.Context, accountID ID, filter string, expiresAt time.Time) (ID, error) {
	if m.Overwrites.LinkTagAccount != nil {
		return m.Overwrites.LinkTagAccount(ctx, accountID, filter, expiresAt)
	} else if m.BaseClient != nil {
		return m.BaseClient.LinkTagAccount(ctx, accountID, filter, expiresAt)
	}
	panic("MockClient.LinkTagAccount not implemented")
}

func (m *MockClient) UnLinkTagAccount(ctx context.Context, linkIDs ...ID) error {
	if m.Overwrites.UnLinkTagAccount != nil {
		return m.Overwrites.UnLinkTagAccount(ctx, linkIDs...)
	} else if m.BaseClient != nil {
		return m.BaseClient.UnLinkTagAccount(ctx, linkIDs...)
	}
	panic("MockClient.UnLinkTagAccount not implemented")
}

func (m *MockClient) ResolvePublicKeyLinks(ctx context.Context, accountID ID) ([]Link, error) {
	if m.Overwrites.ResolvePublicKeyLinks != nil {
		return m.Overwrites.ResolvePublicKeyLinks(ctx, accountID)
	} else if m.BaseClient != nil {
		return m.BaseClient.ResolvePublicKeyLinks(ctx, accountID)
	}
	panic("MockClient.ResolvePublicKeyLinks not implemented")
}

func (m *MockClient) ResolveAccountLinks(ctx context.Context, publicKeyID ID) ([]Link, error) {
	if m.Overwrites.ResolveAccountLinks != nil {
		return m.Overwrites.ResolveAccountLinks(ctx, publicKeyID)
	} else if m.BaseClient != nil {
		return m.BaseClient.ResolveAccountLinks(ctx, publicKeyID)
	}
	panic("MockClient.ResolveAccountLinks not implemented")
}

func (m *MockClient) ResolvePublicKeysForAccount(ctx context.Context, accountID ID) ([]PublicKey, error) {
	if m.Overwrites.ResolvePublicKeysForAccount != nil {
		return m.Overwrites.ResolvePublicKeysForAccount(ctx, accountID)
	} else if m.BaseClient != nil {
		return m.BaseClient.ResolvePublicKeysForAccount(ctx, accountID)
	}
	panic("MockClient.ResolvePublicKeysForAccount not implemented")
}

func (m *MockClient) ResolveAccountsForPublicKey(ctx context.Context, publicKeyID ID) ([]Account, error) {
	if m.Overwrites.ResolveAccountsForPublicKey != nil {
		return m.Overwrites.ResolveAccountsForPublicKey(ctx, publicKeyID)
	} else if m.BaseClient != nil {
		return m.BaseClient.ResolveAccountsForPublicKey(ctx, publicKeyID)
	}
	panic("MockClient.ResolveAccountsForPublicKey not implemented")
}

// --- Onboarding & Decommision ---

func (m *MockClient) OnboardHost(ctx context.Context, host string, port int, accountName string, deploymentKey string) (chan OnboardHostProgress, error) {
	if m.Overwrites.OnboardHost != nil {
		return m.Overwrites.OnboardHost(ctx, host, port, accountName, deploymentKey)
	} else if m.BaseClient != nil {
		return m.BaseClient.OnboardHost(ctx, host, port, accountName, deploymentKey)
	}
	panic("MockClient.OnboardHost not implemented")
}

func (m *MockClient) DecommisionAccount(ctx context.Context, id ID) (chan DecommisionAccountProgress, error) {
	if m.Overwrites.DecommisionAccount != nil {
		return m.Overwrites.DecommisionAccount(ctx, id)
	} else if m.BaseClient != nil {
		return m.BaseClient.DecommisionAccount(ctx, id)
	}
	panic("MockClient.DecommisionAccount not implemented")
}

// --- Deploy stuff ---

func (m *MockClient) DeployPublicKeys(ctx context.Context, publicKeyID ...ID) (chan DeployProgress, error) {
	if m.Overwrites.DeployPublicKeys != nil {
		return m.Overwrites.DeployPublicKeys(ctx, publicKeyID...)
	} else if m.BaseClient != nil {
		return m.BaseClient.DeployPublicKeys(ctx, publicKeyID...)
	}
	panic("MockClient.DeployPublicKeys not implemented")
}

func (m *MockClient) DeployAccounts(ctx context.Context, accountID ...ID) (chan DeployProgress, error) {
	if m.Overwrites.DeployAccounts != nil {
		return m.Overwrites.DeployAccounts(ctx, accountID...)
	} else if m.BaseClient != nil {
		return m.BaseClient.DeployAccounts(ctx, accountID...)
	}
	panic("MockClient.DeployAccounts not implemented")
}

func (m *MockClient) DeployAll(ctx context.Context) (chan DeployProgress, error) {
	if m.Overwrites.DeployAll != nil {
		return m.Overwrites.DeployAll(ctx)
	} else if m.BaseClient != nil {
		return m.BaseClient.DeployAll(ctx)
	}
	panic("MockClient.DeployAll not implemented")
}
