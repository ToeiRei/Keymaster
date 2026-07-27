// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package mock

import (
	"testing"

	"github.com/toeirei/keymaster/connector"
)

func TestParseCache_RoundTrip(t *testing.T) {
	c := &Connector{}

	blank, err := c.ParseCache("")
	if err != nil {
		t.Fatalf("ParseCache(\"\"): %v", err)
	}
	if blank.(*cache).Hash != "" {
		t.Fatal("expected an empty raw cache to parse to a zero cache")
	}

	hash, err := c.hash(connector.Deployment{Secret: &secret{}})
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	raw, err := (&cache{hash}).Serialize()
	if err != nil {
		t.Fatalf("Serialize: %v", err)
	}
	parsed, err := c.ParseCache(raw)
	if err != nil {
		t.Fatalf("ParseCache(%q): %v", raw, err)
	}
	if parsed.(*cache).Hash != hash {
		t.Fatalf("expected hash to survive the round trip through %q", raw)
	}

	// Only [Connector.hash] writes this field, so anything that is not one of its
	// fingerprints is corruption rather than a stale cache.
	if _, err := c.ParseCache(`{"hash":"abc123"}`); err == nil {
		t.Fatal("expected ParseCache to reject a hash that is not a sha256 fingerprint")
	}
}
