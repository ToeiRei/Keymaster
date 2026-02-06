// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package keys

import (
	"strings"
	"testing"
	"time"

	"github.com/toeirei/keymaster/core/model"
)

func TestBuildAuthorizedKeysContent_NoSystemKey(t *testing.T) {
	_, err := BuildAuthorizedKeysContent(nil, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "no active system key") {
		t.Fatalf("expected no active system key error, got: %v", err)
	}
}

func TestBuildAuthorizedKeysContent_BasicAndExpiryDedupSort(t *testing.T) {
	sys := &model.SystemKey{Serial: 7, PublicKey: "SYSKEY"}

	// global key (not expired)
	gk := model.PublicKey{ID: 1, Algorithm: "ssh-ed25519", KeyData: "GDATA", Comment: "b-comment"}
	// account key with same ID should dedupe
	ak := model.PublicKey{ID: 1, Algorithm: "ssh-ed25519", KeyData: "GDATA", Comment: "b-comment"}
	// another account key with different comment sorts earlier
	ak2 := model.PublicKey{ID: 2, Algorithm: "ssh-rsa", KeyData: "ADATA", Comment: "a-comment"}
	// expired key should be filtered out
	expired := model.PublicKey{ID: 3, Algorithm: "ssh-ed25519", KeyData: "X", Comment: "z", ExpiresAt: time.Now().Add(-24 * time.Hour)}

	out, err := BuildAuthorizedKeysContent(sys, []model.PublicKey{gk, expired}, []model.PublicKey{ak, ak2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "# Keymaster Managed Keys (Serial: 7)") {
		t.Fatalf("missing header in output: %q", out)
	}
	if !strings.Contains(out, "command=\"internal-sftp\"") {
		t.Fatalf("missing restricted system key line: %q", out)
	}

	// expired key must not appear
	if strings.Contains(out, "X") {
		t.Fatalf("expired key appeared in output: %q", out)
	}

	// Dedup: only one occurrence of GDATA
	if strings.Count(out, "GDATA") != 1 {
		t.Fatalf("expected single occurrence of GDATA, got output: %q", out)
	}

	// Sorting by comment: a-comment should come before b-comment
	idxA := strings.Index(out, "a-comment")
	idxB := strings.Index(out, "b-comment")
	if idxA == -1 || idxB == -1 || idxA > idxB {
		t.Fatalf("expected a-comment before b-comment in output: %q", out)
	}
}

func TestSSHKeyTypeToVerifyCommand(t *testing.T) {
	cases := map[string]string{
		"ssh-rsa":             "ssh-keygen -lf /etc/ssh/ssh_host_rsa_key.pub",
		"ecdsa-sha2-nistp256": "ssh-keygen -lf /etc/ssh/ssh_host_ecdsa_key.pub",
		"ssh-ed25519":         "ssh-keygen -lf /etc/ssh/ssh_host_ed25519_key.pub",
		"something-unknown":   "ssh-keygen -lf /etc/ssh/ssh_host_*_key.pub",
		"ecdsa-sha2-nistp521": "ssh-keygen -lf /etc/ssh/ssh_host_ecdsa_key.pub",
	}

	for k, want := range cases {
		if got := SSHKeyTypeToVerifyCommand(k); got != want {
			t.Fatalf("for %q expected %q got %q", k, want, got)
		}
	}
}
