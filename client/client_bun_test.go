package client

import (
	"context"
	"io"
	"log"
	"testing"

	"github.com/toeirei/keymaster/config"
	"github.com/toeirei/keymaster/ui/tui/util"
)

func TestBunClient_CreatePublicKey(t *testing.T) {
	cfg := config.Config{Database: config.ConfigDatabase{Type: "sqlite", Dsn: ":memory:"}}
	logger := log.New(io.Discard, "", 0)

	c, err := NewBunClient(cfg, logger)
	if err != nil {
		t.Fatalf("NewBunClient failed: %v", err)
	}
	defer func() { _ = c.Close(context.Background()) }()

	pk, err := c.CreatePublicKey(context.Background(), "test-key", util.NewPointer("some comment"), []string{"tag1"})
	if err != nil {
		t.Fatalf("CreatePublicKey failed: %v", err)
	}
	if pk.Id == 0 {
		t.Fatalf("expected non-zero id")
	}
	if pk.Data != "test-key" {
		t.Fatalf("unexpected identity: %s", pk.Data)
	}
	if len(pk.Tags) != 1 || pk.Tags[0] != "tag1" {
		t.Fatalf("unexpected tags: %#v", pk.Tags)
	}
}
