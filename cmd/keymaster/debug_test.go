// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package main

// TODO: consider removal — OWNER_APPROVAL_REQUIRED
// Candidate: local DB-backed debug test; keep only if owner confirms it's useful.

import (
	"fmt"
	"testing"

	"github.com/toeirei/keymaster/internal/core/db"
)

func TestDebugAddAccount(t *testing.T) {
	dbdsn := "file:TestDebugAddAccount?mode=memory&cache=shared"
	// configure viper indirectly by calling db.New (simulate setupTestDB)
	if _, err := db.New("sqlite", dbdsn); err != nil {
		t.Fatalf("db.New failed: %v", err)
	}
	mgr := db.DefaultAccountManager()
	if mgr == nil {
		t.Fatalf("no account manager available")
	}
	id, err := mgr.AddAccount("user1", "host1.com", "prod-web-1", "")
	if err != nil {
		t.Fatalf("AddAccount returned error: %v", err)
	}
	fmt.Printf("AddAccount id=%d\n", id)
	accts, err := db.GetAllActiveAccounts()
	if err != nil {
		t.Fatalf("GetAllActiveAccounts failed: %v", err)
	}
	fmt.Printf("Got %d active accounts\n", len(accts))
	for _, a := range accts {
		fmt.Printf("acct: %+v\n", a)
	}
}
