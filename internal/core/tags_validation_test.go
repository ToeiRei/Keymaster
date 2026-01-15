// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core

import (
	"testing"

	"github.com/toeirei/keymaster/internal/model"
)

func TestBuildAccountsByTagAndUniqueTags(t *testing.T) {
	accounts := []model.Account{
		{ID: 1, Username: "a", Tags: ""},
		{ID: 2, Username: "b", Tags: "alpha,beta"},
		{ID: 3, Username: "c", Tags: " alpha , beta , "},
		{ID: 4, Username: "d", Tags: "gamma"},
	}

	m := BuildAccountsByTag(accounts)
	if len(m) == 0 {
		t.Fatalf("expected non-empty tag map")
	}

	if _, ok := m[untaggedLabel]; !ok {
		t.Fatalf("expected untagged bucket present")
	}
	if len(m["alpha"]) != 2 {
		t.Fatalf("expected 2 accounts for alpha, got %d", len(m["alpha"]))
	}

	tags := UniqueTags(accounts)
	// expected sorted unique tags: alpha, beta, gamma, (no tags)
	if len(tags) != 4 {
		t.Fatalf("expected 4 unique tags including untagged, got %d: %v", len(tags), tags)
	}
	if tags[0] != "alpha" || tags[1] != "beta" || tags[2] != "gamma" || tags[3] != untaggedLabel {
		t.Fatalf("unexpected tag order: %v", tags)
	}
}

// Validation already tested in validation_test.go
