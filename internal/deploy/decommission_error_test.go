// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package deploy

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/toeirei/keymaster/internal/db"
)

func TestRemoveAuthorizedKeysFile_NoFile(t *testing.T) {
	mock := newMockSftpClient()
	// Simulate Stat returning a "file does not exist" error string that the
	// production code checks for.
	mock.statErr[".ssh/authorized_keys"] = errors.New("file does not exist")

	d := &Deployer{sftp: mock}
	var res DecommissionResult

	if err := removeAuthorizedKeysFile(d, &res); err != nil {
		t.Fatalf("expected no error when authorized_keys missing, got: %v", err)
	}

	// Ensure we only stat and did not attempt to remove
	for _, a := range mock.actions {
		if strings.HasPrefix(a, "remove:") {
			t.Fatalf("unexpected remove action: %s", a)
		}
	}
}

func TestRemoveSelectiveKeymasterContent_RemoveSystemKeyOnly(t *testing.T) {
	// Init in-memory DB and create system key
	if err := db.InitDB("sqlite", ":memory:"); err != nil {
		t.Fatalf("db.InitDB failed: %v", err)
	}

	mgr := db.DefaultAccountManager()
	if mgr == nil {
		t.Fatalf("no account manager available")
	}
	acctID, err := mgr.AddAccount("u2", "h2", "lbl", "")
	if err != nil {
		t.Fatalf("AddAccount failed: %v", err)
	}

	// Create system key
	if _, err := db.CreateSystemKey("sys-pub-2", "sys-priv-2"); err != nil {
		t.Fatalf("CreateSystemKey failed: %v", err)
	}

	// Add a user key so it remains after system key removal
	km := db.DefaultKeyManager()
	if km == nil {
		t.Fatalf("no key manager available")
	}
	k1, err := km.AddPublicKeyAndGetModel("ssh-ed25519", "AAAAB3...X", "k-remains", false, time.Time{})
	if err != nil || k1 == nil {
		t.Fatalf("AddPublicKeyAndGetModel failed: %v", err)
	}
	if err := km.AssignKeyToAccount(k1.ID, acctID); err != nil {
		t.Fatalf("AssignKeyToAccount failed: %v", err)
	}

	// Generate current authorized_keys content and place into mock
	content, err := GenerateKeysContent(acctID)
	if err != nil {
		t.Fatalf("GenerateKeysContent failed: %v", err)
	}

	mock := newMockSftpClient()
	buf := &bytes.Buffer{}
	buf.WriteString(content)
	mock.files[".ssh/authorized_keys"] = &mockSftpFile{Buffer: buf, path: ".ssh/authorized_keys", parent: mock}

	d := &Deployer{sftp: mock}
	var res DecommissionResult

	// Remove only the system key
	if err := removeSelectiveKeymasterContent(d, &res, acctID, nil, true); err != nil {
		t.Fatalf("removeSelectiveKeymasterContent failed: %v", err)
	}

	final, ok := mock.files[".ssh/authorized_keys"]
	if !ok {
		t.Fatalf("final authorized_keys missing from mock files")
	}
	out := final.Buffer.String()
	if strings.Contains(out, "sys-pub-2") || strings.Contains(out, "command=\"internal-sftp\"") {
		t.Fatalf("expected system key to be removed from final content, but found system key")
	}
	if !strings.Contains(out, "k-remains") {
		t.Fatalf("expected user key 'k-remains' to remain, but it was removed")
	}
}

