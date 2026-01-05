// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package deploy

import (
	"bytes"
	"testing"

	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/i18n"
	"github.com/toeirei/keymaster/internal/model"
)

func TestAuditAccountSerial_Errors(t *testing.T) {
	if err := db.InitDB("sqlite", ":memory:"); err != nil {
		t.Fatalf("db.InitDB failed: %v", err)
	}
	i18n.Init("en")

	mgr := db.DefaultAccountManager()
	if mgr == nil {
		t.Fatal("no account manager")
	}
	acctID, err := mgr.AddAccount("u1", "host.local", "lbl", "")
	if err != nil {
		t.Fatalf("AddAccount failed: %v", err)
	}

	// Case: Serial == 0 should return an error
	acct := model.Account{ID: acctID, Username: "u1", Hostname: "host.local", Serial: 0}
	if err := AuditAccountSerial(acct); err == nil {
		t.Fatalf("expected error for serial==0, got nil")
	}

	// Case: Serial set but no system key in DB should return an error without network
	if err := db.UpdateAccountSerial(acctID, 999); err != nil {
		t.Fatalf("UpdateAccountSerial failed: %v", err)
	}
	acct.Serial = 999
	if err := AuditAccountSerial(acct); err == nil {
		t.Fatalf("expected error for missing system key, got nil")
	}
}

func TestAuditAccountStrict_DriftDetected(t *testing.T) {
	if err := db.InitDB("sqlite", ":memory:"); err != nil {
		t.Fatalf("db.InitDB failed: %v", err)
	}
	i18n.Init("en")

	mgr := db.DefaultAccountManager()
	if mgr == nil {
		t.Fatal("no account manager")
	}
	acctID, err := mgr.AddAccount("u2", "host2.local", "lbl", "")
	if err != nil {
		t.Fatalf("AddAccount failed: %v", err)
	}

	serial, err := db.CreateSystemKey("sys-pub-audit2", "sys-priv-audit2")
	if err != nil {
		t.Fatalf("CreateSystemKey failed: %v", err)
	}
	if err := db.UpdateAccountSerial(acctID, serial); err != nil {
		t.Fatalf("UpdateAccountSerial failed: %v", err)
	}

	// Remote content differs from generated expected content
	mock := newMockSftpClient()
	mock.files[".ssh/authorized_keys"] = &mockSftpFile{Buffer: bytesFromString("some other content"), path: ".ssh/authorized_keys", parent: mock}

	orig := NewDeployerFunc
	NewDeployerFunc = func(host, user, privateKey string, passphrase []byte) (*Deployer, error) {
		return &Deployer{sftp: mock, client: nil, config: DefaultConnectionConfig()}, nil
	}
	defer func() { NewDeployerFunc = orig }()

	acct := model.Account{ID: acctID, Username: "u2", Hostname: "host2.local", Serial: serial}
	if err := AuditAccountStrict(acct); err == nil {
		t.Fatalf("expected AuditAccountStrict to detect drift, but it returned nil")
	}
}

// small helpers to avoid importing bytes package directly in test bodies
func bytesFromString(s string) *bytes.Buffer { return bytes.NewBufferString(s) }

