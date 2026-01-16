// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package security

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"
)

// Secret is a thin wrapper around a byte slice intended to hold sensitive
// material (private keys, passphrases). It implements redaction helpers so
// accidental formatting, JSON marshaling, or SQL drivers do not reveal data.
type Secret []byte

// String redacts the secret for fmt.Print* convenience.
func (s Secret) String() string { return "[SECRET]" }

// Format implements fmt.Formatter to ensure `%v`, `%#v` and friends are redacted.
func (s Secret) Format(f fmt.State, c rune) {
	if _, err := io.WriteString(f, "[SECRET]"); err != nil {
		_ = err // intentionally ignore write error when formatting secrets for logs
	}
}

// Bytes returns a copy of the underlying bytes. Callers are responsible for
// zeroing sensitive copies when done.
func (s Secret) Bytes() []byte {
	out := make([]byte, len(s))
	copy(out, s)
	return out
}

// Zero overwrites the underlying byte slice with zeros.
func (s *Secret) Zero() {
	if s == nil || *s == nil {
		return
	}
	for i := range *s {
		(*s)[i] = 0
	}
}

// Use executes fn with the underlying bytes (not a copy). Prefer this when
// callers need to avoid copies; responsibility for zeroing belongs to the
// caller if they retain the slice.
func (s Secret) Use(fn func([]byte) error) error {
	return fn([]byte(s))
}

// MarshalJSON redacts secrets in JSON marshaling.
func (s Secret) MarshalJSON() ([]byte, error) { return json.Marshal("[SECRET]") }

// MarshalText redacts secrets for text encoding.
func (s Secret) MarshalText() ([]byte, error) { return []byte("[SECRET]"), nil }

// Value implements database/sql/driver.Valuer to store raw bytes as-is.
func (s Secret) Value() (driver.Value, error) { return []byte(s), nil }

// Scan implements sql.Scanner to read bytes from DB into a Secret.
func (s *Secret) Scan(src interface{}) error {
	if src == nil {
		*s = nil
		return nil
	}
	switch v := src.(type) {
	case []byte:
		tmp := make([]byte, len(v))
		copy(tmp, v)
		*s = Secret(tmp)
		return nil
	case string:
		*s = Secret([]byte(v))
		return nil
	default:
		return fmt.Errorf("unsupported scan type %T", src)
	}
}

// Helper to create a Secret from a string input (callers should zero any
// intermediate []byte they create from user input).
func FromString(in string) Secret { return Secret([]byte(in)) }

// Helper to create a Secret from bytes (it makes a copy).
func FromBytes(in []byte) Secret {
	out := make([]byte, len(in))
	copy(out, in)
	return Secret(out)
}

// Redacted returns a short human-readable placeholder useful for logs.
func (s Secret) Redacted() string { return "[SECRET]" }
