// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package db

import (
	"reflect"
	"testing"
)

func TestTokenizeSearchQuery_Empty(t *testing.T) {
	if got := TokenizeSearchQuery(""); got != nil {
		t.Fatalf("expected nil for empty input, got %#v", got)
	}
}

func TestTokenizeSearchQuery_Single(t *testing.T) {
	want := []string{"foo"}
	if got := TokenizeSearchQuery("FOO"); !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected tokens: got %#v want %#v", got, want)
	}
}

func TestTokenizeSearchQuery_MultipleAndTrim(t *testing.T) {
	want := []string{"one", "two", "three"}
	if got := TokenizeSearchQuery("  One   Two Three  "); !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected tokens: got %#v want %#v", got, want)
	}
}

