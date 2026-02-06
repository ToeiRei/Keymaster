// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package db

import (
	"testing"
)

func TestHashAuthorizedKeysContent_Normalization(t *testing.T) {
	a := []byte("ssh-ed25519 AAAAB3Nza... comment\r\n")
	b := []byte("ssh-ed25519 AAAAB3Nza... comment\n")
	c := []byte("ssh-ed25519 AAAAB3Nza... comment \n")

	ha := HashAuthorizedKeysContent(a)
	hb := HashAuthorizedKeysContent(b)
	hc := HashAuthorizedKeysContent(c)

	if ha != hb {
		t.Fatalf("expected CRLF->LF normalization: ha(%s) != hb(%s)", ha, hb)
	}
	if ha != hc {
		t.Fatalf("expected trailing-space trimming: ha(%s) != hc(%s)", ha, hc)
	}

	// Different content yields different hash
	d := []byte("ssh-rsa AAAAB3NzaDifferentKey comment\n")
	hd := HashAuthorizedKeysContent(d)
	if ha == hd {
		t.Fatalf("expected different content to produce different hash: ha(%s) == hd(%s)", ha, hd)
	}
}
