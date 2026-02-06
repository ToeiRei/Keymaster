// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core

import (
	"testing"

	"github.com/toeirei/keymaster/core/model"
	"github.com/toeirei/keymaster/core/security"
)

type fakeDM struct {
	called []int
}

func (f *fakeDM) DeployForAccount(account model.Account, keepFile bool) error {
	f.called = append(f.called, account.ID)
	return nil
}
func (f *fakeDM) AuditSerial(account model.Account) error { return nil }
func (f *fakeDM) AuditStrict(account model.Account) error { return nil }
func (f *fakeDM) DecommissionAccount(account model.Account, systemPrivateKey security.Secret, options interface{}) (DecommissionResult, error) {
	return DecommissionResult{}, nil
}
func (f *fakeDM) BulkDecommissionAccounts(accounts []model.Account, systemPrivateKey security.Secret, options interface{}) ([]DecommissionResult, error) {
	return nil, nil
}
func (f *fakeDM) CanonicalizeHostPort(host string) string                   { return host }
func (f *fakeDM) ParseHostPort(host string) (string, string, error)         { return host, "22", nil }
func (f *fakeDM) GetRemoteHostKey(host string) (string, error)              { return "", nil }
func (f *fakeDM) FetchAuthorizedKeys(account model.Account) ([]byte, error) { return nil, nil }
func (f *fakeDM) ImportRemoteKeys(account model.Account) ([]model.PublicKey, int, string, error) {
	return nil, 0, "", nil
}
func (f *fakeDM) IsPassphraseRequired(err error) bool { return false }

func TestDirtyAccountsAndDeployList(t *testing.T) {
	accounts := []model.Account{
		{ID: 1, Username: "a", Hostname: "h1", IsDirty: false},
		{ID: 2, Username: "b", Hostname: "h2", IsDirty: true},
		{ID: 3, Username: "c", Hostname: "h3", IsDirty: true},
	}

	dirty := DirtyAccounts(accounts)
	if len(dirty) != 2 || dirty[0].ID != 2 || dirty[1].ID != 3 {
		t.Fatalf("unexpected dirty accounts: %+v", dirty)
	}

	f := &fakeDM{}
	results := DeployList(f, dirty)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if len(f.called) != 2 || f.called[0] != 2 || f.called[1] != 3 {
		t.Fatalf("deploy called for unexpected ids: %v", f.called)
	}
}
