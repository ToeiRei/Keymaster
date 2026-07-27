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
	if fields := blank.Fields(); len(fields) != 2 || fields[0].Value != "false" || fields[1].Value != "" {
		t.Fatalf("unexpected blank fields: %+v", fields)
	}

	fromFields, err := c.ParseSecretFromFields(map[string]string{"succeed": "true", "key": "primary"})
	if err != nil {
		t.Fatalf("ParseSecretFromFields: %v", err)
	}
	raw, err := fromFields.Serialize()
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
	if got := parsed.(*secret).Key; got != "primary" {
		t.Fatalf("expected key to survive the round trip through %q, got %q", raw, got)
	}

	if _, err := c.ParseSecretFromFields(map[string]string{"succeed": "maybe"}); err == nil {
		t.Fatal("expected ParseSecretFromFields to reject a non-boolean value")
	}
}
