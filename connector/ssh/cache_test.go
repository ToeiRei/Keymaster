// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package ssh

import (
	"context"
	"testing"

	"github.com/toeirei/keymaster/connector"
)

func TestNewCache_RoundTrip(t *testing.T) {
	c := &Connector{}

	blank, err := c.ParseCache("")
	if err != nil {
		t.Fatalf("NewCache(\"\"): %v", err)
	}
	if *blank.(*Cache) != (Cache{}) {
		t.Fatalf("expected an empty raw cache to parse to a zero cache, got %+v", blank)
	}

	raw, err := (&Cache{"abc123", "ssh-ed25519 AAAAknownhost"}).Serialize()
	if err != nil {
		t.Fatalf("Serialize: %v", err)
	}
	parsed, err := c.ParseCache(raw)
	if err != nil {
		t.Fatalf("NewCache(%q): %v", raw, err)
	}
	if cache := *parsed.(*Cache); cache != (Cache{"abc123", "ssh-ed25519 AAAAknownhost"}) {
		t.Fatalf("cache did not survive the round trip through %q: %+v", raw, cache)
	}

	if _, err := c.ParseCache("not json"); err == nil {
		t.Fatal("expected NewCache to reject a non-JSON raw value")
	}
}

func TestVerifyOffline_IgnoresKnownHost(t *testing.T) {
	secret, err := generateTestPrivateKeyPEM()
	if err != nil {
		t.Fatalf("generate test private key: %v", err)
	}

	c := &Connector{}
	records := []connector.DeployRecord{{
		Algorithm: "ssh-ed25519",
		Data:      "AAAAC3NzaC1lZDI1NTE5AAAAIexample",
		Comment:   "alice@example",
	}}
	deployData := connector.DeployData{records, &Secret{PrivateKey: secret}, nil, 7}

	if ok, err := c.VerifyOffline(context.Background(), deployData); err != nil || ok {
		t.Fatalf("expected VerifyOffline to be false with no cache, got ok=%v err=%v", ok, err)
	}

	// A cache holding only a known host has never seen a deploy.
	deployData.Cache = &Cache{KnownHost: "ssh-ed25519 AAAAknownhost"}
	if ok, err := c.VerifyOffline(context.Background(), deployData); err != nil || ok {
		t.Fatalf("expected VerifyOffline to be false with only a known host, got ok=%v err=%v", ok, err)
	}

	publicKey, err := publicKeyFromPrivateKey(secret, "")
	if err != nil {
		t.Fatalf("publicKeyFromPrivateKey: %v", err)
	}
	hash := c.hashAuthorizedKeys(c.makeAuthorizedKeys(7, publicKey, records))

	// The known host must not participate in the comparison either way.
	deployData.Cache = &Cache{hash, "ssh-ed25519 AAAAknownhost"}
	if ok, err := c.VerifyOffline(context.Background(), deployData); err != nil || !ok {
		t.Fatalf("expected VerifyOffline to match, got ok=%v err=%v", ok, err)
	}
	deployData.Cache = &Cache{AuthorizedKeysHash: hash}
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
