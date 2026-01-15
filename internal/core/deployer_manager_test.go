// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core

import (
	"testing"

	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/model"
)

type fakeGetterDeployer struct {
	content []byte
}

func (f *fakeGetterDeployer) DeployAuthorizedKeys(content string) error { return nil }
func (f *fakeGetterDeployer) GetAuthorizedKeys() ([]byte, error)        { return f.content, nil }
func (f *fakeGetterDeployer) Close()                                    {}

func TestBuiltinDeployerManager_FetchAuthorizedKeys(t *testing.T) {
	if err := db.InitDB("sqlite", ":memory:"); err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	// create active system key
	if _, err := db.CreateSystemKey("pubdata", "privdata"); err != nil {
		t.Fatalf("CreateSystemKey failed: %v", err)
	}

	// override NewDeployerFactory to return a fake deployer
	orig := NewDeployerFactory
	NewDeployerFactory = func(host, user, privateKey string, passphrase []byte) (RemoteDeployer, error) {
		return &fakeGetterDeployer{content: []byte("authorized-keys-content")}, nil
	}
	defer func() { NewDeployerFactory = orig }()

	acct := model.Account{ID: 1, Username: "u", Hostname: "h", Serial: 0}
	dm := builtinDeployerManager{}
	out, err := dm.FetchAuthorizedKeys(acct)
	if err != nil {
		t.Fatalf("FetchAuthorizedKeys failed: %v", err)
	}
	if string(out) != "authorized-keys-content" {
		t.Fatalf("unexpected content: %q", string(out))
	}
}

func TestPerformDecommissionWithKeys_Delegates(t *testing.T) {
	acc := model.Account{ID: 1, Username: "u"}
	called := false
	decomm := func(a model.Account, keep map[int]bool) (DecommissionResult, error) {
		called = true
		return DecommissionResult{Account: a, AccountID: a.ID, Skipped: false}, nil
	}
	res, err := PerformDecommissionWithKeys(acc, map[int]bool{1: true}, decomm)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatalf("expected decommander to be called")
	}
	if res.AccountID != acc.ID {
		t.Fatalf("unexpected result account id: %d", res.AccountID)
	}
}

