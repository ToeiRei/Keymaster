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

func TestBunClient_CreatePublicKey(t *testing.T) {
	cfg := config.Config{Database: config.ConfigDatabase{Type: "sqlite", Dsn: ":memory:"}}
	logger := log.New(io.Discard, "", 0)

	client, err := bun.NewBunClient(cfg, logger)
	if err != nil {
		t.Fatalf("NewBunClient failed: %v", err)
	}
	defer func() { _ = client.Close(context.Background()) }()

	pk, err := client.CreatePublicKey(context.Background(), "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5 test-comment", "test-comment", nil)
	if err != nil {
		t.Fatalf("CreatePublicKey failed: %v", err)
	}
	if pk.Id == 0 {
		t.Fatal("expected non-zero public key ID")
	}
	if pk.Comment != "test-comment" {
		t.Errorf("expected comment 'test-comment', got %s", pk.Comment)
	}
}

func TestBunClient_GetPublicKey(t *testing.T) {
	cfg := config.Config{Database: config.ConfigDatabase{Type: "sqlite", Dsn: ":memory:"}}
	logger := log.New(io.Discard, "", 0)

	client, err := bun.NewBunClient(cfg, logger)
	if err != nil {
		t.Fatalf("NewBunClient failed: %v", err)
	}
	defer func() { _ = client.Close(context.Background()) }()

	ctx := context.Background()

	// Create a public key
	created, err := client.CreatePublicKey(ctx, "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5 test-comment", "test-comment", nil)
	if err != nil {
		t.Fatalf("CreatePublicKey failed: %v", err)
	}

	// Get the key
	got, err := client.GetPublicKey(ctx, created.Id)
	if err != nil {
		t.Fatalf("GetPublicKey failed: %v", err)
	}

	if got.Id != created.Id {
		t.Errorf("ID mismatch: expected %d, got %d", created.Id, got.Id)
	}
	if got.Comment != created.Comment {
		t.Errorf("comment mismatch: expected %s, got %s", created.Comment, got.Comment)
	}
}

func TestBunClient_ListPublicKeys(t *testing.T) {
	cfg := config.Config{Database: config.ConfigDatabase{Type: "sqlite", Dsn: ":memory:"}}
	logger := log.New(io.Discard, "", 0)

	client, err := bun.NewBunClient(cfg, logger)
	if err != nil {
		t.Fatalf("NewBunClient failed: %v", err)
	}
	defer func() { _ = client.Close(context.Background()) }()

	ctx := context.Background()

	// Create two public keys
	_, err = client.CreatePublicKey(ctx, "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5 key1", "key1", nil)
	if err != nil {
		t.Fatalf("CreatePublicKey 1 failed: %v", err)
	}

	_, err = client.CreatePublicKey(ctx, "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5 key2", "key2", nil)
	if err != nil {
		t.Fatalf("CreatePublicKey 2 failed: %v", err)
	}

	// List keys
	keys, err := client.ListPublicKeys(ctx, "")
	if err != nil {
		t.Fatalf("ListPublicKeys failed: %v", err)
	}

	if len(keys) < 2 {
		t.Fatalf("expected at least 2 keys, got %d", len(keys))
	}
}

func TestBunClient_DeletePublicKeys(t *testing.T) {
	cfg := config.Config{Database: config.ConfigDatabase{Type: "sqlite", Dsn: ":memory:"}}
	logger := log.New(io.Discard, "", 0)

	client, err := bun.NewBunClient(cfg, logger)
	if err != nil {
		t.Fatalf("NewBunClient failed: %v", err)
	}
	defer func() { _ = client.Close(context.Background()) }()

	ctx := context.Background()

	// Create a public key
	created, err := client.CreatePublicKey(ctx, "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5 test-comment", "test-comment", nil)
	if err != nil {
		t.Fatalf("CreatePublicKey failed: %v", err)
	}

	// Delete it
	err = client.DeletePublicKeys(ctx, created.Id)
	if err != nil {
		t.Fatalf("DeletePublicKeys failed: %v", err)
	}

	// Verify deletion
	_, err = client.GetPublicKey(ctx, created.Id)
	if err == nil {
		t.Fatal("expected error when getting deleted public key")
	}
}
