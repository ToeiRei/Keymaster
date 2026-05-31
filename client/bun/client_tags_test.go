// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package bun_test

import (
	"context"
	"io"
	"log"
	"testing"
	"time"

	"github.com/toeirei/keymaster/client/bun"
	"github.com/toeirei/keymaster/config"
)

// TestBunClient_CreateLink_NotYetImplemented verifies that link operations
// are properly stubbed (TODO for phase 2).
func TestBunClient_CreateLink_NotYetImplemented(t *testing.T) {
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

	// CreateLink should return "not yet implemented" error
	link, err := client.CreateLink(ctx, acc.Id, "tag:value", time.Now().Add(24*time.Hour))
	if err == nil {
		t.Fatal("expected error for unimplemented CreateLink")
	}
	if link.Id != 0 {
		t.Error("expected zero link ID for failed CreateLink")
	}
}
