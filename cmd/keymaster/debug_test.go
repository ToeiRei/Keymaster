// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package main

import (
	"fmt"
	"testing"

	"github.com/toeirei/keymaster/internal/db"
)

func TestDebugAddAccount(t *testing.T) {
	dbdsn := "file:TestDebugAddAccount?mode=memory&cache=shared"
	// configure viper indirectly by calling db.InitDB (simulate setupTestDB)
	if err := db.InitDB("sqlite", dbdsn); err != nil {
		t.Fatalf("InitDB failed: %v", err)
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
