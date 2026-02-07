// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package cli

// TODO: consider removal — OWNER_APPROVAL_REQUIRED
// Candidate: local DB-backed debug test; keep only if owner confirms it's useful.

import (
	"fmt"
	"testing"

	"github.com/spf13/viper"
	"github.com/toeirei/keymaster/core"
)

func TestDebugAddAccount(t *testing.T) {
	setupTestDB(t)
	mgr := core.DefaultAccountManager()
	if mgr == nil {
		t.Fatalf("no account manager available")
	}
	id, err := mgr.AddAccount("user1", "host1.com", "prod-web-1", "")
	if err != nil {
		t.Fatalf("AddAccount returned error: %v", err)
	}
	fmt.Printf("AddAccount id=%d\n", id)
	dsn := viper.GetString("database.dsn")
	st, err := core.NewStoreFromDSN("sqlite", dsn)
	if err != nil {
		t.Fatalf("NewStoreFromDSN failed: %v", err)
	}
	defer func() { _ = core.CloseStore(st) }()
	accts, err := st.GetAllActiveAccounts()
	if err != nil {
		t.Fatalf("GetAllActiveAccounts failed: %v", err)
	}
	fmt.Printf("Got %d active accounts\n", len(accts))
	for _, a := range accts {
		fmt.Printf("acct: %+v\n", a)
	}
}
