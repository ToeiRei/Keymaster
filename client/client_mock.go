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
	Close                       func(ctx context.Context) error
	CreateAccount               func(ctx context.Context, targetID ID, name string, deploymentKey string) (Account, error)
	CreatePublicKey             func(ctx context.Context, identity string, tags []string) (PublicKey, error)
	CreateTarget                func(ctx context.Context, host string, port int) (Target, error)
	DecommisionAccount          func(ctx context.Context, id ID) (chan DecommisionAccountProgress, error)
	DecommisionTarget           func(ctx context.Context, id ID) (chan DecommisionTargetProgress, error)
	DeleteAccounts              func(ctx context.Context, ids ...ID) error
	DeletePublicKeys            func(ctx context.Context, ids ...ID) error
	DeleteTargets               func(ctx context.Context, ids ...ID) error
	DeployAccounts              func(ctx context.Context, accountID ...ID) (chan DeployProgress, error)
	DeployAll                   func(ctx context.Context) (chan DeployProgress, error)
	DeployPublicKeys            func(ctx context.Context, publicKeyID ...ID) (chan DeployProgress, error)
	DeployTargets               func(ctx context.Context, targetID ...ID) (chan DeployProgress, error)
	GetAccount                  func(ctx context.Context, id ID) (Account, error)
	GetAccounts                 func(ctx context.Context, ids ...ID) ([]Account, error)
	GetDirtyAccounts            func(ctx context.Context) ([]Account, error)
	GetPublicKey                func(ctx context.Context, id ID) (PublicKey, error)
	GetPublicKeys               func(ctx context.Context, ids ...ID) ([]PublicKey, error)
	GetTarget                   func(ctx context.Context, id ID) (Target, error)
	GetTargets                  func(ctx context.Context, ids ...ID) ([]Target, error)
	LinkTagAccount              func(ctx context.Context, accountID ID, filter string, expiresAt time.Time) (ID, error)
	ListAccountsByTarget        func(ctx context.Context, targetID ID) ([]Account, error)
	ListPublicKeys              func(ctx context.Context, tagFilter string) ([]PublicKey, error)
	ListTargets                 func(ctx context.Context) ([]Target, error)
	OnboardHost                 func(ctx context.Context, host string, port int, accountName string, deploymentKey string) (chan OnboardHostProgress, error)
	ResolveAccountsForPublicKey func(ctx context.Context, publicKeyID ID) ([]Account, error)
	ResolvePublicKeysForAccount func(ctx context.Context, accountID ID) ([]PublicKey, error)
	UnLinkTagAccount            func(ctx context.Context, linkIDs ...ID) error
	UpdatePublicKeyTags         func(ctx context.Context, id ID, tags []string) error
	UpdateTarget                func(ctx context.Context, id ID, target Target) error
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

func (m *MockClient) Close(ctx context.Context) error {
	if m.Overwrites.Close != nil {
		return m.Overwrites.Close(ctx)
	} else if m.BaseClient != nil {
		return m.BaseClient.Close(ctx)
	}
	panic("MockClient.Close not implemented")
}
func (m *MockClient) CreateAccount(ctx context.Context, targetID ID, name string, deploymentKey string) (Account, error) {
	if m.Overwrites.CreateAccount != nil {
		return m.Overwrites.CreateAccount(ctx, targetID, name, deploymentKey)
	} else if m.BaseClient != nil {
		return m.BaseClient.CreateAccount(ctx, targetID, name, deploymentKey)
	}
	panic("MockClient.CreateAccount not implemented")
}
func (m *MockClient) CreatePublicKey(ctx context.Context, identity string, tags []string) (PublicKey, error) {
	if m.Overwrites.CreatePublicKey != nil {
		return m.Overwrites.CreatePublicKey(ctx, identity, tags)
	} else if m.BaseClient != nil {
		return m.BaseClient.CreatePublicKey(ctx, identity, tags)
	}
	panic("MockClient.CreatePublicKey not implemented")
}
func (m *MockClient) CreateTarget(ctx context.Context, host string, port int) (Target, error) {
	if m.Overwrites.CreateTarget != nil {
		return m.Overwrites.CreateTarget(ctx, host, port)
	} else if m.BaseClient != nil {
		return m.BaseClient.CreateTarget(ctx, host, port)
	}
	panic("MockClient.CreateTarget not implemented")
}
func (m *MockClient) DecommisionAccount(ctx context.Context, id ID) (chan DecommisionAccountProgress, error) {
	if m.Overwrites.DecommisionAccount != nil {
		return m.Overwrites.DecommisionAccount(ctx, id)
	} else if m.BaseClient != nil {
		return m.BaseClient.DecommisionAccount(ctx, id)
	}
	panic("MockClient.DecommisionAccount not implemented")
}
func (m *MockClient) DecommisionTarget(ctx context.Context, id ID) (chan DecommisionTargetProgress, error) {
	if m.Overwrites.DecommisionTarget != nil {
		return m.Overwrites.DecommisionTarget(ctx, id)
	} else if m.BaseClient != nil {
		return m.BaseClient.DecommisionTarget(ctx, id)
	}
	panic("MockClient.DecommisionTarget not implemented")
}
func (m *MockClient) DeleteAccounts(ctx context.Context, ids ...ID) error {
	if m.Overwrites.DeleteAccounts != nil {
		return m.Overwrites.DeleteAccounts(ctx, ids...)
	} else if m.BaseClient != nil {
		return m.BaseClient.DeleteAccounts(ctx, ids...)
	}
	panic("MockClient.DeleteAccounts not implemented")
}
func (m *MockClient) DeletePublicKeys(ctx context.Context, ids ...ID) error {
	if m.Overwrites.DeletePublicKeys != nil {
		return m.Overwrites.DeletePublicKeys(ctx, ids...)
	} else if m.BaseClient != nil {
		return m.BaseClient.DeletePublicKeys(ctx, ids...)
	}
	panic("MockClient.DeletePublicKeys not implemented")
}
func (m *MockClient) DeleteTargets(ctx context.Context, ids ...ID) error {
	if m.Overwrites.DeleteTargets != nil {
		return m.Overwrites.DeleteTargets(ctx, ids...)
	} else if m.BaseClient != nil {
		return m.BaseClient.DeleteTargets(ctx, ids...)
	}
	panic("MockClient.DeleteTargets not implemented")
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
func (m *MockClient) DeployPublicKeys(ctx context.Context, publicKeyID ...ID) (chan DeployProgress, error) {
	if m.Overwrites.DeployPublicKeys != nil {
		return m.Overwrites.DeployPublicKeys(ctx, publicKeyID...)
	} else if m.BaseClient != nil {
		return m.BaseClient.DeployPublicKeys(ctx, publicKeyID...)
	}
	panic("MockClient.DeployPublicKeys not implemented")
}
func (m *MockClient) DeployTargets(ctx context.Context, targetID ...ID) (chan DeployProgress, error) {
	if m.Overwrites.DeployTargets != nil {
		return m.Overwrites.DeployTargets(ctx, targetID...)
	} else if m.BaseClient != nil {
		return m.BaseClient.DeployTargets(ctx, targetID...)
	}
	panic("MockClient.DeployTargets not implemented")
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
func (m *MockClient) GetDirtyAccounts(ctx context.Context) ([]Account, error) {
	if m.Overwrites.GetDirtyAccounts != nil {
		return m.Overwrites.GetDirtyAccounts(ctx)
	} else if m.BaseClient != nil {
		return m.BaseClient.GetDirtyAccounts(ctx)
	}
	panic("MockClient.GetDirtyAccounts not implemented")
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
func (m *MockClient) GetTarget(ctx context.Context, id ID) (Target, error) {
	if m.Overwrites.GetTarget != nil {
		return m.Overwrites.GetTarget(ctx, id)
	} else if m.BaseClient != nil {
		return m.BaseClient.GetTarget(ctx, id)
	}
	panic("MockClient.GetTarget not implemented")
}
func (m *MockClient) GetTargets(ctx context.Context, ids ...ID) ([]Target, error) {
	if m.Overwrites.GetTargets != nil {
		return m.Overwrites.GetTargets(ctx, ids...)
	} else if m.BaseClient != nil {
		return m.BaseClient.GetTargets(ctx, ids...)
	}
	panic("MockClient.GetTargets not implemented")
}
func (m *MockClient) LinkTagAccount(ctx context.Context, accountID ID, filter string, expiresAt time.Time) (ID, error) {
	if m.Overwrites.LinkTagAccount != nil {
		return m.Overwrites.LinkTagAccount(ctx, accountID, filter, expiresAt)
	} else if m.BaseClient != nil {
		return m.BaseClient.LinkTagAccount(ctx, accountID, filter, expiresAt)
	}
	panic("MockClient.LinkTagAccount not implemented")
}
func (m *MockClient) ListAccountsByTarget(ctx context.Context, targetID ID) ([]Account, error) {
	if m.Overwrites.ListAccountsByTarget != nil {
		return m.Overwrites.ListAccountsByTarget(ctx, targetID)
	} else if m.BaseClient != nil {
		return m.BaseClient.ListAccountsByTarget(ctx, targetID)
	}
	panic("MockClient.ListAccountsByTarget not implemented")
}
func (m *MockClient) ListPublicKeys(ctx context.Context, tagFilter string) ([]PublicKey, error) {
	if m.Overwrites.ListPublicKeys != nil {
		return m.Overwrites.ListPublicKeys(ctx, tagFilter)
	} else if m.BaseClient != nil {
		return m.BaseClient.ListPublicKeys(ctx, tagFilter)
	}
	panic("MockClient.ListPublicKeys not implemented")
}
func (m *MockClient) ListTargets(ctx context.Context) ([]Target, error) {
	if m.Overwrites.ListTargets != nil {
		return m.Overwrites.ListTargets(ctx)
	} else if m.BaseClient != nil {
		return m.BaseClient.ListTargets(ctx)
	}
	panic("MockClient.ListTargets not implemented")
}
func (m *MockClient) OnboardHost(ctx context.Context, host string, port int, accountName string, deploymentKey string) (chan OnboardHostProgress, error) {
	if m.Overwrites.OnboardHost != nil {
		return m.Overwrites.OnboardHost(ctx, host, port, accountName, deploymentKey)
	} else if m.BaseClient != nil {
		return m.BaseClient.OnboardHost(ctx, host, port, accountName, deploymentKey)
	}
	panic("MockClient.OnboardHost not implemented")
}
func (m *MockClient) ResolveAccountsForPublicKey(ctx context.Context, publicKeyID ID) ([]Account, error) {
	if m.Overwrites.ResolveAccountsForPublicKey != nil {
		return m.Overwrites.ResolveAccountsForPublicKey(ctx, publicKeyID)
	} else if m.BaseClient != nil {
		return m.BaseClient.ResolveAccountsForPublicKey(ctx, publicKeyID)
	}
	panic("MockClient.ResolveAccountsForPublicKey not implemented")
}
func (m *MockClient) ResolvePublicKeysForAccount(ctx context.Context, accountID ID) ([]PublicKey, error) {
	if m.Overwrites.ResolvePublicKeysForAccount != nil {
		return m.Overwrites.ResolvePublicKeysForAccount(ctx, accountID)
	} else if m.BaseClient != nil {
		return m.BaseClient.ResolvePublicKeysForAccount(ctx, accountID)
	}
	panic("MockClient.ResolvePublicKeysForAccount not implemented")
}
func (m *MockClient) UnLinkTagAccount(ctx context.Context, linkIDs ...ID) error {
	if m.Overwrites.UnLinkTagAccount != nil {
		return m.Overwrites.UnLinkTagAccount(ctx, linkIDs...)
	} else if m.BaseClient != nil {
		return m.BaseClient.UnLinkTagAccount(ctx, linkIDs...)
	}
	panic("MockClient.UnLinkTagAccount not implemented")
}
func (m *MockClient) UpdatePublicKeyTags(ctx context.Context, id ID, tags []string) error {
	if m.Overwrites.UpdatePublicKeyTags != nil {
		return m.Overwrites.UpdatePublicKeyTags(ctx, id, tags)
	} else if m.BaseClient != nil {
		return m.BaseClient.UpdatePublicKeyTags(ctx, id, tags)
	}
	panic("MockClient.UpdatePublicKeyTags not implemented")
}
func (m *MockClient) UpdateTarget(ctx context.Context, id ID, target Target) error {
	if m.Overwrites.UpdateTarget != nil {
		return m.Overwrites.UpdateTarget(ctx, id, target)
	} else if m.BaseClient != nil {
		return m.BaseClient.UpdateTarget(ctx, id, target)
	}
	panic("MockClient.UpdateTarget not implemented")
}
