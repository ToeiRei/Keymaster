// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core

import (
	"errors"
	"testing"

	"github.com/toeirei/keymaster/core/db"
	"github.com/toeirei/keymaster/core/model"
	"github.com/toeirei/keymaster/core/security"
	"github.com/toeirei/keymaster/i18n"
)

type fakeDeployer struct {
	deployed string
}

func (f *fakeDeployer) DeployAuthorizedKeys(content string) error { f.deployed = content; return nil }
func (f *fakeDeployer) GetAuthorizedKeys() ([]byte, error)        { return []byte(f.deployed), nil }
func (f *fakeDeployer) Close()                                    {}

func TestRunDeploymentForAccount_SetsSerial(t *testing.T) {
	if _, err := db.New("sqlite", ":memory:"); err != nil {
		t.Fatalf("db.New failed: %v", err)
	}
	// ensure i18n is initialized for messages
	i18n.Init("en")

	// Create an active system key so deployment can proceed.
	serial, err := db.CreateSystemKey("sys-pub-test", "sys-priv-test")
	if err != nil {
		t.Fatalf("CreateSystemKey failed: %v", err)
	}

	mgr := db.DefaultAccountManager()
	if mgr == nil {
		t.Fatal("no account manager")
	}
	acctID, err := mgr.AddAccount("deployuser", "example.test", "lbl", "")
	if err != nil {
		t.Fatalf("AddAccount failed: %v", err)
	}

	// Override deployer factory with a fake that records content.
	orig := NewDeployerFactory
	NewDeployerFactory = func(host, user string, privateKey security.Secret, passphrase []byte) (RemoteDeployer, error) {
		return &fakeDeployer{}, nil
	}
	defer func() { NewDeployerFactory = orig }()

	acct := model.Account{ID: acctID, Username: "deployuser", Hostname: "example.test", Serial: 0}
	if err := RunDeploymentForAccount(acct, false); err != nil {
		t.Fatalf("RunDeploymentForAccount failed: %v", err)
	}

	// Verify serial was updated to the active system key serial.
	accts, err := db.GetAllAccounts()
	if err != nil {
		t.Fatalf("GetAllAccounts failed: %v", err)
	}
	var found *model.Account
	for _, a := range accts {
		if a.ID == acctID {
			found = &a
			break
		}
	}
	if found == nil {
		t.Fatalf("account not found after deploy")
	}
	if found.Serial != serial {
		t.Fatalf("expected serial %d, got %d", serial, found.Serial)
	}
}

func TestRunDeploymentForAccount_ConnectionError(t *testing.T) {
	if _, err := db.New("sqlite", ":memory:"); err != nil {
		t.Fatalf("db.New failed: %v", err)
	}
	i18n.Init("en")

	mgr := db.DefaultAccountManager()
	if mgr == nil {
		t.Fatal("no account manager")
	}
	acctID, err := mgr.AddAccount("failuser", "fail.test", "lbl", "")
	if err != nil {
		t.Fatalf("AddAccount failed: %v", err)
	}

	// Override factory to return an error simulating connection failure.
	orig := NewDeployerFactory
	NewDeployerFactory = func(host, user string, privateKey security.Secret, passphrase []byte) (RemoteDeployer, error) {
		return nil, errors.New("connect failed")
	}
	defer func() { NewDeployerFactory = orig }()

	acct := model.Account{ID: acctID, Username: "failuser", Hostname: "fail.test", Serial: 0}
	if err := RunDeploymentForAccount(acct, false); err == nil {
		t.Fatalf("expected connection error, got nil")
	}
}
