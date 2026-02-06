// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package deploy_test

import (
	"testing"

	"github.com/toeirei/keymaster/core"
	"github.com/toeirei/keymaster/core/db"
	"github.com/toeirei/keymaster/core/model"
	"github.com/toeirei/keymaster/core/security"
	"github.com/toeirei/keymaster/i18n"
)

func TestAuditAccountSerial_Errors(t *testing.T) {
	if _, err := db.New("sqlite", ":memory:"); err != nil {
		t.Fatalf("db.New failed: %v", err)
	}
	i18n.Init("en")

	mgr := db.DefaultAccountManager()
	if mgr == nil {
		t.Fatal("no account manager")
	}
	acctID, err := mgr.AddAccount("u1", "host.local", "lbl", "")
	if err != nil {
		t.Fatalf("AddAccount failed: %v", err)
	}

	// Case: Serial == 0 should return an error
	acct := model.Account{ID: acctID, Username: "u1", Hostname: "host.local", Serial: 0}
	if err := core.AuditAccountSerial(acct); err == nil {
		t.Fatalf("expected error for serial==0, got nil")
	}

	// Case: Serial set but no system key in DB should return an error without network
	if err := db.UpdateAccountSerial(acctID, 999); err != nil {
		t.Fatalf("UpdateAccountSerial failed: %v", err)
	}
	acct.Serial = 999
	if err := core.AuditAccountSerial(acct); err == nil {
		t.Fatalf("expected error for missing system key, got nil")
	}
}

func TestAuditAccountStrict_DriftDetected(t *testing.T) {
	if _, err := db.New("sqlite", ":memory:"); err != nil {
		t.Fatalf("db.New failed: %v", err)
	}
	i18n.Init("en")

	mgr := db.DefaultAccountManager()
	if mgr == nil {
		t.Fatal("no account manager")
	}
	acctID, err := mgr.AddAccount("u2", "host2.local", "lbl", "")
	if err != nil {
		t.Fatalf("AddAccount failed: %v", err)
	}

	serial, err := db.CreateSystemKey("sys-pub-audit2", "sys-priv-audit2")
	if err != nil {
		t.Fatalf("CreateSystemKey failed: %v", err)
	}
	if err := db.UpdateAccountSerial(acctID, serial); err != nil {
		t.Fatalf("UpdateAccountSerial failed: %v", err)
	}

	// Remote content differs from generated expected content -- override core factory
	origFactory := core.NewDeployerFactory
	core.NewDeployerFactory = func(host, user string, privateKey security.Secret, passphrase []byte) (core.RemoteDeployer, error) {
		return &fakeDeployer{content: []byte("some other content")}, nil
	}
	defer func() { core.NewDeployerFactory = origFactory }()

	acct := model.Account{ID: acctID, Username: "u2", Hostname: "host2.local", Serial: serial}
	if err := core.AuditAccountStrict(acct); err == nil {
		t.Fatalf("expected AuditAccountStrict to detect drift, but it returned nil")
	}
}

func TestAuditAccountStrict_Match(t *testing.T) {
	if _, err := db.New("sqlite", ":memory:"); err != nil {
		t.Fatalf("db.New failed: %v", err)
	}
	i18n.Init("en")

	mgr := db.DefaultAccountManager()
	if mgr == nil {
		t.Fatal("no account manager")
	}
	acctID, err := mgr.AddAccount("u3", "host3.local", "lbl", "")
	if err != nil {
		t.Fatalf("AddAccount failed: %v", err)
	}

	serial, err := db.CreateSystemKey("sys-pub-audit3", "sys-priv-audit3")
	if err != nil {
		t.Fatalf("CreateSystemKey failed: %v", err)
	}
	if err := db.UpdateAccountSerial(acctID, serial); err != nil {
		t.Fatalf("UpdateAccountSerial failed: %v", err)
	}

	// Generate expected content using core helper
	expectedContent, err := core.GenerateKeysContent(acctID)
	if err != nil {
		t.Fatalf("GenerateKeysContent failed: %v", err)
	}

	// Override deployer factory to return the expected content
	origFactory := core.NewDeployerFactory
	core.NewDeployerFactory = func(host, user string, privateKey security.Secret, passphrase []byte) (core.RemoteDeployer, error) {
		return &fakeDeployer{content: []byte(expectedContent)}, nil
	}
	defer func() { core.NewDeployerFactory = origFactory }()

	acct := model.Account{ID: acctID, Username: "u3", Hostname: "host3.local", Serial: serial}
	if err := core.AuditAccountStrict(acct); err != nil {
		t.Fatalf("expected AuditAccountStrict to succeed with matching content, but got error: %v", err)
	}
}

// bytesFromString is provided by testhelpers_test.go
