// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core_test

import (
	"testing"

	"github.com/toeirei/keymaster/internal/core"
	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/model"
	"github.com/toeirei/keymaster/internal/security"
)

type localFakeDeployer struct{ content []byte }

func (l *localFakeDeployer) DeployAuthorizedKeys(content string) error { return nil }
func (l *localFakeDeployer) GetAuthorizedKeys() ([]byte, error)        { return l.content, nil }
func (l *localFakeDeployer) Close()                                    {}

func TestImportRemoteKeys_AddsKey(t *testing.T) {
	if err := db.InitDB("sqlite", ":memory:"); err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}

	mgr := db.DefaultAccountManager()
	if mgr == nil {
		t.Fatal("no account manager")
	}
	acctID, err := mgr.AddAccount("impuser", "example.test", "lbl", "")
	if err != nil {
		t.Fatalf("AddAccount failed: %v", err)
	}

	// Override factory to return the fake deployer with one public key line.
	orig := core.NewDeployerFactory
	core.NewDeployerFactory = func(host, user string, privateKey security.Secret, passphrase []byte) (core.RemoteDeployer, error) {
		return &localFakeDeployer{content: []byte("ssh-ed25519 AAAAB3NzaC1lZDI1NTE5AAAAITestKey comment@example.com\n")}, nil
	}
	defer func() { core.NewDeployerFactory = orig }()

	acct := model.Account{ID: acctID, Username: "impuser", Hostname: "example.test", Serial: 0}
	imported, skipped, warn, err := core.ImportRemoteKeys(acct)
	if err != nil {
		t.Fatalf("ImportRemoteKeys failed: %v (warn=%s)", err, warn)
	}
	if skipped != 0 {
		t.Fatalf("expected 0 skipped, got %d", skipped)
	}
	if len(imported) != 1 {
		t.Fatalf("expected 1 imported key, got %d", len(imported))
	}
}
