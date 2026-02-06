// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package security

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
)

func TestSecretRedactionAndJSON(t *testing.T) {
	s := FromString("supersecret")
	if fmt.Sprintf("%v", s) != "[SECRET]" {
		t.Fatalf("unexpected fmt output: %q", fmt.Sprintf("%v", s))
	}
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	if string(b) != "\"[SECRET]\"" {
		t.Fatalf("unexpected json marshal: %s", string(b))
	}
}

func TestSecretZero(t *testing.T) {
	s := FromString("abc123")
	// Zero the underlying secret
	(&s).Zero()
	// Inspect the underlying bytes using Use to avoid creating copies.
	if err := s.Use(func(b []byte) error {
		for i := range b {
			if b[i] != 0 {
				t.Fatalf("expected zeroed byte at index %d, got %d", i, b[i])
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("s.Use failed: %v", err)
	}
}

// TestSecretBytes tests that Bytes() returns a copy of underlying bytes.
func TestSecretBytes(t *testing.T) {
	original := []byte("sensitive")
	s := Secret(original)

	// Get a copy
	copy1 := s.Bytes()

	// Verify copy matches original
	if !bytes.Equal(copy1, []byte("sensitive")) {
		t.Fatalf("copy doesn't match original: %v", copy1)
	}

	// Modify the copy and ensure original secret is not modified
	copy1[0] = 'X'
	if s[0] != 's' {
		t.Fatalf("modifying copy affected original: %v", s)
	}

	// Verify a second copy is independent of the first
	copy2 := s.Bytes()
	copy2[1] = 'Y'
	// copy1 and copy2 should be different (copy2[1] == 'Y', copy1[1] == 'e')
	if copy1[1] != 'e' || copy2[1] != 'Y' {
		t.Fatalf("copies are not independent: copy1=%v, copy2=%v", copy1, copy2)
	}
	if s[1] != 'e' {
		t.Fatalf("modifying a copy affected original")
	}
}

// TestSecretBytesEmpty tests Bytes() with empty secret.
func TestSecretBytesEmpty(t *testing.T) {
	s := Secret([]byte{})
	copy := s.Bytes()
	if len(copy) != 0 {
		t.Fatalf("expected empty copy, got %v", copy)
	}
}

// TestSecretUse tests that Use executes callback with underlying bytes without copying.
func TestSecretUse(t *testing.T) {
	s := FromString("testdata")
	callCount := 0

	err := s.Use(func(b []byte) error {
		callCount++
		if string(b) != "testdata" {
			return errors.New("unexpected byte slice content")
		}
		return nil
	})

	if err != nil {
		t.Fatalf("Use failed: %v", err)
	}
	if callCount != 1 {
		t.Fatalf("callback not called exactly once, count: %d", callCount)
	}
}

// TestSecretUseError tests that Use propagates callback errors.
func TestSecretUseError(t *testing.T) {
	s := FromString("testdata")
	testErr := errors.New("callback error")

	err := s.Use(func(b []byte) error {
		return testErr
	})

	if err != testErr {
		t.Fatalf("expected %v, got %v", testErr, err)
	}
}

// TestSecretUseModification tests that Use allows in-place modification of bytes.
func TestSecretUseModification(t *testing.T) {
	s := Secret([]byte{1, 2, 3, 4})

	err := s.Use(func(b []byte) error {
		for i := range b {
			b[i] = 0
		}
		return nil
	})

	if err != nil {
		t.Fatalf("Use failed: %v", err)
	}

	// Verify original secret was modified
	if err := s.Use(func(b []byte) error {
		for _, v := range b {
			if v != 0 {
				t.Fatalf("expected all zeros, got %v", b)
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("verification Use failed: %v", err)
	}
}

// TestSecretFormat tests that Format redacts secrets.
func TestSecretFormat(t *testing.T) {
	s := FromString("mysecretvalue")

	// Test %v formatting
	output := fmt.Sprintf("%v", s)
	if output != "[SECRET]" {
		t.Fatalf("unexpected %%v output: %q", output)
	}

	// Test %s formatting
	output = fmt.Sprintf("%s", s)
	if output != "[SECRET]" {
		t.Fatalf("unexpected %%s output: %q", output)
	}

	// Test %#v formatting
	output = fmt.Sprintf("%#v", s)
	if output != "[SECRET]" {
		t.Fatalf("unexpected %%#v output: %q", output)
	}
}

// TestSecretString tests String() method redaction.
func TestSecretString(t *testing.T) {
	s := FromString("verysecret")
	if s.String() != "[SECRET]" {
		t.Fatalf("unexpected String output: %q", s.String())
	}
}

// TestSecretMarshalText tests MarshalText redaction.
func TestSecretMarshalText(t *testing.T) {
	s := FromString("textdata")
	bytes, err := s.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText failed: %v", err)
	}
	if string(bytes) != "[SECRET]" {
		t.Fatalf("unexpected MarshalText output: %q", string(bytes))
	}
}

// TestSecretValue tests Value() implements driver.Valuer interface.
func TestSecretValue(t *testing.T) {
	s := FromString("dbvalue")
	val, err := s.Value()
	if err != nil {
		t.Fatalf("Value failed: %v", err)
	}

	// Should return raw bytes as driver.Value
	bytesVal, ok := val.([]byte)
	if !ok {
		t.Fatalf("Value didn't return []byte, got %T", val)
	}
	if !bytes.Equal(bytesVal, []byte("dbvalue")) {
		t.Fatalf("Value returned incorrect bytes: %v", bytesVal)
	}
}

// TestSecretScanBytes tests Scan with []byte input.
func TestSecretScanBytes(t *testing.T) {
	var s Secret
	input := []byte("scannedbytes")

	err := s.Scan(input)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if !bytes.Equal([]byte(s), []byte("scannedbytes")) {
		t.Fatalf("Scan didn't properly set Secret, got %v", []byte(s))
	}

	// Modify input and verify Secret is unaffected (independent copy)
	input[0] = 'X'
	if s[0] != 's' {
		t.Fatalf("Scan didn't make independent copy, original modified")
	}
}

// TestSecretScanString tests Scan with string input.
func TestSecretScanString(t *testing.T) {
	var s Secret
	input := "scannedstring"

	err := s.Scan(input)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if !bytes.Equal([]byte(s), []byte("scannedstring")) {
		t.Fatalf("Scan didn't properly set Secret from string, got %v", []byte(s))
	}
}

// TestSecretScanNil tests Scan with nil input.
func TestSecretScanNil(t *testing.T) {
	s := FromString("shouldbecleaned")
	err := s.Scan(nil)
	if err != nil {
		t.Fatalf("Scan with nil failed: %v", err)
	}

	if s != nil {
		t.Fatalf("Scan with nil should set Secret to nil, got %v", s)
	}
}

// TestSecretScanUnsupported tests Scan with unsupported type.
func TestSecretScanUnsupported(t *testing.T) {
	var s Secret
	err := s.Scan(42) // int is unsupported
	if err == nil {
		t.Fatalf("Scan should have failed with unsupported type")
	}

	// Verify error message is informative
	if !bytes.Contains([]byte(err.Error()), []byte("unsupported scan type")) {
		t.Fatalf("unexpected error message: %v", err)
	}
}

// TestSecretScanUnsupportedComplex tests Scan with other unsupported types.
func TestSecretScanUnsupportedComplex(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
	}{
		{"int", 123},
		{"float", 1.23},
		{"bool", true},
		{"struct", struct{}{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s Secret
			err := s.Scan(tt.input)
			if err == nil {
				t.Fatalf("Scan(%T) should have failed", tt.input)
			}
		})
	}
}

// TestSecretScanEmptyBytes tests Scan with empty byte slice.
func TestSecretScanEmptyBytes(t *testing.T) {
	var s Secret
	err := s.Scan([]byte{})
	if err != nil {
		t.Fatalf("Scan empty bytes failed: %v", err)
	}

	if len(s) != 0 {
		t.Fatalf("expected empty Secret, got %v", s)
	}
}

// TestSecretFromBytes tests FromBytes makes independent copy.
func TestSecretFromBytes(t *testing.T) {
	original := []byte("frombytes")
	s := FromBytes(original)

	// Verify content matches
	if !bytes.Equal([]byte(s), original) {
		t.Fatalf("FromBytes didn't copy content correctly")
	}

	// Modify original and verify Secret is unaffected
	original[0] = 'X'
	if s[0] != 'f' {
		t.Fatalf("FromBytes didn't make independent copy, original affected")
	}
}

// TestSecretFromBytesEmpty tests FromBytes with empty slice.
func TestSecretFromBytesEmpty(t *testing.T) {
	s := FromBytes([]byte{})
	if len(s) != 0 {
		t.Fatalf("FromBytes empty should create empty Secret, got %v", s)
	}
}

// TestSecretRedacted tests Redacted() method.
func TestSecretRedacted(t *testing.T) {
	s := FromString("anothersecret")
	if s.Redacted() != "[SECRET]" {
		t.Fatalf("unexpected Redacted output: %q", s.Redacted())
	}
}

// TestSecretFromString tests FromString creates Secret from string.
func TestSecretFromString(t *testing.T) {
	s := FromString("test123")
	if !bytes.Equal([]byte(s), []byte("test123")) {
		t.Fatalf("FromString didn't create correct Secret: %v", []byte(s))
	}
}

// TestSecretZeroNilSecret tests Zero on nil Secret pointer.
func TestSecretZeroNilSecret(t *testing.T) {
	var s *Secret
	// Should not panic
	s.Zero()
}

// TestSecretZeroNilValue tests Zero on Secret with nil value.
func TestSecretZeroNilValue(t *testing.T) {
	s := Secret(nil)
	(&s).Zero() // Should not panic
	if s != nil {
		t.Fatalf("Zero should leave nil Secret as nil")
	}
}

// TestSecretZeroAndVerify tests that Zero truly overwrites bytes.
func TestSecretZeroAndVerify(t *testing.T) {
	s := FromString("password123")

	// Verify not zero before
	if err := s.Use(func(b []byte) error {
		for _, v := range b {
			if v == 0 {
				return errors.New("secret already zeroed")
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("verification failed: %v", err)
	}

	// Zero it
	(&s).Zero()

	// Verify all bytes are zero
	if err := s.Use(func(b []byte) error {
		for i, v := range b {
			if v != 0 {
				return fmt.Errorf("byte %d is %d, expected 0", i, v)
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("post-zero verification failed: %v", err)
	}
}

// TestSecretScanImplementsSQLScanner tests Scan signature matches sql.Scanner interface.
func TestSecretScanImplementsSQLScanner(t *testing.T) {
	var s Secret
	var _ interface{ Scan(interface{}) error } = &s
}

// TestSecretValueImplementsSQLValuer tests Value signature matches sql/driver.Valuer interface.
func TestSecretValueImplementsSQLValuer(t *testing.T) {
	s := Secret([]byte("test"))
	var _ driver.Valuer = s
}

// TestSecretIntegration tests round-trip through SQL interfaces.
func TestSecretIntegration(t *testing.T) {
	original := FromString("integration")

	// Simulate SQL value export
	val, err := original.Value()
	if err != nil {
		t.Fatalf("Value failed: %v", err)
	}

	// Simulate SQL scan import
	var restored Secret
	err = restored.Scan(val)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	// Verify round-trip integrity
	if !bytes.Equal([]byte(original), []byte(restored)) {
		t.Fatalf("round-trip failed: %v -> %v", []byte(original), []byte(restored))
	}
}
