// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package sshkey

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestParse_NormalLine(t *testing.T) {
	line := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC3 test-key@example.com"
	alg, key, comment, err := Parse(line)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if alg != "ssh-rsa" {
		t.Fatalf("unexpected alg: %s", alg)
	}
	if key == "" {
		t.Fatalf("empty key data")
	}
	if comment != "test-key@example.com" {
		t.Fatalf("unexpected comment: %s", comment)
	}
}

func TestParse_WithOptions(t *testing.T) {
	line := "no-agent-forwarding,command=\"echo hi\" ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIBk comment"
	alg, key, comment, err := Parse(line)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if alg != "ssh-ed25519" {
		t.Fatalf("unexpected alg: %s", alg)
	}
	if comment != "comment" {
		t.Fatalf("unexpected comment: %s", comment)
	}
	if key == "" {
		t.Fatalf("empty key data")
	}
}

func TestParse_Errors(t *testing.T) {
	if _, _, _, err := Parse(""); err == nil {
		t.Fatalf("expected error for empty line")
	}
	if _, _, _, err := Parse("just-some-text"); err == nil {
		t.Fatalf("expected error for no key type")
	}
}

func TestParseSerial_ValidAndInvalid(t *testing.T) {
	good := "# Keymaster Managed Keys (Serial: 42)"
	s, err := ParseSerial(good)
	if err != nil {
		t.Fatalf("ParseSerial failed: %v", err)
	}
	if s != 42 {
		t.Fatalf("unexpected serial: %d", s)
	}

	if _, err := ParseSerial("# Not the header"); err == nil {
		t.Fatalf("expected error for non-header line")
	}
}

func TestCheckHostKeyAlgorithm_FromTestKeys(t *testing.T) {
	// Use provided testdata public keys
	files := []struct {
		path          string
		expectWarning bool
	}{
		{"../../testdata/ssh_host_rsa_key.pub", true},
		{"../../testdata/ssh_host_ed25519_key.pub", false},
	}

	for _, f := range files {
		data, err := os.ReadFile(filepath.Clean(f.path))
		if err != nil {
			t.Fatalf("failed reading %s: %v", f.path, err)
		}
		pk, _, _, _, err := ssh.ParseAuthorizedKey(data)
		if err != nil {
			t.Fatalf("ssh.ParseAuthorizedKey failed for %s: %v", f.path, err)
		}
		warn := CheckHostKeyAlgorithm(pk)
		if f.expectWarning && warn == "" {
			t.Fatalf("expected warning for %s, got none", f.path)
		}
		if !f.expectWarning && warn != "" {
			t.Fatalf("did not expect warning for %s, got: %s", f.path, warn)
		}
	}
}
