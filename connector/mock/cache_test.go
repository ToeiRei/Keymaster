// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package mock

import "testing"

func TestNewCache_RoundTrip(t *testing.T) {
	c := &Connector{}

	blank, err := c.NewCache("")
	if err != nil {
		t.Fatalf("NewCache(\"\"): %v", err)
	}
	if blank.(*Cache).Hash != "" {
		t.Fatal("expected an empty raw cache to parse to a zero cache")
	}

	raw, err := (&Cache{"abc123"}).Serialize()
	if err != nil {
		t.Fatalf("Serialize: %v", err)
	}
	parsed, err := c.NewCache(raw)
	if err != nil {
		t.Fatalf("NewCache(%q): %v", raw, err)
	}
	if parsed.(*Cache).Hash != "abc123" {
		t.Fatalf("expected hash to survive the round trip through %q", raw)
	}
}
