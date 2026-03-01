package client

import (
	"context"
	"io"
	"log"
	"testing"

	"github.com/toeirei/keymaster/config"
)

func TestBunClient_TargetsCRUD(t *testing.T) {
	cfg := config.Config{Database: config.ConfigDatabase{Type: "sqlite", Dsn: ":memory:"}}
	logger := log.New(io.Discard, "", 0)

	c, err := NewBunClient(cfg, logger)
	if err != nil {
		t.Fatalf("NewBunClient failed: %v", err)
	}
	defer func() { _ = c.Close(context.Background()) }()

	ctx := context.Background()

	// Create
	t1, err := c.CreateTarget(ctx, "example.com", 22)
	if err != nil {
		t.Fatalf("CreateTarget failed: %v", err)
	}
	if t1.id == 0 {
		t.Fatalf("expected non-zero id")
	}
	if t1.host != "example.com" {
		t.Fatalf("unexpected host: %s", t1.host)
	}
	if t1.port != 22 {
		t.Fatalf("unexpected port: %d", t1.port)
	}

	// Get
	g, err := c.GetTarget(ctx, t1.id)
	if err != nil {
		t.Fatalf("GetTarget failed: %v", err)
	}
	if g.id != t1.id || g.host != t1.host || g.port != t1.port {
		t.Fatalf("GetTarget returned mismatch: %#v vs %#v", g, t1)
	}

	// Create same host with different port -> should return same id and update port
	t2, err := c.CreateTarget(ctx, "example.com", 2222)
	if err != nil {
		t.Fatalf("CreateTarget (update) failed: %v", err)
	}
	if t2.id != t1.id {
		t.Fatalf("expected same id for same host")
	}
	if t2.port != 2222 {
		t.Fatalf("expected updated port 2222, got %d", t2.port)
	}

	// GetTargets
	list, err := c.GetTargets(ctx, t1.id)
	if err != nil {
		t.Fatalf("GetTargets failed: %v", err)
	}
	if len(list) != 1 || list[0].id != t1.id {
		t.Fatalf("GetTargets unexpected result: %#v", list)
	}

	// ListTargets
	all, err := c.ListTargets(ctx)
	if err != nil {
		t.Fatalf("ListTargets failed: %v", err)
	}
	found := false
	for _, tt := range all {
		if tt.id == t1.id && tt.host == "example.com" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("ListTargets did not include created target")
	}

	// UpdateTarget: change host
	if err := c.UpdateTarget(ctx, t1.id, Target{0, "new.example.com", 2222}); err != nil {
		t.Fatalf("UpdateTarget failed: %v", err)
	}
	updated, err := c.GetTarget(ctx, t1.id)
	if err != nil {
		t.Fatalf("GetTarget after update failed: %v", err)
	}
	if updated.host != "new.example.com" {
		t.Fatalf("UpdateTarget did not change host, got %s", updated.host)
	}

	// DeleteTargets
	if err := c.DeleteTargets(ctx, t1.id); err != nil {
		t.Fatalf("DeleteTargets failed: %v", err)
	}
	after, err := c.ListTargets(ctx)
	if err != nil {
		t.Fatalf("ListTargets after delete failed: %v", err)
	}
	for _, tt := range after {
		if tt.id == t1.id {
			t.Fatalf("target was not deleted")
		}
	}
}
