package client

import (
	"context"
	"io"
	"log"
	"testing"
	"time"

	"github.com/toeirei/keymaster/config"
	"github.com/toeirei/keymaster/core"
)

func TestBunClient_TagLinkAndResolve(t *testing.T) {
	cfg := config.Config{Database: config.ConfigDatabase{Type: "sqlite", Dsn: ":memory:"}}
	logger := log.New(io.Discard, "", 0)

	c, err := NewBunClient(cfg, logger)
	if err != nil {
		t.Fatalf("NewBunClient failed: %v", err)
	}
	defer func() { _ = c.Close(context.Background()) }()

	ctx := context.Background()

	// create target/account
	tgt, err := c.CreateTarget(ctx, "link.example.com", 22)
	if err != nil {
		t.Fatalf("CreateTarget failed: %v", err)
	}
	acct, err := c.CreateAccount(ctx, tgt.Id, "alice", "")
	if err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}

	// create a public key and assign to account
	pk, err := c.CreatePublicKey(ctx, "alice-key", nil)
	if err != nil {
		t.Fatalf("CreatePublicKey failed: %v", err)
	}
	km := core.DefaultKeyManager()
	if km == nil {
		t.Fatalf("no key manager")
	}
	if err := km.AssignKeyToAccount(int(pk.Id), int(acct.Id)); err != nil {
		t.Fatalf("AssignKeyToAccount failed: %v", err)
	}

	// ResolvePublicKeysForAccount should include this key
	keys, err := c.ResolvePublicKeysForAccount(ctx, acct.Id)
	if err != nil {
		t.Fatalf("ResolvePublicKeysForAccount failed: %v", err)
	}
	found := false
	for _, k := range keys {
		if k.Id == pk.Id {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected assigned key in resolved keys")
	}

	// ResolveAccountsForPublicKey should return the account
	accts, err := c.ResolveAccountsForPublicKey(ctx, pk.Id)
	if err != nil {
		t.Fatalf("ResolveAccountsForPublicKey failed: %v", err)
	}
	if len(accts) == 0 || accts[0].Id != acct.Id {
		t.Fatalf("ResolveAccountsForPublicKey unexpected: %#v", accts)
	}

	// LinkTagAccount / UnLinkTagAccount basic behavior
	linkID, err := c.LinkTagAccount(ctx, acct.Id, "env:prod", time.Now().Add(24*time.Hour))
	if err != nil {
		t.Fatalf("LinkTagAccount failed: %v", err)
	}
	if linkID == 0 {
		t.Fatalf("expected non-zero link id")
	}
	if err := c.UnLinkTagAccount(ctx, linkID); err != nil {
		t.Fatalf("UnLinkTagAccount failed: %v", err)
	}
}
