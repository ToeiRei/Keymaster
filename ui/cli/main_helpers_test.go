// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package cli

import (
	"os"
	"runtime/debug"
	"testing"

	"github.com/spf13/cobra"
)

func TestResolveBuildVersion_WithBuildInfo(t *testing.T) {
	info := &debug.BuildInfo{
		Main: debug.Module{Path: "github.com/toeirei/keymaster", Version: "v1.2.3"},
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "deadbeef"},
			{Key: "vcs.time", Value: "2025-01-01T00:00:00Z"},
		},
	}

	v, c, d := resolveBuildVersion(info)
	if v != "v1.2.3" {
		t.Fatalf("expected version v1.2.3, got %s", v)
	}
	if c != "deadbeef" {
		t.Fatalf("expected commit deadbeef, got %s", c)
	}
	if d != "2025-01-01T00:00:00Z" {
		t.Fatalf("expected date set, got %s", d)
	}
}

func TestApplyDefaultFlags_AddsFlags(t *testing.T) {
	cmd := &cobra.Command{}
	applyDefaultFlags(cmd)

	if cmd.Flags().Lookup("database.type") == nil {
		t.Fatalf("database.type flag not present")
	}
	if cmd.Flags().Lookup("database.dsn") == nil {
		t.Fatalf("database.dsn flag not present")
	}
}

func TestGetConfigPathFromCli_FlagNotSet(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("config", "", "config file")

	p, err := getConfigPathFromCli(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p != nil {
		t.Fatalf("expected nil path when flag not set, got %v", *p)
	}
}

func TestGetConfigPathFromCli_WithValidFile(t *testing.T) {
	file, err := os.CreateTemp("", "kmcfg-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer func() { _ = os.Remove(file.Name()) }()
	_ = file.Close()

	cmd := &cobra.Command{}
	cmd.Flags().String("config", "", "config file")
	// Simulate user setting the flag
	if err := cmd.Flags().Set("config", file.Name()); err != nil {
		t.Fatalf("failed to set flag: %v", err)
	}

	p, err := getConfigPathFromCli(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil || *p != file.Name() {
		t.Fatalf("expected path %s, got %v", file.Name(), p)
	}
}
