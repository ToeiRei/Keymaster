// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core

import (
	"context"
	"testing"

	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/model"
	"github.com/toeirei/keymaster/internal/testutil"
)

// Test core-level transfer end-to-end: build package and accept it using DB-backed deps.
func TestCoreTransfer_EndToEnd(t *testing.T) {
	if err := db.InitDB("sqlite", ":memory:"); err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}

	fake := &testutil.FakeAuditWriter{}
	db.SetDefaultAuditWriter(fake)
	defer db.ClearDefaultAuditWriter()

	pkg, err := BuildTransferPackage("alice", "example.test", "lbl", "")
	if err != nil {
		t.Fatalf("BuildTransferPackage failed: %v", err)
	}

	// Use DB-backed AddAccount so account is actually persisted.
	deps := BootstrapDeps{
		AddAccount: func(u, h, l, t string) (int, error) {
			mgr := db.DefaultAccountManager()
			return mgr.AddAccount(u, h, l, t)
		},
		AssignKey: func(kid, aid int) error { return nil },
		GenerateKeysContent: func(accountID int) (string, error) {
			return "ssh-ed25519 AAA... test", nil
		},
		NewBootstrapDeployer: func(hostname, username, privateKey, expectedHostKey string) (BootstrapDeployer, error) {
			return &fakeDeployer{}, nil
		},
		GetActiveSystemKey: func() (*model.SystemKey, error) { return nil, nil },
		LogAudit:           func(e BootstrapAuditEvent) error { return nil },
	}

	res, err := AcceptTransferPackage(context.Background(), pkg, deps)
	if err != nil {
		t.Fatalf("AcceptTransferPackage failed: %v", err)
	}
	if res.Account.Username != "alice" || res.Account.Hostname != "example.test" {
		t.Fatalf("unexpected account in result: %+v", res.Account)
	}

	// Ensure account exists in DB
	accts, err := db.GetAllAccounts()
	if err != nil {
		t.Fatalf("GetAllAccounts failed: %v", err)
	}
	found := false
	for _, a := range accts {
		if a.Username == "alice" && a.Hostname == "example.test" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected account alice@example.test in DB")
	}
}

