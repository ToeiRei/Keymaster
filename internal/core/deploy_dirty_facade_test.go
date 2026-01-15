// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core

import (
	"context"
	"testing"

	"github.com/toeirei/keymaster/internal/model"
)

type fakeStoreForDirty struct {
	accounts []model.Account
	cleared  []int
}

func (f *fakeStoreForDirty) GetAllActiveAccounts() ([]model.Account, error) { return f.accounts, nil }
func (f *fakeStoreForDirty) UpdateAccountIsDirty(id int, dirty bool) error {
	if !dirty {
		f.cleared = append(f.cleared, id)
	}
	return nil
}

// implement remaining Store methods as no-ops to satisfy interface
func (f *fakeStoreForDirty) GetAccounts() ([]model.Account, error)     { return nil, nil }
func (f *fakeStoreForDirty) GetAllAccounts() ([]model.Account, error)  { return nil, nil }
func (f *fakeStoreForDirty) GetAccount(id int) (*model.Account, error) { return nil, nil }
func (f *fakeStoreForDirty) AddAccount(username, hostname, label, tags string) (int, error) {
	return 0, nil
}
func (f *fakeStoreForDirty) DeleteAccount(accountID int) error                         { return nil }
func (f *fakeStoreForDirty) AssignKeyToAccount(keyID, accountID int) error             { return nil }
func (f *fakeStoreForDirty) CreateSystemKey(publicKey, privateKey string) (int, error) { return 0, nil }
func (f *fakeStoreForDirty) RotateSystemKey(publicKey, privateKey string) (int, error) { return 0, nil }
func (f *fakeStoreForDirty) GetActiveSystemKey() (*model.SystemKey, error)             { return nil, nil }
func (f *fakeStoreForDirty) AddKnownHostKey(hostname, key string) error                { return nil }
func (f *fakeStoreForDirty) ExportDataForBackup() (*model.BackupData, error)           { return nil, nil }
func (f *fakeStoreForDirty) ImportDataFromBackup(*model.BackupData) error              { return nil }
func (f *fakeStoreForDirty) IntegrateDataFromBackup(*model.BackupData) error           { return nil }

type fakeDMForDirty struct{ called []int }

func (f *fakeDMForDirty) DeployForAccount(account model.Account, keepFile bool) error {
	f.called = append(f.called, account.ID)
	return nil
}
func (f *fakeDMForDirty) AuditSerial(account model.Account) error { return nil }
func (f *fakeDMForDirty) AuditStrict(account model.Account) error { return nil }
func (f *fakeDMForDirty) DecommissionAccount(account model.Account, systemPrivateKey string, options interface{}) (DecommissionResult, error) {
	return DecommissionResult{}, nil
}
func (f *fakeDMForDirty) BulkDecommissionAccounts(accounts []model.Account, systemPrivateKey string, options interface{}) ([]DecommissionResult, error) {
	return nil, nil
}
func (f *fakeDMForDirty) CanonicalizeHostPort(host string) string                   { return host }
func (f *fakeDMForDirty) ParseHostPort(host string) (string, string, error)         { return host, "22", nil }
func (f *fakeDMForDirty) GetRemoteHostKey(host string) (string, error)              { return "", nil }
func (f *fakeDMForDirty) FetchAuthorizedKeys(account model.Account) ([]byte, error) { return nil, nil }
func (f *fakeDMForDirty) ImportRemoteKeys(account model.Account) ([]model.PublicKey, int, string, error) {
	return nil, 0, "", nil
}
func (f *fakeDMForDirty) IsPassphraseRequired(err error) bool { return false }

func TestDeployDirtyAccounts_ClearsOnSuccess(t *testing.T) {
	st := &fakeStoreForDirty{accounts: []model.Account{{ID: 1, IsDirty: false}, {ID: 2, IsDirty: true}, {ID: 3, IsDirty: true}}}
	dm := &fakeDMForDirty{}

	res, err := DeployDirtyAccounts(context.Background(), st, dm, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) != 2 {
		t.Fatalf("expected 2 results, got %d", len(res))
	}
	if len(dm.called) != 2 || dm.called[0] != 2 || dm.called[1] != 3 {
		t.Fatalf("unexpected deploy calls: %v", dm.called)
	}
	if len(st.cleared) != 2 {
		t.Fatalf("expected 2 cleared flags, got %d", len(st.cleared))
	}
}

