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

// TestBunClient_DeployAccount_NotYetImplemented verifies that deploy operations
// are properly stubbed (TODO for phase 2).
func TestBunClient_DeployAccount_NotYetImplemented(t *testing.T) {
	cfg := config.Config{Database: config.ConfigDatabase{Type: "sqlite", Dsn: ":memory:"}}
	logger := log.New(io.Discard, "", 0)

	client, err := bun.NewBunClient(cfg, logger)
	if err != nil {
		t.Fatalf("NewBunClient failed: %v", err)
	}
	defer func() { _ = client.Close(context.Background()) }()

	ctx := context.Background()

	// Create an account to deploy
	acc, err := client.CreateAccount(ctx, "deploy", "example.com", 22, "ssh", "")
	if err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}

	// DeployAccount should return "not yet implemented" error
	ch, err := client.DeployAccount(ctx, acc.Id)
	if err == nil {
		t.Fatal("expected error for unimplemented DeployAccount")
	}
	if ch != nil {
		t.Error("expected nil channel for failed DeployAccount")
	}
}
