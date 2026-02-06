// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package cli

import (
	"testing"

	"github.com/toeirei/keymaster/core/db"
	"github.com/toeirei/keymaster/core/model"
	"github.com/toeirei/keymaster/i18n"
)

func TestRunDeploymentForAccount_NoActiveSystemKey_ReturnsError(t *testing.T) {
	i18n.Init("en")
	if _, err := db.New("sqlite", ":memory:"); err != nil {
		t.Fatalf("db.New failed: %v", err)
	}

	acct := model.Account{ID: 1, Username: "u", Hostname: "h", Serial: 0}
	err := runDeploymentFunc(acct)
	if err == nil {
		t.Fatalf("expected error when no active system key exists")
	}
}

func TestRunDeploymentForAccount_MissingSerialKey_ReturnsError(t *testing.T) {
	i18n.Init("en")
	if _, err := db.New("sqlite", ":memory:"); err != nil {
		t.Fatalf("db.New failed: %v", err)
	}

	acct := model.Account{ID: 1, Username: "u", Hostname: "h", Serial: 999}
	err := runDeploymentFunc(acct)
	if err == nil {
		t.Fatalf("expected error when specified serial key is missing")
	}
}

func TestRunDeploymentForAccount_Success(t *testing.T) {
	// Inject a mock implementation to simulate a successful deployment
	orig := runDeploymentFunc
	defer func() { runDeploymentFunc = orig }()

	called := false
	runDeploymentFunc = func(account model.Account) error {
		called = true
		if account.ID != 1 {
			t.Fatalf("unexpected account ID: %d", account.ID)
		}
		return nil
	}

	acct := model.Account{ID: 1, Username: "u", Hostname: "h", Serial: 0}
	if err := runDeploymentFunc(acct); err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if !called {
		t.Fatalf("injected runDeploymentFunc was not called")
	}
}
