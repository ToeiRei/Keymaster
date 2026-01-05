// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package db

import (
	"errors"
	"testing"
)

func TestMapDBError_DuplicateDetection(t *testing.T) {
	cases := map[string]bool{
		"duplicate entry":                                       true,
		"UNIQUE constraint failed: table.column":                true,
		"ERROR: duplicate key value violates unique constraint": true,
		"Error 1062: Duplicate entry 'x' for key 'PRIMARY'":     true,
		"some unrelated error":                                  false,
	}

	for msg, expectDup := range cases {
		err := MapDBError(errors.New(msg))
		if expectDup {
			if !errors.Is(err, ErrDuplicate) {
				t.Fatalf("expected ErrDuplicate for message %q, got %v", msg, err)
			}
		} else {
			if errors.Is(err, ErrDuplicate) {
				t.Fatalf("did not expect ErrDuplicate for message %q", msg)
			}
		}
	}
}

func TestMapDBError_NonDuplicatePassthrough(t *testing.T) {
	e := errors.New("some network error")
	mapped := MapDBError(e)
	if mapped == nil {
		t.Fatalf("expected non-nil error for non-duplicate input")
	}
	if errors.Is(mapped, ErrDuplicate) {
		t.Fatalf("did not expect ErrDuplicate for non-duplicate error")
	}
	if mapped.Error() != e.Error() {
		t.Fatalf("expected original error to be returned unchanged, got: %v", mapped)
	}
}
