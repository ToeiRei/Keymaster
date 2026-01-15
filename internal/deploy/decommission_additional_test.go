// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package deploy

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/toeirei/keymaster/internal/db"
)

func TestRemoveSelectiveKeymasterContent_RemovesExcludedKey(t *testing.T) {
	// Init in-memory DB
	if err := db.InitDB("sqlite", ":memory:"); err != nil {
		t.Fatalf("db.InitDB failed: %v", err)
	}

	// Create account and keys
	mgr := db.DefaultAccountManager()
	if mgr == nil {
		t.Fatalf("no account manager available")
	}
	acctID, err := mgr.AddAccount("u1", "h1", "lbl", "")
	if err != nil {
		t.Fatalf("AddAccount failed: %v", err)
	}
	// Create system key
	if _, err := db.CreateSystemKey("sys-pub", "sys-priv"); err != nil {
		t.Fatalf("CreateSystemKey failed: %v", err)
	}

	km := db.DefaultKeyManager()
	if km == nil {
		t.Fatalf("no key manager available")
	}
	k1, err := km.AddPublicKeyAndGetModel("ssh-ed25519", "AAAAB3...1", "k-one", false, time.Time{})
	if err != nil || k1 == nil {
		t.Fatalf("AddPublicKeyAndGetModel k1 failed: %v %v", err, k1)
	}
	k2, err := km.AddPublicKeyAndGetModel("ssh-rsa", "AAAAB3...2", "k-two", false, time.Time{})
	if err != nil || k2 == nil {
		t.Fatalf("AddPublicKeyAndGetModel k2 failed: %v %v", err, k2)
	}

	// Assign both keys to account
	if err := km.AssignKeyToAccount(k1.ID, acctID); err != nil {
		t.Fatalf("AssignKeyToAccount k1 failed: %v", err)
	}
	if err := km.AssignKeyToAccount(k2.ID, acctID); err != nil {
		t.Fatalf("AssignKeyToAccount k2 failed: %v", err)
	}

	// Generate current authorized_keys content
	content, err := GenerateKeysContent(acctID)
	if err != nil {
		t.Fatalf("GenerateKeysContent failed: %v", err)
	}

	// Prepare mock sftp client with current file
	mock := newMockSftpClient()
	buf := &bytes.Buffer{}
	buf.WriteString(content)
	mock.files[".ssh/authorized_keys"] = &mockSftpFile{Buffer: buf, path: ".ssh/authorized_keys", parent: mock}

	d := &Deployer{sftp: mock}
	var result DecommissionResult

	// Remove k2 by excluding its ID
	if err := removeSelectiveKeymasterContent(d, &result, acctID, []int{k2.ID}, false); err != nil {
		t.Fatalf("removeSelectiveKeymasterContent failed: %v", err)
	}

	// After removal, final content should have been written back via DeployAuthorizedKeys which uses Rename to place final file.
	final, ok := mock.files[".ssh/authorized_keys"]
	if !ok {
		t.Fatalf("final authorized_keys missing from mock files")
	}
	out := final.Buffer.String()
	if strings.Contains(out, "k-two") {
		t.Fatalf("expected excluded key 'k-two' to be removed, but found in final content")
	}
}

