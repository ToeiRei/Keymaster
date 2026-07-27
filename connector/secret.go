// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package connector

import "fmt"

// Secret holds a connector's credential material.
type Secret interface {
	// Serialize serializes the secret for persistence, to be parsed back by [Connector.ParseSecret].
	Serialize() (string, error)

	// Fields provides the secrets current values as editable fields,
	// so clients can render them without knowing the connector specific shape.
	Fields() []SecretField

	// String formats the secret with its credential material redacted.
	fmt.Stringer
}

// SecretField describes one editable component of a secret, keyed for [Connector.ParseSecretFromFields].
type SecretField struct {
	Key   string
	Value string

	// field decoration/presentation

	Label     fmt.Stringer
	Multiline bool
	Masked    bool
}
