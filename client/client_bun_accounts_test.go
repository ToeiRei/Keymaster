package client

import (
	"context"
	"io"
	"log"
	"testing"

	"github.com/toeirei/keymaster/config"
)

func TestBunClient_AccountsCRUD(t *testing.T) {
	cfg := config.Config{Database: config.ConfigDatabase{Type: "sqlite", Dsn: ":memory:"}}
	logger := log.New(io.Discard, "", 0)

	c, err := NewBunClient(cfg, logger)
	if err != nil {
		t.Fatalf("NewBunClient failed: %v", err)
	}
	defer func() { _ = c.Close(context.Background()) }()

	ctx := context.Background()

	// Create target then account
	tgt, err := c.CreateTarget(ctx, "acct.example.com", 22)
	if err != nil {
		t.Fatalf("CreateTarget failed: %v", err)
	}

	acct, err := c.CreateAccount(ctx, tgt.Id, "deploy", "")
	if err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}
	if acct.Id == 0 {
		t.Fatalf("expected non-zero account id")
	}
	if acct.Name != "deploy" {
		t.Fatalf("unexpected account name: %s", acct.Name)
	}

	// GetAccount
	got, err := c.GetAccount(ctx, acct.Id)
	if err != nil {
		t.Fatalf("GetAccount failed: %v", err)
	}
	if got.Id != acct.Id || got.Name != acct.Name {
		t.Fatalf("GetAccount mismatch: %#v vs %#v", got, acct)
	}

	// ListAccountsByTarget
	list, err := c.ListAccountsByTarget(ctx, tgt.Id)
	if err != nil {
		t.Fatalf("ListAccountsByTarget failed: %v", err)
	}
	if len(list) != 1 || list[0].Id != acct.Id {
		t.Fatalf("ListAccountsByTarget unexpected: %#v", list)
	}

	// GetAccounts
	many, err := c.GetAccounts(ctx, acct.Id)
	if err != nil {
		t.Fatalf("GetAccounts failed: %v", err)
	}
	if len(many) != 1 || many[0].Id != acct.Id {
		t.Fatalf("GetAccounts unexpected: %#v", many)
	}

	// DeleteAccounts
	if err := c.DeleteAccounts(ctx, acct.Id); err != nil {
		t.Fatalf("DeleteAccounts failed: %v", err)
	}
	after, err := c.ListAccountsByTarget(ctx, tgt.Id)
	if err != nil {
		t.Fatalf("ListAccountsByTarget after delete failed: %v", err)
	}
	if len(after) != 0 {
		t.Fatalf("expected zero accounts after delete, got: %#v", after)
	}
}
