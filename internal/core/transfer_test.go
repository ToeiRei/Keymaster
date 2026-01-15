// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/toeirei/keymaster/internal/model"
)

// Reuse package fakeDeployer (defined in deploy_run_test.go) for tests.

func TestBuildAndAcceptTransferPackage_Success(t *testing.T) {
	pkgBytes, err := BuildTransferPackage("alice", "example.com", "lbl", "")
	if err != nil {
		t.Fatalf("BuildTransferPackage failed: %v", err)
	}

	// Validate package JSON fields exist
	var pkgMap map[string]string
	if err := json.Unmarshal(pkgBytes, &pkgMap); err != nil {
		t.Fatalf("invalid package JSON: %v", err)
	}
	if pkgMap["magic"] != "keymaster-transfer-v1" {
		t.Fatalf("unexpected magic: %s", pkgMap["magic"])
	}

	// Prepare deps using existing fakeDeployer from deploy_run_test.go
	fd := &fakeDeployer{}
	deps := BootstrapDeps{
		AddAccount: func(u, h, l, t string) (int, error) { return 42, nil },
		AssignKey:  func(kid, aid int) error { return nil },
		GenerateKeysContent: func(accountID int) (string, error) {
			return "ssh-ed25519 AAA... test", nil
		},
		NewBootstrapDeployer: func(hostname, username string, privateKey interface{}, expectedHostKey string) (BootstrapDeployer, error) {
			return fd, nil
		},
		GetActiveSystemKey: func() (*model.SystemKey, error) { return nil, nil },
		LogAudit:           func(e BootstrapAuditEvent) error { return nil },
	}
	res, err := AcceptTransferPackage(context.Background(), pkgBytes, deps)
	if err != nil {
		t.Fatalf("AcceptTransferPackage failed: %v", err)
	}
	if res.Account.ID != 42 {
		t.Fatalf("unexpected account id: %d", res.Account.ID)
	}
	if !res.RemoteDeployed {
		t.Fatalf("expected remote deployed true")
	}
	if fd.deployed == "" {
		t.Fatalf("expected deployer to be called")
	}
}

func TestAcceptTransferPackage_CRCMismatch(t *testing.T) {
	pkgBytes, err := BuildTransferPackage("bob", "example.net", "", "")
	if err != nil {
		t.Fatalf("BuildTransferPackage failed: %v", err)
	}

	// Tamper with the package by replacing a character in payload
	s := string(pkgBytes)
	s = strings.Replace(s, "example.net", "evil.example.net", 1)

	deps := BootstrapDeps{}
	if _, err := AcceptTransferPackage(context.Background(), []byte(s), deps); err == nil {
		t.Fatalf("expected CRC mismatch error, got nil")
	}
}
