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

// TestBunClient_AccountHostPort verifies that accounts properly encode/decode
// host:port information from the model.Account.Hostname field.
func TestBunClient_AccountHostPort(t *testing.T) {
	cfg := config.Config{Database: config.ConfigDatabase{Type: "sqlite", Dsn: ":memory:"}}
	logger := log.New(io.Discard, "", 0)

	client, err := bun.NewBunClient(cfg, logger)
	if err != nil {
		t.Fatalf("NewBunClient failed: %v", err)
	}
	defer func() { _ = client.Close(context.Background()) }()

	ctx := context.Background()

	// Create account with non-standard port
	acc, err := client.CreateAccount(ctx, "deploy", "remote.server.com", 2222, "ssh", "secret")
	if err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}

	// Verify host and port are correctly encoded
	if acc.Host != "remote.server.com" {
		t.Errorf("expected host 'remote.server.com', got %s", acc.Host)
	}
	if acc.Port != 2222 {
		t.Errorf("expected port 2222, got %d", acc.Port)
	}

	// Retrieve and verify the encoding persisted
	retrieved, err := client.GetAccount(ctx, acc.Id)
	if err != nil {
		t.Fatalf("GetAccount failed: %v", err)
	}

	if retrieved.Host != "remote.server.com" {
		t.Errorf("persisted host mismatch: expected 'remote.server.com', got %s", retrieved.Host)
	}
	if retrieved.Port != 2222 {
		t.Errorf("persisted port mismatch: expected 2222, got %d", retrieved.Port)
	}
}
