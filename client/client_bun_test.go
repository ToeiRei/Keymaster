package client

import (
	"context"
	"io"
	"log"
	"testing"

	"github.com/toeirei/keymaster/config"
)

func TestBunClient_CreatePublicKey(t *testing.T) {
	cfg := config.Config{Database: config.ConfigDatabase{Type: "sqlite", Dsn: ":memory:"}}
	logger := log.New(io.Discard, "", 0)

	c, err := NewBunClient(cfg, logger)
	if err != nil {
		t.Fatalf("NewBunClient failed: %v", err)
	}
	defer func() { _ = c.Close(context.Background()) }()

	pk, err := c.CreatePublicKey(context.Background(), "test-identity", []string{"tag1"})
	if err != nil {
		t.Fatalf("CreatePublicKey failed: %v", err)
	}
	if pk.id == 0 {
		t.Fatalf("expected non-zero id")
	}
	if pk.identity != "test-identity" {
		t.Fatalf("unexpected identity: %s", pk.identity)
	}
	if len(pk.tags) != 1 || pk.tags[0] != "tag1" {
		t.Fatalf("unexpected tags: %#v", pk.tags)
	}
}
