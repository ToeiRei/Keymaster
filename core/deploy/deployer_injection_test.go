// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package deploy_test

import (
	"strings"
	"testing"

	"github.com/toeirei/keymaster/internal/core"
	"github.com/toeirei/keymaster/internal/core/db"
	"github.com/toeirei/keymaster/internal/core/model"
	"github.com/toeirei/keymaster/internal/core/security"
)

// reuse fakeDeployer defined in other tests (same package)

// TestRunDeploymentWithInjectedDeployer verifies we can inject a fake deployer
// via core.NewDeployerFactory to avoid real network connections during deployment.
func TestRunDeploymentWithInjectedDeployer(t *testing.T) {
	// Init DB and create account via DefaultAccountManager
	if _, err := db.New("sqlite", ":memory:"); err != nil {
		t.Fatalf("db.New failed: %v", err)
	}

	mgr := db.DefaultAccountManager()
	if mgr == nil {
		t.Fatalf("no account manager available")
	}
	accID, err := mgr.AddAccount("deployer", "example.local", "lbl", "")
	if err != nil {
		t.Fatalf("AddAccount failed: %v", err)
	}

	// Create an active system key so GenerateKeysContent has something to include
	if _, err := db.CreateSystemKey("sys-pub-inject", "sys-priv-inject"); err != nil {
		t.Fatalf("CreateSystemKey failed: %v", err)
	}

	// Override core factory
	orig := core.NewDeployerFactory
	fake := &fakeDeployer{}
	core.NewDeployerFactory = func(host, user string, privateKey security.Secret, passphrase []byte) (core.RemoteDeployer, error) {
		return fake, nil
	}
	defer func() { core.NewDeployerFactory = orig }()

	acct := model.Account{ID: accID, Username: "deployer", Hostname: "example.local", Serial: 0}
	if err := core.RunDeploymentForAccount(acct, false); err != nil {
		t.Fatalf("RunDeploymentForAccount failed: %v", err)
	}

	if !strings.Contains(fake.seen, "sys-pub-inject") {
		t.Fatalf("authorized_keys did not include system key: %q", fake.seen)
	}
}
