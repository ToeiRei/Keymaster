// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package ssh2

import (
	"testing"
	"time"

	"github.com/toeirei/keymaster/connector"
)

// goldenPublicKey is a fixed key so the rendered file below is reproducible.
const goldenPublicKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIB1KcHDrJx3lSAsK7yFPMLLLbKzOqCwYVJRJk7VXBYIm keymaster"

// goldenAuthorizedKeys is what makeAuthorizedKeys must produce, byte for byte.
// Every deployed target's cached fingerprint is taken over this content, so an
// unintended change here reads as fleet-wide drift and marks every account dirty.
// Update it only together with a deliberate, announced format change.
const goldenAuthorizedKeys = `# Keymaster Managed Keys
# Managed Key Fingerprint: SHA256:CkqtXfnjJi9MhlUv1/FvOFcsqjzEJnA3V1bl0sm+M7c
# Payload Hash: sha256:b832d6b7b508ab81e18fb4c78a92faaa355f82cb3b5d3f92446c2b36c7c93632
# Verify: sha256sum ~/.ssh/authorized_keys | awk '{print $1}'
command="internal-sftp",no-port-forwarding,no-x11-forwarding,no-agent-forwarding,no-pty ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIB1KcHDrJx3lSAsK7yFPMLLLbKzOqCwYVJRJk7VXBYIm keymaster do not remove


# Global User Keys
expiry-time="20301231235959Z" ssh-ed25519 AAAAglobal GLOBAL-KEY ops@example
ssh-rsa AAAAglobal2 GLOBAL-KEY audit@example

# User Keys
ssh-ed25519 AAAAlocal alice@example
ssh-rsa AAAAlocal2 bob@example
`

func goldenRecords() []connector.DeployRecord {
	expires := time.Date(2030, 12, 31, 23, 59, 59, 0, time.UTC)
	// Deliberately unsorted, and interleaving global with local, so the fixture
	// also pins the ordering makeAuthorizedKeys imposes.
	return []connector.DeployRecord{
		{Algorithm: "ssh-rsa", Data: "AAAAlocal2", Comment: "bob@example"},
		{Algorithm: "ssh-ed25519", Data: "AAAAglobal", ExpiresAt: expires, Comment: "ops@example", IsGlobal: true},
		{Algorithm: "ssh-ed25519", Data: "AAAAlocal", Comment: "alice@example"},
		{Algorithm: "ssh-rsa", Data: "AAAAglobal2", Comment: "audit@example", IsGlobal: true},
	}
}

// TestMakeAuthorizedKeys_Golden pins the rendered file. It guards the header
// block, the managed key line and its options, the global/local split, the
// expiry-time option, the sorting and the trailing newline all at once.
func TestMakeAuthorizedKeys_Golden(t *testing.T) {
	c := &Connector{}

	got := c.makeAuthorizedKeys(goldenPublicKey, goldenRecords())
	if got != goldenAuthorizedKeys {
		t.Fatalf("rendered authorized_keys changed.\n got:\n%s\nwant:\n%s", got, goldenAuthorizedKeys)
	}
}

// The payload hash in the header covers only the user keys, so re-keying the
// managed line must not move it. This is what lets a rotation be diffed.
func TestMakeAuthorizedKeys_PayloadHashIgnoresTheManagedKey(t *testing.T) {
	c := &Connector{}
	records := goldenRecords()

	payload := c.userKeyPayload(records)
	other := c.makeAuthorizedKeys("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIsomethingelse other", records)

	if want := "# Payload Hash: sha256:" + c.hashAuthorizedKeys(payload); !contains(other, want) {
		t.Fatalf("expected %q in the rendering keyed to another secret:\n%s", want, other)
	}
}

// hashAuthorizedKeys normalizes line endings and trailing whitespace so a file
// that made a round trip through a Windows editor still fingerprints the same.
func TestHashAuthorizedKeys_NormalizesLineEndingsAndTrailingSpace(t *testing.T) {
	c := &Connector{}

	lf := c.makeAuthorizedKeys(goldenPublicKey, goldenRecords())
	crlf := ""
	for _, r := range lf {
		if r == '\n' {
			crlf += "\r\n"
			continue
		}
		crlf += string(r)
	}

	if c.hashAuthorizedKeys(lf) != c.hashAuthorizedKeys(crlf) {
		t.Fatal("expected CRLF content to hash the same as LF content")
	}
	if c.hashAuthorizedKeys(lf) != c.hashAuthorizedKeys(lf+"") {
		t.Fatal("expected the hash to be stable")
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
