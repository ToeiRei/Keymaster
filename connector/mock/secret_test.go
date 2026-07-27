// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package mock

import "testing"

func TestParseSecret_RoundTrip(t *testing.T) {
	c := &Connector{}

	blank, err := c.ParseSecret("")
	if err != nil {
		t.Fatalf("ParseSecret(\"\"): %v", err)
	}
	if fields := blank.Fields(); len(fields) != 1 || fields[0].Value != "false" {
		t.Fatalf("unexpected blank fields: %+v", fields)
	}

	fromValues, err := c.ParseSecretFromValues(map[string]string{"succeed": "true"})
	if err != nil {
		t.Fatalf("ParseSecretFromValues: %v", err)
	}
	raw, err := fromValues.Serialize()
	if err != nil {
		t.Fatalf("Serialize: %v", err)
	}

	parsed, err := c.ParseSecret(raw)
	if err != nil {
		t.Fatalf("ParseSecret(%q): %v", raw, err)
	}
	if !parsed.(*secret).Succeed {
		t.Fatalf("expected succeed to survive the round trip through %q", raw)
	}

	if _, err := c.ParseSecretFromValues(map[string]string{"succeed": "maybe"}); err == nil {
		t.Fatal("expected ParseSecretFromValues to reject a non-boolean value")
	}
}
