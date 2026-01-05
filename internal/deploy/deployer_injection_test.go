// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package deploy

import (
	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/model"
	"testing"
)

// TestRunDeploymentWithInjectedDeployer verifies we can inject a fake deployer
// via NewDeployerFunc to avoid real network connections during deployment.
func TestRunDeploymentWithInjectedDeployer(t *testing.T) {
	// Init DB and create account via DefaultAccountManager
	if err := db.InitDB("sqlite", ":memory:"); err != nil {
		t.Fatalf("InitDB failed: %v", err)
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

	// Prepare mock sftp client and inject via NewDeployerFunc
	mock := newMockSftpClient()
	origFactory := NewDeployerFunc
	NewDeployerFunc = func(host, user, privateKey string, passphrase []byte) (*Deployer, error) {
		return &Deployer{sftp: mock, client: nil, config: DefaultConnectionConfig()}, nil
	}
	defer func() { NewDeployerFunc = origFactory }()

	acct := model.Account{ID: accID, Username: "deployer", Hostname: "example.local", Serial: 0}
	if err := RunDeploymentForAccount(acct, false); err != nil {
		t.Fatalf("RunDeploymentForAccount failed: %v", err)
	}

	// Verify authorized_keys was written
	f, ok := mock.files[".ssh/authorized_keys"]
	if !ok {
		t.Fatalf("expected authorized_keys to be created")
	}
	if !contains(f.Buffer.String(), "sys-pub-inject") {
		t.Fatalf("authorized_keys did not include system key: %q", f.Buffer.String())
	}
}
