// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package connector

import "fmt"

// Secret holds a connector's credential material. Implementations serialize to
// JSON for persistence and redact themselves when formatted.
type Secret interface {
	Serialize() (string, error)
	Fields() []SecretField
	fmt.Stringer
}

// SecretField describes one editable component of a secret, with its current
// value, so a UI can render it without knowing the connector's JSON shape.
type SecretField struct {
	Key       string
	Label     fmt.Stringer
	Value     string
	Multiline bool
	Masked    bool
}
