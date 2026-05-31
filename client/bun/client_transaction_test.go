// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package bun_test

import (
	"context"
	"errors"
	"io"
	"log"
	"testing"

	"github.com/toeirei/keymaster/client"
	"github.com/toeirei/keymaster/client/bun"
	"github.com/toeirei/keymaster/config"
)

func TestBunClient_WithTransaction_CommitsOnSuccess(t *testing.T) {
	cfg := config.Config{Database: config.ConfigDatabase{Type: "sqlite", Dsn: ":memory:"}}
	logger := log.New(io.Discard, "", 0)

	c, err := bun.NewBunClient(cfg, logger)
	if err != nil {
		t.Fatalf("NewBunClient failed: %v", err)
	}
	defer func() { _ = c.Close(context.Background()) }()

	ctx := context.Background()
	if err := c.WithTransaction(ctx, func(tx client.Client) error {
		_, err := tx.CreateAccount(ctx, "alice", "example.com", 22, "ssh", "")
		return err
	}); err != nil {
		t.Fatalf("WithTransaction failed: %v", err)
	}

	accounts, err := c.ListAccounts(ctx)
	if err != nil {
		t.Fatalf("ListAccounts failed: %v", err)
	}
	if len(accounts) != 1 {
		t.Fatalf("expected 1 account after commit, got %d", len(accounts))
	}
}

func TestBunClient_WithTransaction_RollsBackOnError(t *testing.T) {
	cfg := config.Config{Database: config.ConfigDatabase{Type: "sqlite", Dsn: ":memory:"}}
	logger := log.New(io.Discard, "", 0)

	c, err := bun.NewBunClient(cfg, logger)
	if err != nil {
		t.Fatalf("NewBunClient failed: %v", err)
	}
	defer func() { _ = c.Close(context.Background()) }()

	ctx := context.Background()
	expected := errors.New("boom")
	if err := c.WithTransaction(ctx, func(tx client.Client) error {
		if _, err := tx.CreateAccount(ctx, "bob", "rollback.example", 22, "ssh", ""); err != nil {
			return err
		}
		return expected
	}); !errors.Is(err, expected) {
		t.Fatalf("expected rollback error %q, got: %v", expected, err)
	}

	accounts, err := c.ListAccounts(ctx)
	if err != nil {
		t.Fatalf("ListAccounts failed: %v", err)
	}
	if len(accounts) != 0 {
		t.Fatalf("expected 0 accounts after rollback, got %d", len(accounts))
	}
}
