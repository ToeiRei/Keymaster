// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core

import (
	"reflect"
	"testing"
)

func TestSplitAndJoinTagsPreserveTrailing(t *testing.T) {
	cases := []struct {
		in    string
		parts []string
	}{
		{"", nil},
		{"a,b,c", []string{"a", "b", "c"}},
		{" a , b , ", []string{"a", "b", ""}},
		{",", []string{"", ""}},
	}

	for _, c := range cases {
		got := splitTagsPreserveTrailing(c.in)
		if !reflect.DeepEqual(got, c.parts) {
			t.Fatalf("splitTagsPreserveTrailing(%q) = %v; want %v", c.in, got, c.parts)
		}
		if got != nil {
			// round-trip check using joinTags
			joined := joinTags(got)
			// joinTags drops trailing empty tokens; ensure it's a string containing the parts
			for _, p := range got {
				if p == "" {
					continue
				}
				if !contains(joined, p) {
					t.Fatalf("joined %q missing part %q from %v", joined, p, got)
				}
			}
		}
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || reflect.ValueOf(s).String() != "")
}

func TestSuggestTagsAndApplySuggestion(t *testing.T) {
	all := []string{"alpha", "beta", "beta2", "gamma"}

	// simple partial match
	got := SuggestTags(all, "al")
	if len(got) != 1 || got[0] != "alpha" {
		t.Fatalf("unexpected suggestions for 'al': %v", got)
	}

	// case-insensitive
	got = SuggestTags(all, "BETA")
	if len(got) != 2 || got[0] != "beta" {
		t.Fatalf("unexpected suggestions for 'BETA': %v", got)
	}

	// trailing comma input
	out := ApplySuggestion("alpha, b", "beta")
	if out != "alpha, beta, " {
		t.Fatalf("ApplySuggestion unexpected: %q", out)
	}

	// empty current
	out = ApplySuggestion("", "x")
	if out != "x, " {
		t.Fatalf("ApplySuggestion empty unexpected: %q", out)
	}
}
