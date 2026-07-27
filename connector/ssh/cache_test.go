// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package ssh

import (
	"context"
	"strings"
	"testing"

	"github.com/toeirei/keymaster/connector"
)

func TestParseCache_RoundTrip(t *testing.T) {
	c := &Connector{}

	blank, err := c.ParseCache("")
	if err != nil {
		t.Fatalf("ParseCache(\"\"): %v", err)
	}
	if *blank.(*cache) != (cache{}) {
		t.Fatalf("expected an empty raw cache to parse to a zero cache, got %+v", blank)
	}

	want := cache{c.hashAuthorizedKeys("authorized_keys"), marshalKnownHost(testHostKey(t))}
	raw, err := want.Serialize()
	if err != nil {
		t.Fatalf("Serialize: %v", err)
	}
	parsed, err := c.ParseCache(raw)
	if err != nil {
		t.Fatalf("ParseCache(%q): %v", raw, err)
	}
	if got := *parsed.(*cache); got != want {
		t.Fatalf("cache did not survive the round trip through %q: %+v", raw, got)
	}

	if _, err := c.ParseCache("not json"); err == nil {
		t.Fatal("expected ParseCache to reject a non-JSON raw value")
	}
}

// TestParseCache_Validates covers what a stored cache is not allowed to hold: a
// fingerprint hashAuthorizedKeys could not have written, and a host key that
// resolveKnownHost would never match against a presented one.
func TestParseCache_Validates(t *testing.T) {
	c := &Connector{}
	knownHost := marshalKnownHost(testHostKey(t))

	for name, raw := range map[string]string{
		"short hash":              `{"authorized_keys_hash":"abc123"}`,
		"uppercase hash":          `{"authorized_keys_hash":"` + strings.ToUpper(c.hashAuthorizedKeys("x")) + `"}`,
		"non-hex hash":            `{"authorized_keys_hash":"` + strings.Repeat("z", 64) + `"}`,
		"unparsable known host":   `{"known_host":"ssh-ed25519 AAAAknownhost"}`,
		"known host with comment": `{"known_host":"` + knownHost + ` alice@example"}`,
		"known host with options": `{"known_host":"no-pty ` + knownHost + `"}`,
	} {
		if _, err := c.ParseCache(raw); err == nil {
			t.Errorf("expected ParseCache to reject %s", name)
		}
	}
}

func TestVerifyOffline_IgnoresKnownHost(t *testing.T) {
	privateKey, err := generateTestPrivateKeyPEM()
	if err != nil {
		t.Fatalf("generate test private key: %v", err)
	}

	publicKey, _, err := publicKeyFromPrivateKey(privateKey, "")
	if err != nil {
		t.Fatalf("publicKeyFromPrivateKey: %v", err)
	}

	c := &Connector{}
	records := []connector.DeployRecord{{
		Algorithm: "ssh-ed25519",
		Data:      "AAAAC3NzaC1lZDI1NTE5AAAAIexample",
		Comment:   "alice@example",
	}}
	deployData := connector.DeployData{records, &secret{privateKey, "", false, publicKey}, nil, 7}

	if ok, err := c.VerifyOffline(context.Background(), deployData); err != nil || ok {
		t.Fatalf("expected VerifyOffline to be false with no cache, got ok=%v err=%v", ok, err)
	}

	// A cache holding only a known host has never seen a deploy.
	deployData.Cache = &cache{KnownHost: "ssh-ed25519 AAAAknownhost"}
	if ok, err := c.VerifyOffline(context.Background(), deployData); err != nil || ok {
		t.Fatalf("expected VerifyOffline to be false with only a known host, got ok=%v err=%v", ok, err)
	}

	hash := c.hashAuthorizedKeys(c.makeAuthorizedKeys(7, publicKey, records))

	// The known host must not participate in the comparison either way.
	deployData.Cache = &cache{hash, "ssh-ed25519 AAAAknownhost"}
	if ok, err := c.VerifyOffline(context.Background(), deployData); err != nil || !ok {
		t.Fatalf("expected VerifyOffline to match, got ok=%v err=%v", ok, err)
	}
	deployData.Cache = &cache{AuthorizedKeysHash: hash}
	if ok, err := c.VerifyOffline(context.Background(), deployData); err != nil || !ok {
		t.Fatalf("expected VerifyOffline to match without a known host, got ok=%v err=%v", ok, err)
	}
}

func TestCacheOf_RejectsForeignCache(t *testing.T) {
	if _, err := cacheOf(connector.DeployData{Cache: foreignCache{}}); err == nil {
		t.Fatal("expected cacheOf to reject a cache from another connector")
	}
}

type foreignCache struct{}

func (foreignCache) Serialize() (string, error) { return "", nil }
