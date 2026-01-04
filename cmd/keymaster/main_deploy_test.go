package main

import (
	"testing"

	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/i18n"
	"github.com/toeirei/keymaster/internal/model"
)

func TestRunDeploymentForAccount_NoActiveSystemKey_ReturnsError(t *testing.T) {
	i18n.Init("en")
	if err := db.InitDB("sqlite", ":memory:"); err != nil {
		t.Fatalf("db.InitDB failed: %v", err)
	}

	acct := model.Account{ID: 1, Username: "u", Hostname: "h", Serial: 0}
	err := runDeploymentForAccount(acct)
	if err == nil {
		t.Fatalf("expected error when no active system key exists")
	}
}

func TestRunDeploymentForAccount_MissingSerialKey_ReturnsError(t *testing.T) {
	i18n.Init("en")
	if err := db.InitDB("sqlite", ":memory:"); err != nil {
		t.Fatalf("db.InitDB failed: %v", err)
	}

	acct := model.Account{ID: 1, Username: "u", Hostname: "h", Serial: 999}
	err := runDeploymentForAccount(acct)
	if err == nil {
		t.Fatalf("expected error when specified serial key is missing")
	}
}
