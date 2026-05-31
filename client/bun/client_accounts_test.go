// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package bun_test

import (
	"context"
	"io"
	"log"
	"testing"

	"github.com/toeirei/keymaster/client/bun"
	"github.com/toeirei/keymaster/config"
)

func TestBunClient_CreateAndGetAccount(t *testing.T) {
	cfg := config.Config{Database: config.ConfigDatabase{Type: "sqlite", Dsn: ":memory:"}}
	logger := log.New(io.Discard, "", 0)

	client, err := bun.NewBunClient(cfg, logger)
	if err != nil {
		t.Fatalf("NewBunClient failed: %v", err)
	}
	defer func() { _ = client.Close(context.Background()) }()

	ctx := context.Background()

	// Create an account
	acc, err := client.CreateAccount(ctx, "deploy", "example.com", 22, "ssh", "")
	if err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}

	if acc.Id == 0 {
		t.Fatal("expected non-zero account ID")
	}
	if acc.Username != "deploy" {
		t.Errorf("expected username 'deploy', got %s", acc.Username)
	}
	if acc.Host != "example.com" {
		t.Errorf("expected host 'example.com', got %s", acc.Host)
	}
	if acc.Port != 22 {
		t.Errorf("expected port 22, got %d", acc.Port)
	}

	// GetAccount
	got, err := client.GetAccount(ctx, acc.Id)
	if err != nil {
		t.Fatalf("GetAccount failed: %v", err)
	}

	if got.Id != acc.Id {
		t.Errorf("expected ID %d, got %d", acc.Id, got.Id)
	}
	if got.Username != acc.Username {
		t.Errorf("expected username %s, got %s", acc.Username, got.Username)
	}
}

func TestBunClient_ListAccounts(t *testing.T) {
	cfg := config.Config{Database: config.ConfigDatabase{Type: "sqlite", Dsn: ":memory:"}}
	logger := log.New(io.Discard, "", 0)

	client, err := bun.NewBunClient(cfg, logger)
	if err != nil {
		t.Fatalf("NewBunClient failed: %v", err)
	}
	defer func() { _ = client.Close(context.Background()) }()

	ctx := context.Background()

	// Create multiple accounts
	acc1, err := client.CreateAccount(ctx, "user1", "host1.com", 22, "ssh", "")
	if err != nil {
		t.Fatalf("CreateAccount 1 failed: %v", err)
	}

	acc2, err := client.CreateAccount(ctx, "user2", "host2.com", 2222, "ssh", "")
	if err != nil {
		t.Fatalf("CreateAccount 2 failed: %v", err)
	}

	// List accounts
	accounts, err := client.ListAccounts(ctx)
	if err != nil {
		t.Fatalf("ListAccounts failed: %v", err)
	}

	if len(accounts) < 2 {
		t.Fatalf("expected at least 2 accounts, got %d", len(accounts))
	}

	// Verify created accounts are in list
	found1, found2 := false, false
	for _, a := range accounts {
		if a.Id == acc1.Id {
			found1 = true
		}
		if a.Id == acc2.Id {
			found2 = true
		}
	}

	if !found1 || !found2 {
		t.Fatal("created accounts not found in list")
	}
}

func TestBunClient_UpdateAccount(t *testing.T) {
	cfg := config.Config{Database: config.ConfigDatabase{Type: "sqlite", Dsn: ":memory:"}}
	logger := log.New(io.Discard, "", 0)

	client, err := bun.NewBunClient(cfg, logger)
	if err != nil {
		t.Fatalf("NewBunClient failed: %v", err)
	}
	defer func() { _ = client.Close(context.Background()) }()

	ctx := context.Background()

	// Create account
	acc, err := client.CreateAccount(ctx, "deploy", "example.com", 22, "ssh", "")
	if err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}

	// Update account (change host/port)
	updated, err := client.UpdateAccount(ctx, acc.Id, "deploy", "newhost.com", 2222, "ssh", "newsecret")
	if err != nil {
		t.Fatalf("UpdateAccount failed: %v", err)
	}

	if updated.Host != "newhost.com" {
		t.Errorf("expected host 'newhost.com', got %s", updated.Host)
	}
	if updated.Port != 2222 {
		t.Errorf("expected port 2222, got %d", updated.Port)
	}

	// Verify update persisted
	got, err := client.GetAccount(ctx, acc.Id)
	if err != nil {
		t.Fatalf("GetAccount after update failed: %v", err)
	}

	if got.Host != "newhost.com" {
		t.Errorf("persisted host: expected 'newhost.com', got %s", got.Host)
	}
	if got.Port != 2222 {
		t.Errorf("persisted port: expected 2222, got %d", got.Port)
	}
}

func TestBunClient_DeleteAccount(t *testing.T) {
	cfg := config.Config{Database: config.ConfigDatabase{Type: "sqlite", Dsn: ":memory:"}}
	logger := log.New(io.Discard, "", 0)

	client, err := bun.NewBunClient(cfg, logger)
	if err != nil {
		t.Fatalf("NewBunClient failed: %v", err)
	}
	defer func() { _ = client.Close(context.Background()) }()

	ctx := context.Background()

	// Create account
	acc, err := client.CreateAccount(ctx, "deploy", "example.com", 22, "ssh", "")
	if err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}

	// Delete account
	err = client.DeleteAccounts(ctx, acc.Id)
	if err != nil {
		t.Fatalf("DeleteAccounts failed: %v", err)
	}

	// Verify deletion
	_, err = client.GetAccount(ctx, acc.Id)
	if err == nil {
		t.Fatal("expected error when getting deleted account")
	}
}
