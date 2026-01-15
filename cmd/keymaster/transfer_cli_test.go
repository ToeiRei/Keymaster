// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/toeirei/keymaster/internal/core"
	"github.com/toeirei/keymaster/internal/db"
)

type testDeployer struct{}

func (t *testDeployer) DeployAuthorizedKeys(content string) error { return nil }
func (t *testDeployer) Close()                                    {}

// TestTransferCLI_CreateAndAccept runs the `transfer create` then `transfer accept` CLI paths.
func TestTransferCLI_CreateAndAccept(t *testing.T) {
	// Use an isolated DB per test
	if err := db.InitDB("sqlite", ":memory:"); err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}

	tmpdir := t.TempDir()
	pkgFile := filepath.Join(tmpdir, "transfer.json")

	// Create an active system key so authorized_keys generation succeeds
	if _, err := db.CreateSystemKey("sys-pub-test", "sys-priv-test"); err != nil {
		t.Fatalf("CreateSystemKey failed: %v", err)
	}

	// Disable network host-key lookups during test create
	origDM := core.DefaultDeployerManager
	core.DefaultDeployerManager = nil
	defer func() { core.DefaultDeployerManager = origDM }()

	// Run create
	root := NewRootCmd()
	root.SetArgs([]string{"transfer", "create", "cliuser@example.local", "-o", pkgFile})
	if err := root.Execute(); err != nil {
		t.Fatalf("transfer create failed: %v", err)
	}
	// Ensure file exists
	if _, err := os.Stat(pkgFile); err != nil {
		t.Fatalf("expected transfer package file, stat failed: %v", err)
	}

	// Override bootstrap deployer to avoid network in tests
	orig := core.NewBootstrapDeployerFunc
	core.NewBootstrapDeployerFunc = func(hostname, username, privateKey, expectedHostKey string) (core.BootstrapDeployer, error) {
		return &testDeployer{}, nil
	}
	defer func() { core.NewBootstrapDeployerFunc = orig }()

	// Run accept
	root2 := NewRootCmd()
	root2.SetArgs([]string{"transfer", "accept", pkgFile})
	if err := root2.Execute(); err != nil {
		t.Fatalf("transfer accept failed: %v", err)
	}

	// Verify account created in DB
	accts, err := db.GetAllAccounts()
	if err != nil {
		t.Fatalf("GetAllAccounts failed: %v", err)
	}
	found := false
	for _, a := range accts {
		if a.Username == "cliuser" && a.Hostname == "example.local" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected account cliuser@example.local in DB")
	}
}

